package cli

import (
	"encoding/json"
	"net/http"
	"sync"
)

var agentCancelMu sync.Mutex
var agentCancelMap = map[string]func(){}

func registerAgentCancel(agentID string, cancel func()) {
	agentCancelMu.Lock()
	agentCancelMap[agentID] = cancel
	agentCancelMu.Unlock()
}

func unregisterAgentCancel(agentID string) {
	agentCancelMu.Lock()
	delete(agentCancelMap, agentID)
	agentCancelMu.Unlock()
}

// handleWorkflowRunning returns the set of currently running agent IDs.
// Used by the frontend to distinguish a browser refresh from a server restart.
func handleWorkflowRunning(w http.ResponseWriter, r *http.Request) {
	agentCancelMu.Lock()
	ids := make([]string, 0, len(agentCancelMap))
	for id := range agentCancelMap {
		ids = append(ids, id)
	}
	agentCancelMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]string{"running": ids})
}

func handleWorkflowCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AgentID string `json:"agentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AgentID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	agentCancelMu.Lock()
	cancel, ok := agentCancelMap[req.AgentID]
	agentCancelMu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "not_found"})
		return
	}
	cancel()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}
