package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWorkflowStateFromPath_MissingReturnsDefault(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	state, err := loadWorkflowStateFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, workflowStateVersion, state.Version)
	assert.Empty(t, state.Nodes)
	assert.Empty(t, state.Connections)
	assert.Empty(t, state.Logs)
}

func TestSaveWorkflowStateToPath_RoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	original := workflowState{
		Nodes: []workflowNodeState{{
			ID:     "planner",
			X:      "100px",
			Y:      "50px",
			Status: "awaiting_approval",
			Output: &workflowAgentResult{Summary: "done", Output: "full output", FromAgent: "planner"},
		}},
		Connections: []workflowConnection{{From: "planner", To: "spec"}},
		Logs:        []workflowLogEntry{{Agent: "Planner", Message: "started", Timestamp: "10:00:00"}},
		Approval:    workflowApproval{PendingNodeID: "planner", LastDecision: "approved"},
	}

	require.NoError(t, saveWorkflowStateToPath(path, original))

	loaded, err := loadWorkflowStateFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, workflowStateVersion, loaded.Version)
	require.Len(t, loaded.Nodes, 1)
	assert.Equal(t, "planner", loaded.Nodes[0].ID)
	assert.Equal(t, "awaiting_approval", loaded.Nodes[0].Status)
	require.NotNil(t, loaded.Nodes[0].Output)
	assert.Equal(t, "full output", loaded.Nodes[0].Output.Output)
	assert.Equal(t, "planner", loaded.Approval.PendingNodeID)
	assert.NotEmpty(t, loaded.LastUpdated)
}

func TestLoadWorkflowStateFromPath_InvalidJSONFallsBack(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0o644))

	state, err := loadWorkflowStateFromPath(path)
	require.Error(t, err)
	assert.Equal(t, workflowStateVersion, state.Version)
	assert.Empty(t, state.Nodes)
}
