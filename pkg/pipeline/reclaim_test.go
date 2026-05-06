package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insajin/autopus-adk/pkg/pipeline"
)

func TestReclaimTransition_ValidTerminalStateProducesEvidence(t *testing.T) {
	t.Parallel()

	transition := pipeline.ReclaimTransition{
		TaskID:        "T1",
		RunID:         "run-123",
		BranchRef:     "worktree/T1",
		WorktreeRef:   "/Users/person/secret/project/autopus-worktree-T1",
		TerminalState: pipeline.ReclaimPreservedForManualReview,
	}

	require.NoError(t, transition.Validate())
	evidence, err := transition.Evidence()

	require.NoError(t, err)
	assert.Equal(t, pipeline.ReasonReclaim, evidence.Reason)
	assert.Equal(t, "T1", evidence.TaskID)
	assert.Equal(t, "run-123", evidence.RunID)
	assert.Equal(t, "worktree/T1", evidence.BranchRef)
	assert.Equal(t, "autopus-worktree-T1", evidence.WorktreeRef)
	assert.Equal(t, pipeline.ReclaimPreservedForManualReview, evidence.ReclaimState)
}

func TestReclaimTransition_RequiresExactlyOneTerminalTransition(t *testing.T) {
	t.Parallel()

	valid := pipeline.ReclaimTransition{
		TaskID:        "T1",
		BranchRef:     "worktree/T1",
		WorktreeRef:   "autopus-worktree-T1",
		TerminalState: pipeline.ReclaimDiscarded,
	}

	require.NoError(t, pipeline.ValidateReclaimTransitions(valid))

	err := pipeline.ValidateReclaimTransitions()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")

	err = pipeline.ValidateReclaimTransitions(valid, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestReclaimTransition_RejectsInvalidTerminalState(t *testing.T) {
	t.Parallel()

	transition := pipeline.ReclaimTransition{
		TaskID:        "T2",
		BranchRef:     "worktree/T2",
		WorktreeRef:   "autopus-worktree-T2",
		TerminalState: pipeline.ReclaimTerminalState("kept"),
	}

	err := transition.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
}
