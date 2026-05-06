package pipeline

const (
	// DefaultDelegationDepthCap is the default maximum nested child-agent depth.
	DefaultDelegationDepthCap = 2

	// DegradedModeAuthenticityBlocked marks a default pipeline blocked before Phase 1.
	DegradedModeAuthenticityBlocked = "authenticity_blocked"
)

// SafetyReasonCode is a stable reason code for safety rail decisions.
type SafetyReasonCode string

const (
	// @AX:NOTE: [AUTO] safety reason strings are persisted evidence values; changing them breaks diagnostics and tests.
	ReasonDelegationDepthAllowed       SafetyReasonCode = "delegation_depth_allowed"
	ReasonDelegationDepthOverride      SafetyReasonCode = "delegation_depth_override"
	ReasonDelegationDepthExceeded      SafetyReasonCode = "delegation_depth_exceeded"
	ReasonWorkflowAuthenticityBlocked  SafetyReasonCode = "workflow_authenticity_blocked"
	ReasonWorktreeSlotCap              SafetyReasonCode = "worktree_slot_cap"
	ReasonReclaim                      SafetyReasonCode = "reclaim"
	ReasonHardInterrupt                SafetyReasonCode = "hard_interrupt"
	ReasonSubagentDispatchFailure      SafetyReasonCode = "subagent_dispatch_failure"
	ReasonProviderTimeout              SafetyReasonCode = "provider_timeout"
	ReasonFallback                     SafetyReasonCode = "fallback"
	ReasonWorktreeIsolationUnavailable SafetyReasonCode = "worktree_isolation_unavailable"
)

const (
	SafetyReasonDelegationDepthExceeded     = ReasonDelegationDepthExceeded
	SafetyReasonWorkflowAuthenticityBlocked = ReasonWorkflowAuthenticityBlocked
	SafetyReasonWorktreeSlotCap             = ReasonWorktreeSlotCap
	SafetyReasonReclaim                     = ReasonReclaim
	SafetyReasonHardInterrupt               = ReasonHardInterrupt
	SafetyReasonProviderTimeout             = ReasonProviderTimeout
)

// OverrideStatus describes whether delegation depth override metadata was used.
type OverrideStatus string

const (
	OverrideStatusNone          OverrideStatus = "none"
	OverrideStatusApplied       OverrideStatus = "applied"
	OverrideStatusMissingReason OverrideStatus = "missing_reason"
)

// SafetyDecision is the result of a safety rail check.
type SafetyDecision struct {
	Allowed  bool             `json:"allowed"`
	Blocked  bool             `json:"blocked"`
	Reason   SafetyReasonCode `json:"reason,omitempty"`
	Evidence DegradedEvidence `json:"evidence"`
}

// @AX:ANCHOR: [AUTO] shared degraded-evidence wire schema for delegation, worktree, reclaim, and hard-interrupt safety rails.
// @AX:REASON: Pipeline, worker, orchestra diagnostics, and tests depend on these JSON field names and reason-code semantics.
// DegradedEvidence is the shared evidence envelope for pipeline safety rails.
type DegradedEvidence struct {
	Reason                  SafetyReasonCode     `json:"reason"`
	CurrentDepth            int                  `json:"current_depth,omitempty"`
	Cap                     int                  `json:"cap,omitempty"`
	RequestedRole           string               `json:"requested_role,omitempty"`
	OverrideStatus          OverrideStatus       `json:"override_status,omitempty"`
	OverrideReason          string               `json:"override_reason,omitempty"`
	SubagentDispatchCount   int                  `json:"subagent_dispatch_count,omitempty"`
	SubagentRolesDispatched []string             `json:"subagent_roles_dispatched,omitempty"`
	DegradedMode            string               `json:"degraded_mode,omitempty"`
	UserFacingBlocker       string               `json:"user_facing_blocker,omitempty"`
	ActiveTaskIDs           []string             `json:"active_task_ids,omitempty"`
	QueuedTaskIDs           []string             `json:"queued_task_ids,omitempty"`
	SlotCount               int                  `json:"slot_count,omitempty"`
	QueueDiscipline         string               `json:"queue_discipline,omitempty"`
	TaskID                  string               `json:"task_id,omitempty"`
	RunID                   string               `json:"run_id,omitempty"`
	BranchRef               string               `json:"branch_ref,omitempty"`
	WorktreeRef             string               `json:"worktree_ref,omitempty"`
	ReclaimState            ReclaimTerminalState `json:"reclaim_state,omitempty"`
	ActionSequence          []string             `json:"action_sequence,omitempty"`
	InterruptReason         string               `json:"interrupt_reason,omitempty"`
	SIGTERMSent             bool                 `json:"sigterm_sent,omitempty"`
	SIGKILLSent             bool                 `json:"sigkill_sent,omitempty"`
}

// DelegationContext carries safety metadata for child-agent dispatch.
type DelegationContext struct {
	CurrentDepth             int      `json:"current_depth"`
	DepthCap                 int      `json:"depth_cap,omitempty"`
	RequestedRole            string   `json:"requested_role,omitempty"`
	DelegationDepthOverride  int      `json:"delegation_depth_override,omitempty"`
	OverrideReason           string   `json:"override_reason,omitempty"`
	SubagentDispatchCount    int      `json:"subagent_dispatch_count,omitempty"`
	SubagentRolesDispatched  []string `json:"subagent_roles_dispatched,omitempty"`
	DegradedMode             string   `json:"degraded_mode,omitempty"`
	DefaultSubagentPipeline  bool     `json:"default_subagent_pipeline,omitempty"`
	SubagentSurfaceAvailable bool     `json:"subagent_surface_available,omitempty"`
	SoloMode                 bool     `json:"solo_mode,omitempty"`
}

// AuthenticityPreflightResult reports whether default subagent pipeline can start.
type AuthenticityPreflightResult struct {
	Allowed                 bool             `json:"allowed"`
	Blocked                 bool             `json:"blocked"`
	Reason                  SafetyReasonCode `json:"reason,omitempty"`
	SubagentDispatchCount   int              `json:"subagent_dispatch_count"`
	SubagentRolesDispatched []string         `json:"subagent_roles_dispatched"`
	DegradedMode            string           `json:"degraded_mode,omitempty"`
	Blocker                 string           `json:"blocker,omitempty"`
	Evidence                DegradedEvidence `json:"evidence,omitempty"`
}

// EffectiveDepthCap returns the configured delegation cap after valid override metadata.
func (c DelegationContext) EffectiveDepthCap() int {
	cap, _ := c.effectiveDepthCap()
	return cap
}

// CheckDepth evaluates whether this context may dispatch another child.
func (c DelegationContext) CheckDepth() SafetyDecision {
	return CheckDelegationDepth(c)
}

// CheckDelegationDepth blocks child dispatch at or beyond the effective cap.
func CheckDelegationDepth(ctx DelegationContext) SafetyDecision {
	return EvaluateDelegationDepth(ctx)
}

// EvaluateDelegationDepth blocks child dispatch at or beyond the effective cap.
func EvaluateDelegationDepth(ctx DelegationContext) SafetyDecision {
	cap, status := ctx.effectiveDepthCap()
	reason := ReasonDelegationDepthAllowed
	if status == OverrideStatusApplied {
		reason = ReasonDelegationDepthOverride
	}

	evidence := DegradedEvidence{
		Reason:         reason,
		CurrentDepth:   ctx.CurrentDepth,
		Cap:            cap,
		RequestedRole:  ctx.RequestedRole,
		OverrideStatus: status,
		OverrideReason: ctx.OverrideReason,
	}

	if ctx.CurrentDepth >= cap {
		evidence.Reason = ReasonDelegationDepthExceeded
		return SafetyDecision{
			Allowed:  false,
			Blocked:  true,
			Reason:   ReasonDelegationDepthExceeded,
			Evidence: evidence,
		}
	}

	return SafetyDecision{
		Allowed:  true,
		Blocked:  false,
		Reason:   reason,
		Evidence: evidence,
	}
}

// PreflightSubagentAuthenticity blocks default pipeline runs without a dispatch surface.
func PreflightSubagentAuthenticity(ctx DelegationContext) AuthenticityPreflightResult {
	roles := cloneStrings(ctx.SubagentRolesDispatched)
	if ctx.SoloMode {
		return AuthenticityPreflightResult{
			Allowed:                 true,
			SubagentDispatchCount:   0,
			SubagentRolesDispatched: roles,
		}
	}

	if ctx.DefaultSubagentPipeline && !ctx.SubagentSurfaceAvailable {
		blocker := "workflow authenticity blocked: rerun with a working subagent surface or choose solo mode"
		evidence := DegradedEvidence{
			Reason:                  ReasonWorkflowAuthenticityBlocked,
			SubagentDispatchCount:   ctx.SubagentDispatchCount,
			SubagentRolesDispatched: roles,
			DegradedMode:            DegradedModeAuthenticityBlocked,
			UserFacingBlocker:       blocker,
		}
		return AuthenticityPreflightResult{
			Allowed:                 false,
			Blocked:                 true,
			Reason:                  ReasonWorkflowAuthenticityBlocked,
			SubagentDispatchCount:   ctx.SubagentDispatchCount,
			SubagentRolesDispatched: roles,
			DegradedMode:            DegradedModeAuthenticityBlocked,
			Blocker:                 blocker,
			Evidence:                evidence,
		}
	}

	return AuthenticityPreflightResult{
		Allowed:                 true,
		SubagentDispatchCount:   ctx.SubagentDispatchCount,
		SubagentRolesDispatched: roles,
		DegradedMode:            ctx.DegradedMode,
	}
}

// PreflightWorkflowAuthenticity is an alias for callers using workflow terminology.
func PreflightWorkflowAuthenticity(ctx DelegationContext) AuthenticityPreflightResult {
	return PreflightSubagentAuthenticity(ctx)
}

// CheckWorkflowAuthenticity is an alias for preflight-style callers.
func CheckWorkflowAuthenticity(ctx DelegationContext) AuthenticityPreflightResult {
	return PreflightSubagentAuthenticity(ctx)
}

func (c DelegationContext) effectiveDepthCap() (int, OverrideStatus) {
	cap := c.DepthCap
	if cap <= 0 {
		cap = DefaultDelegationDepthCap
	}
	if c.DelegationDepthOverride <= 0 {
		return cap, OverrideStatusNone
	}
	if c.OverrideReason == "" {
		return cap, OverrideStatusMissingReason
	}
	return c.DelegationDepthOverride, OverrideStatusApplied
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
