package worker

import (
	"os"
	"path/filepath"
	"testing"
)

// B1 dual-flag policy:
//   - empty/unset/"disabled"/unknown                    -> disabled
//   - "dry-run" (regardless of ENFORCE)                 -> dry-run
//   - "enforce" + AUTOPUS_COMMAND_GUARD_ENFORCE=1       -> enforce
//   - "enforce" alone OR ENFORCE=1 alone                -> disabled
//
// These tests exercise the worker resolver in isolation via t.Setenv.
// They MUST NOT execute commands, start processes, touch the network, or
// modify the workspace.

// envMatrixCase pairs an env-state to its expected resolver result. mode="" /
// enforce="" are treated identically to "unset" by os.Getenv.
type workerEnvMatrixCase struct {
	name    string
	mode    string
	enforce string
	want    workerCommandGuardMode
}

func TestWorkerCommandGuardEnforceMode_EnvMatrix(t *testing.T) {
	cases := []workerEnvMatrixCase{
		{"unset_unset_disabled", "", "", workerGuardDisabled},
		{"mode_enforce_only_disabled_fallback", "enforce", "", workerGuardDisabled},
		{"enforce_flag_only_disabled_fallback", "", "1", workerGuardDisabled},
		{"both_set_enforce", "enforce", "1", workerGuardEnforce},
		{"dry_run_alone_remains_dry_run", "dry-run", "", workerGuardDryRun},
		{"dry_run_plus_enforce_flag_remains_dry_run", "dry-run", "1", workerGuardDryRun},
		{"mode_unknown_value_disabled", "bogus", "1", workerGuardDisabled},
		{"mode_enforce_enforce_zero_disabled", "enforce", "0", workerGuardDisabled},
		{"mode_enforce_enforce_true_disabled", "enforce", "true", workerGuardDisabled},
		{"mode_disabled_explicit_disabled", "disabled", "1", workerGuardDisabled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", tc.mode)
			t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", tc.enforce)
			got := resolveWorkerCommandGuardModeFromEnv()
			if got != tc.want {
				t.Fatalf("resolveWorkerCommandGuardModeFromEnv() = %v want %v (MODE=%q ENFORCE=%q)",
					got, tc.want, tc.mode, tc.enforce)
			}
		})
	}
}

func TestWorkerCommandGuardEnforceMode_DefaultDisabledRegression(t *testing.T) {
	var zero workerCommandGuardMode
	if zero != workerGuardDisabled {
		t.Errorf("zero workerCommandGuardMode must equal workerGuardDisabled (got %v)", zero)
	}
	prevMode, prevApplied := workerCommandGuardHook.mode, workerEnvModeApplied
	t.Cleanup(func() {
		workerCommandGuardHook.mode = prevMode
		workerEnvModeApplied = prevApplied
	})
	workerCommandGuardHook.mode = workerGuardDisabled
	workerEnvModeApplied = false
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "")
	if got := currentWorkerCommandGuardMode(); got != workerGuardDisabled {
		t.Errorf("currentWorkerCommandGuardMode at default = %v want disabled", got)
	}
}

func TestWorkerCommandGuardEnforceMode_RollbackPath(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	if got := resolveWorkerCommandGuardModeFromEnv(); got != workerGuardEnforce {
		t.Fatalf("setup: want enforce, got %v", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "0")
	if got := resolveWorkerCommandGuardModeFromEnv(); got != workerGuardDisabled {
		t.Errorf("ENFORCE=0 rollback: got %v want disabled", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "")
	if got := resolveWorkerCommandGuardModeFromEnv(); got != workerGuardDisabled {
		t.Errorf("ENFORCE unset rollback: got %v want disabled", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	if got := resolveWorkerCommandGuardModeFromEnv(); got != workerGuardDisabled {
		t.Errorf("MODE unset rollback: got %v want disabled", got)
	}
}

func TestWorkerCommandGuardEnforceMode_ResolverIsPureAndIdempotent(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	prevDecision := lastWorkerCommandGuardDecision
	prevRequest := lastWorkerCommandGuardRequest
	first := resolveWorkerCommandGuardModeFromEnv()
	for i := 0; i < 100; i++ {
		if got := resolveWorkerCommandGuardModeFromEnv(); got != first {
			t.Fatalf("resolver not idempotent at iter %d: got %v want %v", i, got, first)
		}
	}
	if lastWorkerCommandGuardDecision != prevDecision {
		t.Errorf("resolver must not mutate lastWorkerCommandGuardDecision")
	}
	if lastWorkerCommandGuardRequest.Executable != prevRequest.Executable {
		t.Errorf("resolver must not mutate lastWorkerCommandGuardRequest")
	}
}

func TestWorkerCommandGuardEnforceMode_NoWorkspaceTelemetryCreated(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	probe := filepath.Join(cwd, ".autopus", "telemetry")
	before := statExists(probe)
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	_ = resolveWorkerCommandGuardModeFromEnv()
	if statExists(probe) != before {
		t.Errorf("resolveWorkerCommandGuardModeFromEnv must not create/remove %s (before=%v after=%v)",
			probe, before, statExists(probe))
	}
}

func statExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
