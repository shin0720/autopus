//go:build !windows

package worker

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func prepareCommandProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func watchCommandCancellation(ctx context.Context, cmd *exec.Cmd, taskID string, record func(AuditEvent)) func() {
	if ctx == nil || cmd == nil {
		return func() {}
	}

	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(done)
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			evt := terminateProcessGroup(cmd, taskID)
			if record != nil {
				record(evt)
			}
		case <-done:
		}
	}()

	return stop
}

// @AX:WARN: [AUTO] process-group termination sends SIGTERM then SIGKILL to the negative child PID.
// @AX:REASON: Safety depends on prepareCommandProcessGroup setting Setpgid before start; otherwise signal scope can be wrong or incomplete.
func terminateProcessGroup(cmd *exec.Cmd, taskID string) AuditEvent {
	evt := newAuditInterruptEvent(taskID, "context_cancelled", false, false, nil)
	if cmd == nil || cmd.Process == nil {
		evt.ActionSequence = append(evt.ActionSequence, "no_process")
		return evt
	}

	pgid := -cmd.Process.Pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		log.Printf("[worker] task %s: SIGTERM process group failed: %v", taskID, err)
		evt.ActionSequence = append(evt.ActionSequence, "sigterm_failed")
		return evt
	}
	evt.SIGTERMSent = true
	evt.ActionSequence = append(evt.ActionSequence, "sigterm_sent")

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	<-timer.C

	if err := syscall.Kill(pgid, 0); err != nil {
		return evt
	}
	if err := syscall.Kill(pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		log.Printf("[worker] task %s: SIGKILL process group failed: %v", taskID, err)
		evt.ActionSequence = append(evt.ActionSequence, "sigkill_failed")
		return evt
	}
	evt.SIGKILLSent = true
	evt.ActionSequence = append(evt.ActionSequence, "sigkill_sent")
	return evt
}
