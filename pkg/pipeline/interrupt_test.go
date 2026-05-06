package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insajin/autopus-adk/pkg/pipeline"
)

func TestHardInterruptEvidenceIncludesSignalsAndReason(t *testing.T) {
	t.Parallel()

	transition := pipeline.HardInterruptTransition{
		TaskID:          "T2",
		RunID:           "run-456",
		InterruptReason: "user interrupt",
		SIGTERMSent:     true,
		SIGKILLSent:     true,
		ActionSequence:  []string{"stop_new_dispatches", "sigterm_sent", "sigkill_sent"},
	}

	evidence := transition.Evidence()

	assert.Equal(t, pipeline.ReasonHardInterrupt, evidence.Reason)
	assert.Equal(t, "T2", evidence.TaskID)
	assert.Equal(t, "run-456", evidence.RunID)
	assert.Equal(t, "user interrupt", evidence.InterruptReason)
	assert.True(t, evidence.SIGTERMSent)
	assert.True(t, evidence.SIGKILLSent)
	assert.Equal(t, []string{"stop_new_dispatches", "sigterm_sent", "sigkill_sent"}, evidence.ActionSequence)
}
