package worker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/shin0720/auto-adk/pkg/worker/a2a"
	"github.com/shin0720/auto-adk/pkg/worker/adapter"
	"github.com/shin0720/auto-adk/pkg/worker/budget"
	"github.com/shin0720/auto-adk/pkg/worker/security"
	"github.com/shin0720/auto-adk/pkg/worker/stream"
)

func normalizePhasePlan(phases []Phase) []Phase {
	if len(phases) == 0 && a2a.SignedControlPlaneEnforced() {
		return nil
	}
	if len(phases) == 0 {
		return append([]Phase(nil), defaultPipelinePhases...)
	}
	return append([]Phase(nil), phases...)
}

func (pe *PipelineExecutor) phasePrompt(phase Phase, input string) (string, error) {
	if template, ok := pe.phasePromptTemplates[phase]; ok && strings.TrimSpace(template) != "" {
		return renderPhasePromptTemplate(template, input), nil
	}
	if instruction, ok := pe.phaseInstructions[phase]; ok && strings.TrimSpace(instruction) != "" {
		return fmt.Sprintf("%s\n\n%s", instruction, input), nil
	}
	if a2a.SignedControlPlaneEnforced() {
		return input, nil
	}

	switch phase {
	case PhasePlanner:
		return pe.plannerPrompt(input), nil
	case PhaseExecutor:
		return pe.executorPrompt(input), nil
	case PhaseTester:
		return pe.testerPrompt(input), nil
	case PhaseReviewer:
		return pe.reviewerPrompt(input), nil
	default:
		return "", fmt.Errorf("unsupported phase %q", phase)
	}
}

func renderPhasePromptTemplate(template, input string) string {
	if strings.Contains(template, "{{input}}") {
		return strings.ReplaceAll(template, "{{input}}", input)
	}
	return fmt.Sprintf("%s\n\n%s", template, input)
}

// runPhase spawns a single subprocess for the given phase.
func (pe *PipelineExecutor) runPhase(ctx context.Context, taskID string, phase Phase, prompt, model string) (PhaseResult, error) {
	sessionID := fmt.Sprintf("pipeline-%s-%s", taskID, phase)
	taskCfg := adapter.TaskConfig{
		TaskID:    fmt.Sprintf("%s-%s", taskID, phase),
		SessionID: sessionID,
		Prompt:    prompt,
		MCPConfig: pe.mcpConfig,
		WorkDir:   pe.workDir,
		Model:     model,
	}
	if len(pe.envVars) > 0 {
		taskCfg.EnvVars = make(map[string]string, len(pe.envVars))
		for k, v := range pe.envVars {
			taskCfg.EnvVars[k] = v
		}
	}

	cmd := pe.provider.BuildCommand(ctx, taskCfg)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return PhaseResult{}, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return PhaseResult{}, fmt.Errorf("stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return PhaseResult{}, fmt.Errorf("start subprocess: %w", err)
	}

	var emergencyStop *security.EmergencyStop
	phaseBudget := pe.phaseIterationBudget(phase)
	if phaseBudget != nil {
		emergencyStop = security.NewEmergencyStop()
		emergencyStop.SetProcess(cmd)
		defer emergencyStop.ClearProcess()
	}

	go func() {
		defer stdin.Close()
		if _, err := io.Copy(stdin, strings.NewReader(prompt)); err != nil {
			log.Printf("[pipeline] write prompt to stdin: %v", err)
		}
	}()

	result, parseErr := pe.parsePhaseStream(stdout, phase, phaseBudget, emergencyStop)

	waitErr := cmd.Wait()
	if parseErr != nil {
		if stderrStr := strings.TrimSpace(stderrBuf.String()); stderrStr != "" {
			return PhaseResult{}, fmt.Errorf("parse stream: %w\nstderr: %s", parseErr, stderrStr)
		}
		return PhaseResult{}, fmt.Errorf("parse stream: %w", parseErr)
	}
	if waitErr != nil {
		if result.Output != "" {
			return result, nil
		}
		if stderrStr := strings.TrimSpace(stderrBuf.String()); stderrStr != "" {
			return PhaseResult{}, fmt.Errorf("subprocess exit: %w\nstderr: %s", waitErr, stderrStr)
		}
		return PhaseResult{}, fmt.Errorf("subprocess exit: %w", waitErr)
	}

	return result, nil
}

// parsePhaseStream reads subprocess stdout and extracts the phase result.
// Counts tool_call and tool_use events for budget tracking.
func (pe *PipelineExecutor) parsePhaseStream(r io.Reader, phase Phase, phaseBudget *budget.IterationBudget, emergencyStop *security.EmergencyStop) (PhaseResult, error) {
	scanner := bufio.NewScanner(r)
	var result PhaseResult
	result.Phase = phase
	hasResult := false
	var counter *budget.Counter
	if phaseBudget != nil {
		counter = budget.NewCounter(*phaseBudget)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		evt, err := pe.provider.ParseEvent([]byte(line))
		if err != nil {
			log.Printf("[stream] skipping malformed line: %v", err)
			continue
		}

		// Count tool calls for budget tracking (REQ-BUDGET-05).
		if evt.Type == stream.EventToolCall || evt.Type == "tool_use" {
			result.ToolCalls++
			if counter != nil {
				state := counter.Increment()
				if state.Level == budget.LevelExhausted && emergencyStop != nil {
					log.Printf("[pipeline] phase %s budget exhausted, stopping", phase)
					_ = emergencyStop.Stop("pipeline_iteration_budget_exceeded")
					return result, fmt.Errorf("phase %s iteration budget exceeded: %d/%d", phase, state.Count, state.Budget.Limit)
				}
			}
		}

		if evt.Type == "result" {
			tr := pe.provider.ExtractResult(evt)
			result.Output = tr.Output
			result.CostUSD = tr.CostUSD
			result.DurationMS = tr.DurationMS
			result.SessionID = tr.SessionID
			hasResult = true
		}
	}
	if err := scanner.Err(); err != nil {
		return PhaseResult{}, fmt.Errorf("stream scan: %w", err)
	}

	if !hasResult {
		return PhaseResult{}, fmt.Errorf("no result event for phase %s", phase)
	}
	return result, nil
}

func (pe *PipelineExecutor) phaseIterationBudget(phase Phase) *budget.IterationBudget {
	if pe.iterationBudget == nil || pe.allocator == nil {
		return nil
	}
	phaseBudget := *pe.iterationBudget
	phaseBudget.Limit = pe.allocator.PhaseLimit(string(phase))
	if phaseBudget.Limit <= 0 {
		return nil
	}
	return &phaseBudget
}

// aggregateResults combines all phase results into a single TaskResult.
func (pe *PipelineExecutor) aggregateResults(results []PhaseResult, totalCost float64, totalDuration int64) adapter.TaskResult {
	var sb strings.Builder
	for _, r := range results {
		fmt.Fprintf(&sb, "## Phase: %s\n\n%s\n\n", r.Phase, r.Output)
	}
	return adapter.TaskResult{
		CostUSD:    totalCost,
		DurationMS: totalDuration,
		Output:     sb.String(),
	}
}

// IsContextOverflow checks whether a stream event indicates a context window overflow.
// Returns true if the event is an error containing "context window" or "token limit".
func IsContextOverflow(evt adapter.StreamEvent) bool {
	if evt.Type != "error" {
		return false
	}
	lower := strings.ToLower(string(evt.Data))
	return strings.Contains(lower, "context window") || strings.Contains(lower, "token limit")
}
