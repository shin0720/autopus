package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// setTestWorkspace sets the active workspace to a temp dir and restores it on cleanup.
func setTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	currentWorkspaceMu.Lock()
	currentWorkspaceDir = dir
	currentWorkspaceMu.Unlock()
	t.Cleanup(func() {
		currentWorkspaceMu.Lock()
		currentWorkspaceDir = ""
		currentWorkspaceMu.Unlock()
	})
	return dir
}

func TestWorkflowState_GetEmpty(t *testing.T) {
	setTestWorkspace(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflow/state", nil)
	rr := httptest.NewRecorder()
	handleWorkflowState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var state workflowState
	if err := json.Unmarshal(rr.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if state.Nodes == nil {
		t.Error("Nodes must be non-nil slice when no file exists")
	}
	if len(state.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(state.Nodes))
	}
}

func TestWorkflowState_PostAndGet(t *testing.T) {
	dir := setTestWorkspace(t)

	body := workflowState{
		Nodes:       []workflowNodeState{{ID: "exec", Status: "active"}},
		Connections: []workflowConnection{},
		Logs:        []workflowLogEntry{},
	}
	raw, _ := json.Marshal(body)
	postReq := httptest.NewRequest(http.MethodPost, "/api/workflow/state", bytes.NewReader(raw))
	postRR := httptest.NewRecorder()
	handleWorkflowState(postRR, postReq)

	if postRR.Code != http.StatusNoContent {
		t.Fatalf("POST: expected 204, got %d: %s", postRR.Code, postRR.Body.String())
	}

	// State file should exist.
	statePath := workflowStatePath(dir)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// GET should return the saved state.
	getReq := httptest.NewRequest(http.MethodGet, "/api/workflow/state", nil)
	getRR := httptest.NewRecorder()
	handleWorkflowState(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d: %s", getRR.Code, getRR.Body.String())
	}
	var got workflowState
	if err := json.Unmarshal(getRR.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "exec" {
		t.Errorf("unexpected nodes: %+v", got.Nodes)
	}
	if got.Nodes[0].Status != "active" {
		t.Errorf("status: want active, got %q", got.Nodes[0].Status)
	}
}

func TestWorkflowState_MethodNotAllowed(t *testing.T) {
	setTestWorkspace(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/workflow/state", nil)
	rr := httptest.NewRecorder()
	handleWorkflowState(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}
