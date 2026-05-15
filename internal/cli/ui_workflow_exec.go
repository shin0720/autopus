package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/insajin/autopus-adk/pkg/orchestra"
)

// runWorkflowAgent executes the orchestra run in a goroutine, publishing SSE
// events through uiWorkflowBroker. It is called as go runWorkflowAgent(...)
// so the HTTP handler can return immediately.
func runWorkflowAgent(
	ctx context.Context,
	orchCfg orchestra.OrchestraConfig,
	agentID, agentName string,
	initialChecklist []checklistItem,
	ctxFilePaths []string,
) {
	cancelCtx, cancel := context.WithCancel(ctx)
	registerAgentCancel(agentID, cancel)
	defer func() {
		cancel()
		unregisterAgentCancel(agentID)
	}()

	const maxValidationRetries = 2

	var (
		cleanOutput string
		lastOutcome struct {
			result *orchestra.OrchestraResult
			err    error
		}
	)

	for attempt := 0; attempt <= maxValidationRetries; attempt++ {
		type orchOutcome struct {
			result *orchestra.OrchestraResult
			err    error
		}
		done := make(chan orchOutcome, 1)
		cfgForRun := orchCfg
		go func() {
			r, e := orchestra.RunOrchestra(cancelCtx, cfgForRun)
			done <- orchOutcome{r, e}
		}()

		ticker := time.NewTicker(30 * time.Second)
		startedAt := time.Now()
		var outcome orchOutcome
	wait:
		for {
			select {
			case outcome = <-done:
				break wait
			case <-ticker.C:
				elapsed := int(time.Since(startedAt).Minutes())
				uiWorkflowBroker.publish("working", agentID, agentName,
					fmt.Sprintf("⏳ 작업 중... (%d분 경과)", elapsed))
			case <-cancelCtx.Done():
				ticker.Stop()
				uiWorkflowBroker.publish("error", agentID, agentName, "⛔ 작업이 취소되었습니다.")
				return
			}
		}
		ticker.Stop()

		lastOutcome.result = outcome.result
		lastOutcome.err = outcome.err

		if outcome.err != nil {
			uiWorkflowBroker.publish("error", agentID, agentName, outcome.err.Error())
			return
		}

		cleanOutput = orchestra.ExtractClaudeJSONOutput(outcome.result.Merged)
		reason, ok := validateAgentOutput(agentID, cleanOutput)
		if ok {
			break
		}
		if attempt == maxValidationRetries {
			uiWorkflowBroker.publish("working", agentID, agentName,
				fmt.Sprintf("⚠️ 응답 검증 %d회 실패: %s. 결과를 그대로 표시합니다.", maxValidationRetries+1, reason))
			break
		}
		uiWorkflowBroker.publish("working", agentID, agentName,
			fmt.Sprintf("🔁 응답 검증 실패 (%s). 강화된 지시로 재시도합니다 (%d/%d)...", reason, attempt+1, maxValidationRetries))
		orchCfg.Prompt = buildRetryPrompt(orchCfg.Prompt, reason, attempt+1)
	}
	_ = lastOutcome

	if strings.TrimSpace(cleanOutput) != "" {
		uiWorkflowBroker.publish("chunk", agentID, agentName, cleanOutput)
	}

	finalChecklist := applyChecklistDone(initialChecklist, cleanOutput)
	if len(finalChecklist) > 0 {
		uiWorkflowBroker.publishChecklist(agentID, agentName, finalChecklist)
	}

	agentResult := &workflowAgentResult{
		Summary:      summarizeWorkflowOutput(cleanOutput),
		Output:       cleanOutput,
		FromAgent:    agentID,
		ContextFiles: ctxFilePaths,
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	completedMsg := "결과 생성 완료"
	approvalMsg := "승인 대기 상태로 전환합니다."
	if len(finalChecklist) > 0 && !allChecklistDone(finalChecklist) {
		remaining := 0
		for _, it := range finalChecklist {
			if !it.Done {
				remaining++
			}
		}
		completedMsg = fmt.Sprintf("결과 생성 완료 (미완료 항목 %d개 남음)", remaining)
		approvalMsg = fmt.Sprintf("⚠️ 체크리스트 %d개 항목이 미완료입니다. 결과를 확인하고 승인 여부를 결정하세요.", remaining)
	}
	uiWorkflowBroker.publish("completed", agentID, agentName, completedMsg)
	uiWorkflowBroker.publish("awaiting_approval", agentID, agentName, approvalMsg)
	uiWorkflowBroker.publishWithResult("result", agentID, agentName, "결과 준비 완료", agentResult)
}
