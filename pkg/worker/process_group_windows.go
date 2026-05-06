//go:build windows

package worker

import (
	"context"
	"os/exec"
)

func prepareCommandProcessGroup(cmd *exec.Cmd) {}

// @AX:NOTE: [AUTO] Windows interrupt propagation is intentionally evidence-only until process group support is implemented.
func watchCommandCancellation(ctx context.Context, cmd *exec.Cmd, taskID string, record func(AuditEvent)) func() {
	if ctx == nil {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if record != nil {
				record(newAuditInterruptEvent(taskID, "interrupt_propagation_unsupported", false, false, []string{"unsupported_platform"}))
			}
		case <-done:
		}
	}()
	return func() {
		close(done)
	}
}
