package security

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// EmergencyStop manages graceful-to-forceful subprocess termination.
type EmergencyStop struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stopped bool
}

// @AX:ANCHOR: [AUTO] hard-interrupt evidence schema emitted when emergency stop enforces iteration-budget limits.
// @AX:REASON: Worker audit records and pipeline safety events depend on stable signal flags and action sequence values.
// EmergencyEvidence records the hard-interrupt action sequence.
type EmergencyEvidence struct {
	Reason         string   `json:"reason"`
	PID            int      `json:"pid,omitempty"`
	PGID           int      `json:"pgid,omitempty"`
	SIGTERMSent    bool     `json:"sigterm_sent"`
	SIGKILLSent    bool     `json:"sigkill_sent"`
	ActionSequence []string `json:"action_sequence,omitempty"`
}

// NewEmergencyStop creates an EmergencyStop handler.
func NewEmergencyStop() *EmergencyStop {
	return &EmergencyStop{}
}

// SetProcess registers the active subprocess for emergency termination.
// The subprocess command should be configured with SysProcAttr{Setpgid: true}
// to enable process group termination.
func (e *EmergencyStop) SetProcess(cmd *exec.Cmd) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cmd = cmd
	e.stopped = false
}

// ClearProcess clears the registered subprocess after normal completion.
func (e *EmergencyStop) ClearProcess() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cmd = nil
}

// Stop terminates the subprocess: SIGTERM first, then SIGKILL after 5s.
// reason is logged for audit trail. Thread-safe — only the first call acts.
func (e *EmergencyStop) Stop(reason string) error {
	_, err := e.StopWithEvidence(reason)
	return err
}

// StopWithEvidence terminates the subprocess and returns structured interrupt evidence.
func (e *EmergencyStop) StopWithEvidence(reason string) (*EmergencyEvidence, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	evidence := &EmergencyEvidence{Reason: reason}
	if e.stopped {
		return evidence, nil
	}

	if e.cmd == nil || e.cmd.Process == nil {
		return evidence, errors.New("no process registered for emergency stop")
	}

	e.stopped = true
	pid := e.cmd.Process.Pid
	pgid := -pid // Negative PID targets the process group.
	evidence.PID = pid
	evidence.PGID = pgid

	// Send SIGTERM to the process group.
	if err := sendSignal(pgid, syscall.SIGTERM, reason); err != nil {
		return evidence, err
	}
	evidence.SIGTERMSent = true
	evidence.ActionSequence = append(evidence.ActionSequence, "sigterm_sent")

	// @AX:WARN: [AUTO] wait goroutine has no context and relies on bounded SIGTERM/SIGKILL escalation.
	// @AX:REASON: StopWithEvidence must only be used with registered subprocesses and fixed timeouts so Wait cannot outlive cleanup indefinitely.
	// Wait for process exit or force kill after timeout.
	done := make(chan struct{})
	go func() {
		_ = e.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return evidence, nil
	case <-time.After(5 * time.Second):
		// Process did not exit in time — escalate to SIGKILL.
		if err := sendSignal(pgid, syscall.SIGKILL, reason); err != nil {
			return evidence, err
		}
		evidence.SIGKILLSent = true
		evidence.ActionSequence = append(evidence.ActionSequence, "sigkill_sent")

		// Wait for the killed process to be reaped.
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_, _ = fmt.Fprintf(os.Stderr, "[EMERGENCY] process %d did not exit after SIGKILL (reason: %s)\n", pid, reason)
		}
		return evidence, nil
	}
}
