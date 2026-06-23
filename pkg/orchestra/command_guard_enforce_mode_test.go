package orchestra

import (
	"os"
	"path/filepath"
	"testing"
)

// B1 dual-flag policy (orchestra side, mirrors worker semantics):
//   - empty/unset/"disabled"/unknown                    -> disabled
//   - "dry-run" (regardless of ENFORCE)                 -> dry-run
//   - "enforce" + AUTOPUS_COMMAND_GUARD_ENFORCE=1       -> enforce
//   - "enforce" alone OR ENFORCE=1 alone                -> disabled
//
// These tests exercise the orchestra resolver via t.Setenv only.
// They MUST NOT execute commands, start processes, touch the network, or
// modify the workspace.

type orchestraEnvMatrixCase struct {
	name    string
	mode    string
	enforce string
	want    commandGuardMode
}

func TestOrchestraCommandGuardEnforceMode_EnvMatrix(t *testing.T) {
	cases := []orchestraEnvMatrixCase{
		{"unset_unset_disabled", "", "", guardModeDisabled},
		{"mode_enforce_only_disabled_fallback", "enforce", "", guardModeDisabled},
		{"enforce_flag_only_disabled_fallback", "", "1", guardModeDisabled},
		{"both_set_enforce", "enforce", "1", guardModeEnforce},
		{"dry_run_alone_remains_dry_run", "dry-run", "", guardModeDryRun},
		{"dry_run_plus_enforce_flag_remains_dry_run", "dry-run", "1", guardModeDryRun},
		{"mode_unknown_value_disabled", "bogus", "1", guardModeDisabled},
		{"mode_enforce_enforce_zero_disabled", "enforce", "0", guardModeDisabled},
		{"mode_enforce_enforce_true_disabled", "enforce", "true", guardModeDisabled},
		{"mode_disabled_explicit_disabled", "disabled", "1", guardModeDisabled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", tc.mode)
			t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", tc.enforce)
			got := resolveOrchestraCommandGuardModeFromEnv()
			if got != tc.want {
				t.Fatalf("resolveOrchestraCommandGuardModeFromEnv() = %v want %v (MODE=%q ENFORCE=%q)",
					got, tc.want, tc.mode, tc.enforce)
			}
		})
	}
}

func TestOrchestraCommandGuardEnforceMode_DefaultDisabledRegression(t *testing.T) {
	var zero commandGuardMode
	if zero != guardModeDisabled {
		t.Errorf("zero commandGuardMode must equal guardModeDisabled (got %v)", zero)
	}
	prevMode, prevApplied := commandGuardHook.mode, orchestraEnvModeApplied
	t.Cleanup(func() {
		commandGuardHook.mode = prevMode
		orchestraEnvModeApplied = prevApplied
	})
	commandGuardHook.mode = guardModeDisabled
	orchestraEnvModeApplied = false
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "")
	if got := currentCommandGuardMode(); got != guardModeDisabled {
		t.Errorf("currentCommandGuardMode at default = %v want disabled", got)
	}
}

func TestOrchestraCommandGuardEnforceMode_RollbackPath(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	if got := resolveOrchestraCommandGuardModeFromEnv(); got != guardModeEnforce {
		t.Fatalf("setup: want enforce, got %v", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "0")
	if got := resolveOrchestraCommandGuardModeFromEnv(); got != guardModeDisabled {
		t.Errorf("ENFORCE=0 rollback: got %v want disabled", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "")
	if got := resolveOrchestraCommandGuardModeFromEnv(); got != guardModeDisabled {
		t.Errorf("ENFORCE unset rollback: got %v want disabled", got)
	}
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	if got := resolveOrchestraCommandGuardModeFromEnv(); got != guardModeDisabled {
		t.Errorf("MODE unset rollback: got %v want disabled", got)
	}
}

func TestOrchestraCommandGuardEnforceMode_ResolverIsPureAndIdempotent(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	prevDecision := lastCommandGuardDecision
	first := resolveOrchestraCommandGuardModeFromEnv()
	for i := 0; i < 100; i++ {
		if got := resolveOrchestraCommandGuardModeFromEnv(); got != first {
			t.Fatalf("resolver not idempotent at iter %d: got %v want %v", i, got, first)
		}
	}
	if lastCommandGuardDecision != prevDecision {
		t.Errorf("resolver must not mutate lastCommandGuardDecision")
	}
}

func TestOrchestraCommandGuardEnforceMode_NoWorkspaceTelemetryCreated(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	probe := filepath.Join(cwd, ".autopus", "telemetry")
	before := orchestraStatExists(probe)
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	t.Setenv("AUTOPUS_COMMAND_GUARD_ENFORCE", "1")
	_ = resolveOrchestraCommandGuardModeFromEnv()
	if orchestraStatExists(probe) != before {
		t.Errorf("resolveOrchestraCommandGuardModeFromEnv must not create/remove %s (before=%v after=%v)",
			probe, before, orchestraStatExists(probe))
	}
}

func orchestraStatExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
