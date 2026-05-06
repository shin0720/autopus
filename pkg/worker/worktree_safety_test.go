package worker

import (
	"context"
	"testing"

	"github.com/insajin/autopus-adk/pkg/worker/adapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteWithParallel_BlocksWhenRequiredWorktreeUnavailable(t *testing.T) {
	t.Parallel()

	mock := &mockAdapter{name: "mock", script: `head -c0; echo '{"type":"result","output":"should not run"}'`}
	wl := NewWorkerLoop(LoopConfig{
		Provider:          mock,
		WorkDir:           t.TempDir(),
		MaxConcurrency:    2,
		WorktreeIsolation: true,
	})
	wl.configureExecutionConcurrency()

	result, err := wl.executeWithParallel(context.Background(), adapter.TaskConfig{
		TaskID:  "worktree-required",
		Prompt:  "do work",
		WorkDir: wl.config.WorkDir,
	}, nil, newTaskRunMeta("worktree-required", ""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), reasonWorktreeIsolationUnavailable)
	assert.Empty(t, result.Output)
	assert.Len(t, mock.calls, 0)
}

func TestExecuteWithParallel_BlocksSingleSlotWhenIsolationRequired(t *testing.T) {
	t.Parallel()

	mock := &mockAdapter{name: "mock", script: `head -c0; echo '{"type":"result","output":"should not run"}'`}
	wl := NewWorkerLoop(LoopConfig{
		Provider:          mock,
		WorkDir:           t.TempDir(),
		MaxConcurrency:    1,
		WorktreeIsolation: true,
	})
	wl.configureExecutionConcurrency()

	result, err := wl.executeWithParallel(context.Background(), adapter.TaskConfig{
		TaskID:  "worktree-required-single-slot",
		Prompt:  "do work",
		WorkDir: wl.config.WorkDir,
	}, nil, newTaskRunMeta("worktree-required-single-slot", ""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), reasonWorktreeIsolationUnavailable)
	assert.Empty(t, result.Output)
	assert.Len(t, mock.calls, 0)
}

func TestExecuteWithParallel_ExplicitOverridePermitsFallback(t *testing.T) {
	t.Parallel()

	mock := &mockAdapter{name: "mock", script: `head -c0; echo '{"type":"result","output":"fallback ran"}'`}
	wl := NewWorkerLoop(LoopConfig{
		Provider:                       mock,
		WorkDir:                        t.TempDir(),
		MaxConcurrency:                 2,
		WorktreeIsolation:              true,
		WorktreeFallbackOverrideReason: "manual recovery in trusted checkout",
	})
	wl.configureExecutionConcurrency()

	result, err := wl.executeWithParallel(context.Background(), adapter.TaskConfig{
		TaskID:  "worktree-override",
		Prompt:  "do work",
		WorkDir: wl.config.WorkDir,
	}, nil, newTaskRunMeta("worktree-override", ""))

	require.NoError(t, err)
	assert.Equal(t, "fallback ran", result.Output)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, wl.config.WorkDir, mock.calls[0].WorkDir)
}
