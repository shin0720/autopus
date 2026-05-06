package worker

import (
	"fmt"
	"log"
	"strings"

	"github.com/insajin/autopus-adk/pkg/worker/adapter"
)

// @AX:NOTE: [AUTO] reason code must match pipeline safety evidence for root-worktree fallback diagnostics.
const reasonWorktreeIsolationUnavailable = "worktree_isolation_unavailable"

func (wl *WorkerLoop) assignTaskWorktree(taskID string, taskCfg *adapter.TaskConfig) (func(), error) {
	if wl.worktreeManager == nil || !wl.config.WorktreeIsolation {
		return func() {}, nil
	}

	wtPath, err := wl.worktreeManager.Create(taskID)
	if err != nil {
		return wl.handleWorktreeCreateFailure(taskID, err)
	}
	taskCfg.WorkDir = wtPath
	if prepErr := prepareSymphonyWorkspace(taskCfg.WorkDir, taskCfg.Prompt); prepErr != nil {
		log.Printf("[worker] symphony workspace prepare failed for %s: %v", taskID, prepErr)
	}
	if envErr := prepareTaskRuntimeEnv(taskCfg); envErr != nil {
		log.Printf("[worker] runtime env prepare failed for %s: %v", taskID, envErr)
	}
	return wl.reclaimWorktreeOnExit(taskID, wtPath), nil
}

func (wl *WorkerLoop) assignPipelineWorktree(taskID, prompt string) (string, map[string]string, func(), error) {
	workDir := wl.config.WorkDir
	if wl.worktreeManager == nil || !wl.config.WorktreeIsolation {
		return workDir, nil, func() {}, nil
	}

	wtPath, err := wl.worktreeManager.Create(taskID)
	if err != nil {
		cleanup, fallbackErr := wl.handleWorktreeCreateFailure(taskID, err)
		return workDir, nil, cleanup, fallbackErr
	}
	if prepErr := prepareSymphonyWorkspace(wtPath, prompt); prepErr != nil {
		log.Printf("[worker] symphony workspace prepare failed for %s: %v", taskID, prepErr)
	}
	runtimeCfg := adapter.TaskConfig{TaskID: taskID, WorkDir: wtPath}
	if envErr := prepareTaskRuntimeEnv(&runtimeCfg); envErr != nil {
		log.Printf("[worker] runtime env prepare failed for %s: %v", taskID, envErr)
	}
	return wtPath, runtimeCfg.EnvVars, wl.reclaimWorktreeOnExit(taskID, wtPath), nil
}

func (wl *WorkerLoop) handleWorktreeCreateFailure(taskID string, err error) (func(), error) {
	override := strings.TrimSpace(wl.config.WorktreeFallbackOverrideReason)
	recordWorkerSafetyEvent(wl, newAuditDegradedEvent(taskID, reasonWorktreeIsolationUnavailable, override))
	if override == "" {
		return func() {}, fmt.Errorf("%s: %w", reasonWorktreeIsolationUnavailable, err)
	}
	log.Printf("[worker] worktree create failed for %s, explicit fallback override active: %v", taskID, err)
	return func() {}, nil
}

func (wl *WorkerLoop) reclaimWorktreeOnExit(taskID, wtPath string) func() {
	return func() {
		removed, rmErr := wl.worktreeManager.RemoveIfClean(wtPath)
		wtRef := sanitizeAuditRef(wtPath)
		if rmErr != nil {
			log.Printf("[worker] worktree remove failed for %s: %s", wtRef, sanitizeAuditMessage(rmErr.Error()))
			recordWorkerSafetyEvent(wl, newAuditReclaimedEvent(taskID, wtPath, "cleanup_failed"))
			return
		}
		if !removed {
			log.Printf("[worker] preserving dirty worktree for %s: %s", taskID, wtRef)
			recordWorkerSafetyEvent(wl, newAuditReclaimedEvent(taskID, wtPath, "preserved_for_manual_review"))
			return
		}
		recordWorkerSafetyEvent(wl, newAuditReclaimedEvent(taskID, wtPath, "discarded"))
	}
}

func recordWorkerSafetyEvent(wl *WorkerLoop, evt AuditEvent) {
	if wl != nil && wl.auditWriter != nil {
		recordAuditEvent(wl.auditWriter, evt, wl.auditLogger)
	}
}
