package worker

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/shin0720/auto-adk/pkg/worker/a2a"
	"github.com/shin0720/auto-adk/pkg/worker/adapter"
	"github.com/shin0720/auto-adk/pkg/worker/budget"
	"github.com/shin0720/auto-adk/pkg/worker/compress"
	"github.com/shin0720/auto-adk/pkg/worker/routing"
)

// Phase represents a pipeline execution phase.
type Phase string

const (
	PhasePlanner  Phase = "planner"
	PhaseExecutor Phase = "executor"
	PhaseTester   Phase = "tester"
	PhaseReviewer Phase = "reviewer"
)

// PhaseResult holds the output from a single pipeline phase.
type PhaseResult struct {
	Phase      Phase
	Output     string
	CostUSD    float64
	DurationMS int64
	SessionID  string
	ToolCalls  int // number of tool calls made during this phase
}

var defaultPipelinePhases = []Phase{
	PhasePlanner,
	PhaseExecutor,
	PhaseTester,
	PhaseReviewer,
}

// PipelineExecutor spawns separate subprocesses for each phase:
// planner -> executor(s) -> tester -> reviewer.
// Triggered when a single --print execution exceeds the context window.
type PipelineExecutor struct {
	provider             adapter.ProviderAdapter
	mcpConfig            string
	workDir              string
	envVars              map[string]string
	phaseInstructions    map[Phase]string
	phasePromptTemplates map[Phase]string
	allocator            *budget.PhaseAllocator // nil if budget not configured
	iterationBudget      *budget.IterationBudget
	compressor           compress.ContextCompressor // nil if compression not configured
	router               *routing.Router            // nil if routing not configured
}

// NewPipelineExecutor creates a new PipelineExecutor.
func NewPipelineExecutor(provider adapter.ProviderAdapter, mcpConfig, workDir string) *PipelineExecutor {
	return &PipelineExecutor{
		provider:  provider,
		mcpConfig: mcpConfig,
		workDir:   workDir,
	}
}

// SetBudget configures per-phase budget allocation for the pipeline.
func (pe *PipelineExecutor) SetBudget(total int, alloc budget.PhaseAllocation) {
	pe.allocator = budget.NewPhaseAllocator(total, alloc)
}

// SetIterationBudget configures a server-issued total iteration budget for the pipeline.
func (pe *PipelineExecutor) SetIterationBudget(iterationBudget budget.IterationBudget) {
	pe.iterationBudget = &iterationBudget
	pe.allocator = budget.NewPhaseAllocator(iterationBudget.Limit, budget.DefaultAllocation())
}

// SetCompressor configures context compression for phase transitions.
func (pe *PipelineExecutor) SetCompressor(c compress.ContextCompressor) {
	pe.compressor = c
}

// SetEnvVars configures additional environment variables for all pipeline phases.
func (pe *PipelineExecutor) SetEnvVars(envVars map[string]string) {
	if len(envVars) == 0 {
		pe.envVars = nil
		return
	}
	pe.envVars = make(map[string]string, len(envVars))
	for k, v := range envVars {
		pe.envVars[k] = v
	}
}

// SetPhaseInstructions configures server-selected instructions for pipeline phases.
func (pe *PipelineExecutor) SetPhaseInstructions(instructions map[Phase]string) {
	if len(instructions) == 0 {
		pe.phaseInstructions = nil
		return
	}
	pe.phaseInstructions = make(map[Phase]string, len(instructions))
	for phase, instruction := range instructions {
		pe.phaseInstructions[phase] = instruction
	}
}

// SetPhasePromptTemplates configures server-selected full prompt templates for pipeline phases.
func (pe *PipelineExecutor) SetPhasePromptTemplates(templates map[Phase]string) {
	if len(templates) == 0 {
		pe.phasePromptTemplates = nil
		return
	}
	pe.phasePromptTemplates = make(map[Phase]string, len(templates))
	for phase, template := range templates {
		pe.phasePromptTemplates[phase] = template
	}
}

// SetRouter configures model routing for the pipeline (REQ-ROUTE-01).
func (pe *PipelineExecutor) SetRouter(r *routing.Router) {
	pe.router = r
}

// Execute runs the full pipeline: planner → executor(s) → tester → reviewer.
// Each phase uses an independent --resume session ID.
// Returns an aggregated TaskResult combining all phase outputs.
func (pe *PipelineExecutor) Execute(ctx context.Context, taskID, prompt string) (adapter.TaskResult, error) {
	return pe.ExecuteWithPlan(ctx, taskID, prompt, "", nil)
}

// ExecuteWithPlan runs the pipeline with an optional server-selected model and
// explicit phase plan. When phases is empty, the default sequence is used.
func (pe *PipelineExecutor) ExecuteWithPlan(ctx context.Context, taskID, prompt, model string, phases []Phase) (adapter.TaskResult, error) {
	log.Printf("[pipeline] starting phase-split for task %s", taskID)

	// Resolve model once from the original prompt (REQ-ROUTE-01).
	// In signed control-plane mode, local routing fallback is disabled.
	routedModel := model
	if routedModel == "" && pe.router != nil && !a2a.SignedControlPlaneEnforced() {
		routedModel = pe.router.Route(pe.provider.Name(), prompt)
	}

	var results []PhaseResult
	var totalCost float64
	var totalDuration int64
	prevOutput := prompt
	normalizedPhases := normalizePhasePlan(phases)
	if len(normalizedPhases) == 0 {
		return adapter.TaskResult{}, fmt.Errorf("missing pipeline phase plan")
	}

	for _, phase := range normalizedPhases {
		select {
		case <-ctx.Done():
			return adapter.TaskResult{}, ctx.Err()
		default:
		}

		// Log phase budget if allocator is configured (REQ-BUDGET-09).
		if pe.allocator != nil {
			limit := pe.allocator.PhaseLimit(string(phase))
			log.Printf("[pipeline] phase %s budget: %d tool calls", phase, limit)
		}

		phasePrompt, err := pe.phasePrompt(phase, prevOutput)
		if err != nil {
			return adapter.TaskResult{}, err
		}
		pr, err := pe.runPhase(ctx, taskID, phase, phasePrompt, routedModel)
		if err != nil {
			log.Printf("[pipeline] phase %s failed for task %s: %v", phase, taskID, err)
			return adapter.TaskResult{}, fmt.Errorf("phase %s: %w", phase, err)
		}

		// Record phase completion for budget carry-over (REQ-BUDGET-10).
		if pe.allocator != nil {
			pe.allocator.CompletePhase(string(phase), pr.ToolCalls)
			log.Printf("[pipeline] phase %s used %d tool calls, remaining total: %d",
				phase, pr.ToolCalls, pe.allocator.TotalRemaining())
		}

		results = append(results, pr)
		totalCost += pr.CostUSD
		totalDuration += pr.DurationMS

		// Compress phase output before passing to next phase (REQ-COMP-001).
		if pe.compressor != nil {
			prevOutput = pe.compressor.Compress(string(phase), pr.Output, pe.provider.Name())
		} else {
			prevOutput = pr.Output
		}

		log.Printf("[pipeline] phase %s completed: cost=$%.4f duration=%dms", phase, pr.CostUSD, pr.DurationMS)
	}

	return pe.aggregateResults(results, totalCost, totalDuration), nil
}

// ParsePhasePlan validates and canonicalizes a server-provided phase plan.
func ParsePhasePlan(phases []string) ([]Phase, error) {
	if len(phases) == 0 {
		return nil, nil
	}

	plan := make([]Phase, 0, len(phases))
	for _, raw := range phases {
		phase, err := ParsePhase(raw)
		if err != nil {
			return nil, err
		}
		plan = append(plan, phase)
	}
	if len(plan) == 0 {
		return nil, nil
	}
	return plan, nil
}

// ParsePhase validates and canonicalizes a single phase name.
func ParsePhase(name string) (Phase, error) {
	switch phase := Phase(strings.ToLower(strings.TrimSpace(name))); phase {
	case PhasePlanner, PhaseExecutor, PhaseTester, PhaseReviewer:
		return phase, nil
	case "":
		return "", fmt.Errorf("empty phase name")
	default:
		return "", fmt.Errorf("unsupported phase %q", name)
	}
}

// ParsePhaseInstructions validates phase instruction overrides from the server.
func ParsePhaseInstructions(instructions map[string]string) (map[Phase]string, error) {
	if len(instructions) == 0 {
		return nil, nil
	}

	parsed := make(map[Phase]string, len(instructions))
	for rawPhase, instruction := range instructions {
		phase, err := ParsePhase(rawPhase)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(instruction) == "" {
			continue
		}
		parsed[phase] = strings.TrimSpace(instruction)
	}
	if len(parsed) == 0 {
		return nil, nil
	}
	return parsed, nil
}

// ParsePhasePromptTemplates validates server-provided full prompt templates.
func ParsePhasePromptTemplates(templates map[string]string) (map[Phase]string, error) {
	if len(templates) == 0 {
		return nil, nil
	}

	parsed := make(map[Phase]string, len(templates))
	for rawPhase, template := range templates {
		phase, err := ParsePhase(rawPhase)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(template) == "" {
			continue
		}
		parsed[phase] = strings.TrimSpace(template)
	}
	if len(parsed) == 0 {
		return nil, nil
	}
	return parsed, nil
}
