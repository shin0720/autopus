package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insajin/autopus-adk/pkg/pipeline"
)

func TestDelegationDepth_DefaultCapBlocksAtDepthTwo(t *testing.T) {
	t.Parallel()

	ctx := pipeline.DelegationContext{
		CurrentDepth:  2,
		RequestedRole: "executor",
	}

	decision := pipeline.CheckDelegationDepth(ctx)

	assert.False(t, decision.Allowed)
	assert.True(t, decision.Blocked)
	assert.Equal(t, pipeline.ReasonDelegationDepthExceeded, decision.Reason)
	assert.Equal(t, 2, decision.Evidence.CurrentDepth)
	assert.Equal(t, 2, decision.Evidence.Cap)
	assert.Equal(t, "executor", decision.Evidence.RequestedRole)
	assert.Equal(t, pipeline.OverrideStatusNone, decision.Evidence.OverrideStatus)
}

func TestDelegationDepth_OverrideWithReasonAllowsOneMoreLayer(t *testing.T) {
	t.Parallel()

	ctx := pipeline.DelegationContext{
		CurrentDepth:            2,
		RequestedRole:           "tester",
		DelegationDepthOverride: 3,
		OverrideReason:          "large sibling SPEC set",
	}

	decision := pipeline.CheckDelegationDepth(ctx)

	assert.True(t, decision.Allowed)
	assert.False(t, decision.Blocked)
	assert.Equal(t, pipeline.ReasonDelegationDepthOverride, decision.Reason)
	assert.Equal(t, 3, decision.Evidence.Cap)
	assert.Equal(t, pipeline.OverrideStatusApplied, decision.Evidence.OverrideStatus)
	assert.Equal(t, "large sibling SPEC set", decision.Evidence.OverrideReason)
}

func TestWorkflowAuthenticityPreflight_DefaultPipelineWithoutSurfaceBlocks(t *testing.T) {
	t.Parallel()

	ctx := pipeline.DelegationContext{
		DefaultSubagentPipeline:  true,
		SubagentSurfaceAvailable: false,
	}

	result := pipeline.PreflightSubagentAuthenticity(ctx)

	assert.False(t, result.Allowed)
	assert.True(t, result.Blocked)
	assert.Equal(t, 0, result.SubagentDispatchCount)
	assert.Empty(t, result.SubagentRolesDispatched)
	assert.Equal(t, pipeline.DegradedModeAuthenticityBlocked, result.DegradedMode)
	assert.Equal(t, pipeline.ReasonWorkflowAuthenticityBlocked, result.Reason)
	assert.Contains(t, result.Blocker, "subagent surface")
	assert.Contains(t, result.Blocker, "solo")
}

func TestWorkflowAuthenticityPreflight_SoloModeReportsNoDegradation(t *testing.T) {
	t.Parallel()

	ctx := pipeline.DelegationContext{
		SoloMode: true,
	}

	result := pipeline.PreflightSubagentAuthenticity(ctx)

	assert.True(t, result.Allowed)
	assert.False(t, result.Blocked)
	assert.Equal(t, 0, result.SubagentDispatchCount)
	assert.Empty(t, result.DegradedMode)
}
