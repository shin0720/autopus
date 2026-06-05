package worker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shin0720/auto-adk/pkg/guard"
	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// workerCommandGuardMode controls the P8b worker-path guard hook.
type workerCommandGuardMode int

const (
	// workerGuardDisabled is the DEFAULT: the hook is a no-op and the existing
	// BuildCommand->Start worker path is completely unchanged (regression baseline).
	workerGuardDisabled workerCommandGuardMode = iota
	// workerGuardDryRun evaluates and records a decision but never blocks.
	workerGuardDryRun
	// workerGuardEnforce returns an error before cmd.Start() on a denied command.
	workerGuardEnforce
)

type workerCommandGuardHookConfig struct {
	mode workerCommandGuardMode
	// profiles/providers are OPTIONAL rule sets (default nil). When nil, M3/M4
	// stay inert (the facade skips them). They are injected for testing/wiring
	// here; the real YAML->Go rule-set source is a separate later step.
	profiles  guard.ProfileSet
	providers guard.ProviderBindingSet
}

// workerCommandGuardHook defaults to DISABLED with nil rule sets, so the existing
// BuildCommand->Start path is unchanged. Enabling enforcement in production is a
// separate approval — this step only wires the hook.
var workerCommandGuardHook = workerCommandGuardHookConfig{mode: workerGuardDisabled}

// workerEnvModeApplied indicates a test setter has explicitly set the mode; the
// env override is then bypassed. In production, this stays false and the env
// variable AUTOPUS_COMMAND_GUARD_MODE is consulted.
var workerEnvModeApplied bool

// resolveWorkerCommandGuardModeFromEnv returns the effective mode under the B1
// dual-flag policy. Recognized: empty/unset/"disabled"/unknown -> disabled;
// "dry-run" -> dry-run; "enforce" -> enforce ONLY when AUTOPUS_COMMAND_GUARD_ENFORCE=1
// is also set. MODE=enforce alone or ENFORCE=1 alone falls back to disabled.
func resolveWorkerCommandGuardModeFromEnv() workerCommandGuardMode {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AUTOPUS_COMMAND_GUARD_MODE")))
	if raw == "dry-run" {
		return workerGuardDryRun
	}
	if raw == "enforce" && strings.TrimSpace(os.Getenv("AUTOPUS_COMMAND_GUARD_ENFORCE")) == "1" {
		return workerGuardEnforce
	}
	return workerGuardDisabled
}

// currentWorkerCommandGuardMode returns the effective mode: test-setter value if
// applied, otherwise the env-resolved value (dry-run-only policy).
func currentWorkerCommandGuardMode() workerCommandGuardMode {
	if workerEnvModeApplied {
		return workerCommandGuardHook.mode
	}
	return resolveWorkerCommandGuardModeFromEnv()
}

// lastWorkerCommandGuardDecision exposes the most recent decision for tests/logging.
var lastWorkerCommandGuardDecision guard.CommandGuardDecision

// lastWorkerCommandGuardRequest exposes the most recent request (for tests).
var lastWorkerCommandGuardRequest guard.CommandGuardRequest

// workerCommandGuardCheck evaluates an already-built *exec.Cmd before Start.
//
//   - profileID is the REAL per-task source (TaskConfig.ProfileID).
//   - rawProviderID is the per-worker adapter id (wl.config.Provider.Name()),
//     normalized via guard.NormalizeProviderID before M4.
//
// M3/M4 stay INERT unless a ProfileSet/ProviderBindingSet is injected: the facade
// skips them when Profiles/Providers are nil. The worker path is DISJOINT from
// the orchestra newCommand hook (no double enforcement).
func workerCommandGuardCheck(cmd *exec.Cmd, profileID, rawProviderID string) error {
	mode := currentWorkerCommandGuardMode()
	if mode == workerGuardDisabled || cmd == nil {
		return nil
	}
	// T12 runtime carve-out: a "make" invocation triggers the worker-only
	// Makefile-source check BEFORE the standard facade flow. The only os.ReadFile
	// site for SB8. Safe make targets fall through to the facade as usual.
	if isMakeExecutable(cmd) {
		if ms := inspectMakeCommandFromWorker(cmd); !ms.Allowed {
			lastWorkerCommandGuardDecision = guard.CommandGuardDecision{
				Phase:       guard.PhaseScriptInspector,
				Allowed:     false,
				Tool:        ms.Target,
				MatchedRule: ms.MatchedRule,
				Reason:      ms.Reason,
			}
			emitWorkerTelemetry(mode, cmd, profileID, rawProviderID, lastWorkerCommandGuardDecision, true, ms.Target, ms.MakefileTag)
			if mode == workerGuardEnforce {
				return fmt.Errorf("command blocked by guard (phase=make_runtime): %s", ms.Reason)
			}
			return nil
		}
	}
	dec := workerCommandGuardDecisionForExec(cmd, profileID, rawProviderID)
	lastWorkerCommandGuardDecision = dec
	emitWorkerTelemetry(mode, cmd, profileID, rawProviderID, dec, false, "", "")
	if mode == workerGuardEnforce && !dec.Allowed {
		return fmt.Errorf("command blocked by guard (phase=%s): %s", dec.Phase, dec.Reason)
	}
	return nil
}

// emitWorkerTelemetry builds and forwards a telemetry record for the just-
// evaluated command. mode==disabled or cmd==nil short-circuit before any work.
// The hook decision and execution path are unaffected by emit success/failure
// — telemetry is fire-and-forget and the return value is intentionally ignored.
func emitWorkerTelemetry(mode workerCommandGuardMode, cmd *exec.Cmd, profileID, rawProviderID string, dec guard.CommandGuardDecision, t12FailClosed bool, makeTarget, makefileStatus string) {
	if mode == workerGuardDisabled || cmd == nil {
		return
	}
	name := filepath.Base(cmd.Path)
	var args []string
	if len(cmd.Args) > 1 {
		args = cmd.Args[1:]
	}
	rawPreview := strings.TrimSpace(name + " " + strings.Join(args, " "))
	normalized := guard.NormalizeCommand(name, args).CompareString
	var providerYAMLID string
	if rawProviderID != "" {
		providerYAMLID = guard.NormalizeProviderID(rawProviderID).YAMLProviderID
	}
	rec, ok := telemetry.Build(telemetry.BuildInput{
		Mode:              workerTelemetryMode(mode),
		Source:            "worker",
		Provider:          rawProviderID,
		NormalizedCommand: normalized,
		CommandPreviewRaw: rawPreview,
		Decision:          dec,
		ProfileID:         profileID,
		ProviderID:        providerYAMLID,
		MakeTarget:        makeTarget,
		MakefileStatus:    makefileStatus,
		T12FailClosed:     t12FailClosed,
		M3M4Inert:         workerCommandGuardHook.profiles == nil || workerCommandGuardHook.providers == nil,
		SourceFile:        "pkg/worker/command_guard_hook.go",
		SourceFunction:    "workerCommandGuardCheck",
	})
	if !ok {
		return
	}
	_ = telemetry.Emit(rec)
}

func workerTelemetryMode(m workerCommandGuardMode) telemetry.Mode {
	switch m {
	case workerGuardDryRun:
		return telemetry.ModeDryRun
	case workerGuardEnforce:
		return telemetry.ModeEnforce
	}
	return telemetry.ModeDisabled
}

// workerCommandGuardDecisionForExec builds a facade request from a resolved
// command, the per-task ProfileID, and the per-worker provider id. Provider id
// normalization fail-closes (only when a ProviderBindingSet is injected) on an
// unmapped provider or one whose runtime adapter is absent (e.g. opencode).
func workerCommandGuardDecisionForExec(cmd *exec.Cmd, profileID, rawProviderID string) guard.CommandGuardDecision {
	// cmd.Path is the RESOLVED full path (may contain spaces); use the basename
	// so M1's whitespace split sees a clean executable token.
	name := filepath.Base(cmd.Path)
	var args []string
	if len(cmd.Args) > 1 {
		args = cmd.Args[1:]
	}
	raw := strings.TrimSpace(name + " " + strings.Join(args, " "))
	req := guard.CommandGuardRequest{
		Executable: name,
		Args:       args,
		RawScript:  raw,
		ProfileID:  profileID,
		Profiles:   workerCommandGuardHook.profiles,
		Providers:  workerCommandGuardHook.providers,
	}

	if rawProviderID != "" {
		m := guard.NormalizeProviderID(rawProviderID)
		req.ProviderID = m.YAMLProviderID
		if workerCommandGuardHook.providers != nil && (!m.Mapped || !m.AdapterPresent) {
			lastWorkerCommandGuardRequest = req
			reason := "provider unmapped (fail-closed)"
			if m.Mapped && !m.AdapterPresent {
				reason = "provider runtime adapter absent (fail-closed)"
			}
			return guard.CommandGuardDecision{
				Phase: guard.PhaseProviderBinding, Allowed: false, Tool: rawProviderID, Reason: reason,
			}
		}
	}

	lastWorkerCommandGuardRequest = req
	return guard.EvaluateCommandGuard(req)
}

// setWorkerCommandGuardHookModeForTest sets the mode and marks env override as
// applied (so env is bypassed for the duration), returning a restore func.
func setWorkerCommandGuardHookModeForTest(m workerCommandGuardMode) func() {
	prevMode := workerCommandGuardHook.mode
	prevApplied := workerEnvModeApplied
	workerCommandGuardHook.mode = m
	workerEnvModeApplied = true
	return func() {
		workerCommandGuardHook.mode = prevMode
		workerEnvModeApplied = prevApplied
	}
}

// setWorkerCommandGuardRulesetsForTest injects the optional rule sets, returning
// a restore func. This exercises M3/M4 wiring — it is not a production source.
func setWorkerCommandGuardRulesetsForTest(p guard.ProfileSet, pb guard.ProviderBindingSet) func() {
	prevP, prevPB := workerCommandGuardHook.profiles, workerCommandGuardHook.providers
	workerCommandGuardHook.profiles, workerCommandGuardHook.providers = p, pb
	return func() {
		workerCommandGuardHook.profiles, workerCommandGuardHook.providers = prevP, prevPB
	}
}
