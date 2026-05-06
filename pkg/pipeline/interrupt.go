package pipeline

// @AX:ANCHOR: [AUTO] hard-interrupt evidence schema shared by worker audit, pipeline safety events, and failure diagnostics.
// @AX:REASON: SIGTERM/SIGKILL flags and action sequence names are consumed as structured safety evidence.
// HardInterruptTransition records terminal evidence for a worker interrupt.
type HardInterruptTransition struct {
	TaskID          string   `json:"task_id,omitempty"`
	RunID           string   `json:"run_id,omitempty"`
	InterruptReason string   `json:"interrupt_reason,omitempty"`
	SIGTERMSent     bool     `json:"sigterm_sent"`
	SIGKILLSent     bool     `json:"sigkill_sent"`
	ActionSequence  []string `json:"action_sequence,omitempty"`
}

// Evidence converts interrupt metadata into the shared degraded evidence shape.
func (t HardInterruptTransition) Evidence() DegradedEvidence {
	return DegradedEvidence{
		Reason:          ReasonHardInterrupt,
		TaskID:          sanitizeEvidenceRef(t.TaskID),
		RunID:           sanitizeEvidenceRef(t.RunID),
		InterruptReason: t.InterruptReason,
		SIGTERMSent:     t.SIGTERMSent,
		SIGKILLSent:     t.SIGKILLSent,
		ActionSequence:  cloneStrings(t.ActionSequence),
	}
}
