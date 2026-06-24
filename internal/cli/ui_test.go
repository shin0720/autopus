package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// chdirTemp creates a temp dir, changes into it, and restores the original cwd on cleanup.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

func TestGetWorkflowState_NoFile(t *testing.T) {
	chdirTemp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflow/state", nil)
	rr := httptest.NewRecorder()
	workflowStateHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var state WorkflowState
	if err := json.Unmarshal(rr.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if state.Nodes == nil {
		t.Error("Nodes must be non-nil empty slice when no file exists")
	}
	if len(state.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(state.Nodes))
	}
}

func TestPostWorkflowState_SavesFile(t *testing.T) {
	dir := chdirTemp(t)

	body := `{"nodes":[{"id":"planner","x":"100px","y":"50px","status":"completed","output":"done"}],"connections":[{"from":"planner","to":"spec"}],"logs":[{"agent":"Planner","msg":"finished","time":"15:00"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/workflow/state", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	workflowStateHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "saved" {
		t.Errorf("expected status=saved, got %q", resp["status"])
	}
	stateFile := filepath.Join(dir, filepath.FromSlash(workflowStatePath))
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not created at %s: %v", stateFile, err)
	}
}

func TestGetWorkflowState_LoadsSaved(t *testing.T) {
	chdirTemp(t)

	// POST to save state
	postBody := `{"projectName":"testproj","nodes":[{"id":"exec","x":"200px","y":"100px","status":"active"}],"connections":[],"logs":[]}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/workflow/state", bytes.NewBufferString(postBody))
	postRR := httptest.NewRecorder()
	workflowStateHandler(postRR, postReq)
	if postRR.Code != http.StatusOK {
		t.Fatalf("POST failed %d: %s", postRR.Code, postRR.Body.String())
	}

	// GET should return the saved state
	getReq := httptest.NewRequest(http.MethodGet, "/api/workflow/state", nil)
	getRR := httptest.NewRecorder()
	workflowStateHandler(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("GET failed %d: %s", getRR.Code, getRR.Body.String())
	}

	var state WorkflowState
	if err := json.Unmarshal(getRR.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if state.ProjectName != "testproj" {
		t.Errorf("projectName: want testproj, got %q", state.ProjectName)
	}
	if len(state.Nodes) != 1 || state.Nodes[0].ID != "exec" {
		t.Errorf("unexpected nodes: %+v", state.Nodes)
	}
	if state.Nodes[0].Status != "active" {
		t.Errorf("status: want active, got %q", state.Nodes[0].Status)
	}
}

func TestWorkflowState_LoadsLegacyNumericPositions(t *testing.T) {
	dir := chdirTemp(t)

	// Write a legacy state.json with numeric x/y (no "px" suffix, no status/output/logs)
	legacy := `{"nodes":[{"id":"planner","x":100,"y":50},{"id":"spec","x":300,"y":50}],"connections":[{"from":"planner","to":"spec"}]}`
	stateDst := filepath.Join(dir, filepath.FromSlash(workflowStatePath))
	if err := os.MkdirAll(filepath.Dir(stateDst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateDst, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/workflow/state", nil)
	rr := httptest.NewRecorder()
	workflowStateHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var state WorkflowState
	if err := json.Unmarshal(rr.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(state.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(state.Nodes))
	}
	// X and Y are preserved as raw JSON numbers (no surrounding quotes)
	if string(state.Nodes[0].X) != "100" {
		t.Errorf("node[0].X: want raw JSON 100, got %s", state.Nodes[0].X)
	}
	if string(state.Nodes[0].Y) != "50" {
		t.Errorf("node[0].Y: want raw JSON 50, got %s", state.Nodes[0].Y)
	}
}
