package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReclaimTerminalState is the final state of a reclaimed worktree or session.
type ReclaimTerminalState string

const (
	// @AX:NOTE: [AUTO] reclaim state strings are terminal wire values for degraded evidence and audit summaries.
	ReclaimMerged                   ReclaimTerminalState = "merged"
	ReclaimDiscarded                ReclaimTerminalState = "discarded"
	ReclaimPreservedForManualReview ReclaimTerminalState = "preserved_for_manual_review"
	ReclaimCleanupFailed            ReclaimTerminalState = "cleanup_failed"
)

const (
	ReclaimStateMerged                   = ReclaimMerged
	ReclaimStateDiscarded                = ReclaimDiscarded
	ReclaimStatePreservedForManualReview = ReclaimPreservedForManualReview
	ReclaimStateCleanupFailed            = ReclaimCleanupFailed
	ReclaimTerminalMerged                = ReclaimMerged
	ReclaimTerminalDiscarded             = ReclaimDiscarded
	ReclaimTerminalPreservedForReview    = ReclaimPreservedForManualReview
	ReclaimTerminalCleanupFailed         = ReclaimCleanupFailed
)

// @AX:ANCHOR: [AUTO] reclaim evidence transition contract used to publish exactly one terminal worktree outcome.
// @AX:REASON: Pipeline safety events and worker audit records rely on sanitized refs plus a single terminal state.
// ReclaimTransition records exactly one terminal reclaim result.
type ReclaimTransition struct {
	TaskID         string               `json:"task_id,omitempty"`
	RunID          string               `json:"run_id,omitempty"`
	BranchRef      string               `json:"branch_ref,omitempty"`
	WorktreeRef    string               `json:"worktree_ref,omitempty"`
	TerminalState  ReclaimTerminalState `json:"terminal_state"`
	ActionSequence []string             `json:"action_sequence,omitempty"`
	Reason         string               `json:"reason,omitempty"`
}

// ReclaimTerminalStates returns all valid terminal reclaim states.
func ReclaimTerminalStates() []ReclaimTerminalState {
	return []ReclaimTerminalState{
		ReclaimMerged,
		ReclaimDiscarded,
		ReclaimPreservedForManualReview,
		ReclaimCleanupFailed,
	}
}

// IsTerminal reports whether the state is one of the valid reclaim terminals.
func (s ReclaimTerminalState) IsTerminal() bool {
	switch s {
	case ReclaimMerged, ReclaimDiscarded, ReclaimPreservedForManualReview, ReclaimCleanupFailed:
		return true
	default:
		return false
	}
}

// Validate checks the transition has a valid terminal state.
func (t ReclaimTransition) Validate() error {
	return ValidateReclaimTerminalState(t.TerminalState)
}

// Evidence converts a valid reclaim transition into degraded evidence.
func (t ReclaimTransition) Evidence() (DegradedEvidence, error) {
	if err := t.Validate(); err != nil {
		return DegradedEvidence{}, err
	}
	return DegradedEvidence{
		Reason:          ReasonReclaim,
		TaskID:          sanitizeEvidenceRef(t.TaskID),
		RunID:           sanitizeEvidenceRef(t.RunID),
		BranchRef:       sanitizeEvidenceRef(t.BranchRef),
		WorktreeRef:     sanitizeEvidenceRef(t.WorktreeRef),
		ReclaimState:    t.TerminalState,
		ActionSequence:  cloneStrings(t.ActionSequence),
		InterruptReason: t.Reason,
	}, nil
}

// ValidateReclaimTerminalState verifies a single reclaim terminal state.
func ValidateReclaimTerminalState(state ReclaimTerminalState) error {
	if !state.IsTerminal() {
		return fmt.Errorf("invalid reclaim terminal state: %q", state)
	}
	return nil
}

// ValidateReclaimTransitions requires exactly one terminal reclaim transition.
func ValidateReclaimTransitions(transitions ...ReclaimTransition) error {
	if len(transitions) != 1 {
		return fmt.Errorf("expected exactly one reclaim terminal transition, got %d", len(transitions))
	}
	return transitions[0].Validate()
}

func sanitizeEvidenceRef(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Base(value)
	}
	return value
}
