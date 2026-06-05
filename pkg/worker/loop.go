package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
	"github.com/shin0720/auto-adk/pkg/worker/a2a"
	"github.com/shin0720/auto-adk/pkg/worker/adapter"
	"github.com/shin0720/auto-adk/pkg/worker/audit"
	"github.com/shin0720/auto-adk/pkg/worker/auth"
	"github.com/shin0720/auto-adk/pkg/worker/knowledge"
	workerNet "github.com/shin0720/auto-adk/pkg/worker/net"
	"github.com/shin0720/auto-adk/pkg/worker/parallel"
	"github.com/shin0720/auto-adk/pkg/worker/pidlock"
	"github.com/shin0720/auto-adk/pkg/worker/reaper"
	"github.com/shin0720/auto-adk/pkg/worker/routing"
	"github.com/shin0720/auto-adk/pkg/worker/scheduler"
	"github.com/shin0720/auto-adk/pkg/worker/setup"
	"github.com/shin0720/auto-adk/pkg/worker/tui"
)

// LoopConfig holds configuration for the WorkerLoop.
type LoopConfig struct {
	BackendURL    string
	WorkerName    string
	MemoryAgentID string
	Skills        []string
	Provider      adapter.ProviderAdapter
	MCPConfig     string          // path to worker-mcp.json
	WorkDir       string          // working directory for subprocesses
	AuthToken     string          // bearer token for backend auth
	Router        *routing.Router // optional model router (nil = no routing)
	// Deprecated: use CredentialStore instead. Kept for backward compatibility.
	CredentialsPath   string                // path to credentials.json for token refresh
	CredentialStore   setup.CredentialStore // Secure credential storage (Keychain/encrypted file). If nil and CredentialsPath is set, falls back to plain file mode.
	AuditLogPath      string                // audit log file path (default: {WorkDir}/.autopus/audit.jsonl)
	AuditMaxSize      int64                 // max log size before rotation (default: 10MB)
	AuditMaxAge       time.Duration         // max age of rotated files (default: 7 days)
	WorkspaceID       string                // workspace identifier for scheduler
	MaxConcurrency    int                   // max parallel tasks (0 or 1 = sequential)
	WorktreeIsolation bool                  // enable worktree isolation for parallel tasks
	KnowledgeSync     bool                  // enable local knowledge context loading
	KnowledgeDir      string                // local knowledge directory hint (defaults to WorkDir)
}

// WorkerLoop integrates A2A Server, ProviderAdapter, ContextBuilder, and StreamParser.
// It receives tasks via A2A, builds prompts, spawns CLI subprocesses, and reports results.
type WorkerLoop struct {
	config            LoopConfig
	server            *a2a.Server
	builder           ContextBuilder
	tuiProgram        *tea.Program
	authRefresher     *auth.TokenRefresher
	authReconnector   *auth.Reconnector
	netMonitor        *workerNet.NetMonitor
	lifecycleCtx      context.Context
	lifecycleCancel   context.CancelFunc
	auditWriter       *audit.RotatingWriter
	knowledgeSearcher *knowledge.KnowledgeSearcher
	memorySearcher    *knowledge.MemorySearcher
	schedulerDisp     *scheduler.Dispatcher
	semaphore         *parallel.TaskSemaphore
	worktreeManager   *parallel.WorktreeManager
	auditLogger       *slogAuditLogger
	pidLock           *pidlock.Lock
	zombieReaper      *reaper.Reaper
}

// NewWorkerLoop creates a WorkerLoop with the given configuration.
func NewWorkerLoop(config LoopConfig) *WorkerLoop {
	wl := &WorkerLoop{
		config:      config,
		auditLogger: newSlogAuditLogger(3),
	}

	serverCfg := a2a.ServerConfig{
		BackendURL:            config.BackendURL,
		WorkerName:            config.WorkerName,
		WorkspaceID:           config.WorkspaceID,
		Skills:                config.Skills,
		Handler:               wl.handleTask,
		AuthToken:             config.AuthToken,
		ApprovalCallback:      wl.handleApproval,
		OnConnectionExhausted: wl.activateFallbackPoller,
	}
	wl.server = a2a.NewServer(serverCfg)

	return wl
}

func (wl *WorkerLoop) configureExecutionConcurrency() {
	concurrencyLimit := wl.config.MaxConcurrency
	if concurrencyLimit <= 1 {
		concurrencyLimit = 1
	}
	wl.semaphore = parallel.NewTaskSemaphore(concurrencyLimit)
	if wl.config.WorktreeIsolation && concurrencyLimit > 1 {
		wl.worktreeManager = parallel.NewWorktreeManager(wl.config.WorkDir)
	}
}

// Start connects to the backend and begins processing tasks.
// @AX:ANCHOR[AUTO]: public lifecycle entry point — Start/Close are the primary WorkerLoop API; callers (CLI, tests) depend on error contract
func (wl *WorkerLoop) Start(ctx context.Context) error {
	telemetry.EnsureDefault()
	// Acquire PID lock before starting to enforce single-instance constraint.
	wl.pidLock = pidlock.New(pidlock.DefaultPath())
	if err := wl.pidLock.Acquire(); err != nil {
		return fmt.Errorf("acquire PID lock: %w", err)
	}

	log.Printf("[worker] starting loop: provider=%s backend=%s", wl.config.Provider.Name(), wl.config.BackendURL)
	if err := wl.server.Start(ctx); err != nil {
		if releaseErr := wl.pidLock.Release(); releaseErr != nil {
			log.Printf("[worker] PID lock release failed on start error: %v", releaseErr)
		}
		return err
	}
	wl.startServices(ctx)

	wl.configureExecutionConcurrency()

	return nil
}

// Close shuts down the worker loop and its A2A server.
func (wl *WorkerLoop) Close() error {
	wl.stopServices()
	if wl.pidLock != nil {
		if err := wl.pidLock.Release(); err != nil {
			log.Printf("[worker] PID lock release failed: %v", err)
		}
	}
	return wl.server.Close()
}

// cleanupPolicy removes the cached SecurityPolicy file for the given task.
func cleanupPolicy(taskID string) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("autopus-%d", os.Getuid()))
	path := filepath.Join(dir, fmt.Sprintf("autopus-policy-%s.json", taskID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("[worker] cleanup policy file: %v", err)
	}
}

// SetTUIProgram registers the bubbletea program for sending approval messages.
func (wl *WorkerLoop) SetTUIProgram(p *tea.Program) {
	wl.tuiProgram = p
}

// handleApproval forwards an approval request from A2A to the TUI.
func (wl *WorkerLoop) handleApproval(params a2a.ApprovalRequestParams) {
	if wl.tuiProgram == nil {
		log.Printf("[worker] approval request but no TUI program registered")
		return
	}
	wl.tuiProgram.Send(tui.ApprovalRequestMsg{
		TaskID:    params.TaskID,
		Action:    params.Action,
		RiskLevel: params.RiskLevel,
		Context:   params.Context,
	})
}

// SetOnApprovalDecision returns a callback that sends approval decisions to the backend.
func (wl *WorkerLoop) SetOnApprovalDecision() func(taskID, decision string) {
	return func(taskID, decision string) {
		if err := wl.server.SendApprovalResponse(taskID, decision); err != nil {
			log.Printf("[worker] send approval response error: %v", err)
		}
	}
}

// convertArtifacts converts adapter artifacts to A2A artifacts.
func convertArtifacts(src []adapter.Artifact) []a2a.Artifact {
	if len(src) == 0 {
		return nil
	}
	out := make([]a2a.Artifact, len(src))
	for i, a := range src {
		out[i] = a2a.Artifact{
			Name:     a.Name,
			MimeType: a.MimeType,
			Data:     a.Data,
		}
	}
	return out
}

func ensureOutputArtifact(output string, artifacts []adapter.Artifact) []adapter.Artifact {
	if strings.TrimSpace(output) == "" {
		return artifacts
	}
	for _, artifact := range artifacts {
		if artifact.Name == "output" {
			return artifacts
		}
	}
	return append([]adapter.Artifact{{
		Name:     "output",
		MimeType: "text/plain",
		Data:     output,
	}}, artifacts...)
}
