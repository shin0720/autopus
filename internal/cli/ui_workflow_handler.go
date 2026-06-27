package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func handleWorkflowCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AgentID string `json:"agentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activeAgentCancelsMu.Lock()
	cancel, ok := activeAgentCancels[req.AgentID]
	activeAgentCancelsMu.Unlock()
	if ok {
		cancel()
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleWorkflowRunning(w http.ResponseWriter, r *http.Request) {
	activeAgentCancelsMu.Lock()
	running := make([]string, 0, len(activeAgentCancels))
	for id := range activeAgentCancels {
		running = append(running, id)
	}
	activeAgentCancelsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"running": running})
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	root := getWorkspaceDir()
	dst := filepath.Join(root, filepath.Base(header.Filename))
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "path": dst})
}

func handleProviderStatus(w http.ResponseWriter, r *http.Request) {
	providers := []string{"claude", "codex", "gemini"}
	status := make(map[string]bool, len(providers))
	for _, name := range providers {
		_, err := exec.LookPath(name)
		status[name] = err == nil
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func handleProviderConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Key      string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch req.Provider {
	case "claude":
		_ = os.Setenv("ANTHROPIC_API_KEY", req.Key)
		_ = os.Setenv("CLAUDE_API_KEY", req.Key)
	case "gemini":
		_ = os.Setenv("GEMINI_API_KEY", req.Key)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "connected", "provider": req.Provider})
}
