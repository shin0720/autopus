package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/insajin/autopus-adk/pkg/config"
	"github.com/insajin/autopus-adk/pkg/orchestra"
)

func handleWorkflowState(w http.ResponseWriter, r *http.Request) {
	root := getWorkspaceDir()
	if root == "" {
		root = uiProjectRoot
	}

	switch r.Method {
	case http.MethodGet:
		state, err := loadWorkflowState(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "workflow state load warning: %v\n", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	case http.MethodPost:
		var state workflowState
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := saveWorkflowState(root, state); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorkflowRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req workflowRunRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	agentName := workflowAgentName(req.AgentID)

	projectRoot := getWorkspaceDir()
	if projectRoot == "" {
		projectRoot = uiProjectRoot
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		_ = json.NewEncoder(w).Encode(workflowRunResponse{Status: "error", Message: "Config load failed: " + err.Error()})
		return
	}

	autoCtx := gatherAutoContextFiles(projectRoot, req.AgentID, req.Context)
	allContext := append(req.Context, autoCtx...)

	var ctxFiles strings.Builder
	for _, path := range allContext {
		data, _ := readWorkspaceFile(projectRoot, path)
		if len(data) == 0 {
			continue
		}
		ctxFiles.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", path, string(data)))
	}

	prompt := req.Prompt
	if req.Handoff != nil {
		handoffOutput := req.Handoff.Output
		const maxHandoffRunes = 8000
		if runes := []rune(handoffOutput); len(runes) > maxHandoffRunes {
			handoffOutput = string(runes[:maxHandoffRunes]) + "\n...(이하 생략)"
		}
		prompt = fmt.Sprintf(
			"%s\n\n이전 에이전트 결과 요약:\n%s\n\n이전 에이전트 전체 출력:\n%s\n\n"+
				"---\n[즉시 실행 지시]\n"+
				"이전 단계의 결과를 이어받아 당신의 역할에 맞는 작업을 즉시 실행하라.\n"+
				"결정이 필요한 항목은 스스로 최선의 선택을 내리고 이유를 ## 결정 사항 에 기록하라.\n"+
				"Edit 도구로 파일을 직접 수정하고 ## 작업 요약 형식으로 보고한 뒤 종료하라.\n"+
				"사용자에게 질문하거나 선택지를 제시하지 마라.",
			req.Prompt,
			req.Handoff.Summary,
			handoffOutput,
		)
	}

	var providers []orchestra.ProviderConfig
	var providerErrors []string
	for name, provider := range cfg.Orchestra.Providers {
		resolvedBinary, err := resolveRunnableBinary(provider.Binary)
		if err != nil {
			providerErrors = append(providerErrors, fmt.Sprintf("%s(%s 없음)", name, provider.Binary))
			continue
		}
		args := append([]string(nil), provider.Args...)
		if strings.Contains(strings.ToLower(resolvedBinary), "claude") {
			args = append(args, "--print", "--dangerously-skip-permissions")
		}
		providers = append(providers, orchestra.ProviderConfig{
			Name:          name,
			Binary:        resolvedBinary,
			Args:          args,
			PromptViaArgs: provider.PromptViaArgs,
		})
	}
	if len(req.Providers) > 0 {
		allowed := make(map[string]bool, len(req.Providers))
		for _, p := range req.Providers {
			allowed[p] = true
		}
		filtered := providers[:0]
		for _, p := range providers {
			if allowed[p.Name] {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}

	if len(providers) == 0 {
		msg := "실행 가능한 CLI provider가 없습니다."
		if len(providerErrors) > 0 {
			msg += " 바이너리 미발견: " + strings.Join(providerErrors, ", ")
		}
		uiWorkflowBroker.publish("error", req.AgentID, agentName, msg)
		_ = json.NewEncoder(w).Encode(workflowRunResponse{Status: "error", Message: msg})
		return
	}

	var checklist []checklistItem
	if devAgentIDs[req.AgentID] {
		checklist = buildSpecChecklist(projectRoot, req.Context)
	}
	if len(checklist) > 0 {
		prompt = buildDevPrompt(prompt, checklist)
		uiWorkflowBroker.publishChecklist(req.AgentID, agentName, checklist)
	}

	orchCfg := orchestra.OrchestraConfig{
		WorkingDir:     projectRoot,
		Prompt:         buildAgentPrompt(req.AgentID, agentName, prompt, ctxFiles.String()),
		Strategy:       orchestra.StrategyFastest,
		Providers:      providers,
		TimeoutSeconds: 1800,
		SubprocessMode: true,
	}

	startMsg := "분석 엔진을 가동합니다. 잠시만 기다려주세요..."
	if req.Handoff != nil {
		startMsg = "이전 결과를 바탕으로 심층 분석 중..."
	}
	uiWorkflowBroker.publish("started", req.AgentID, agentName, startMsg)

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(workflowRunResponse{Status: "accepted"})

	go runWorkflowAgent(
		context.Background(),
		orchCfg,
		req.AgentID,
		agentName,
		checklist,
		req.Context,
	)
}

func handleWorkflowEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string `json:"type"`
		AgentID string `json:"agentId"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uiWorkflowBroker.publish(req.Type, req.AgentID, workflowAgentName(req.AgentID), req.Message)
	w.WriteHeader(http.StatusNoContent)
}
