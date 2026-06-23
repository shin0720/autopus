package orchestra

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shin0720/auto-adk/pkg/guard"
	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// commandGuardMode controls the P8b step1 newCommand hook.
type commandGuardMode int

const (
	// guardModeDisabled is the DEFAULT: the hook is a no-op and existing
	// execution paths are completely unchanged (regression baseline).
	guardModeDisabled commandGuardMode = iota
	// guardModeDryRun evaluates and records a decision but never blocks.
	guardModeDryRun
	// guardModeEnforce blocks a denied command before any Start()/Wait().
	guardModeEnforce
)

type commandGuardHookConfig struct {
	mode commandGuardMode
}

// commandGuardHook defaults to DISABLED. Enabling enforcement in production is a
// separate approval — this step only wires the hook so it can be toggled.
var commandGuardHook = commandGuardHookConfig{mode: guardModeDisabled}

// orchestraEnvModeApplied indicates a test setter has explicitly set the mode;
// env override is bypassed while true. In production, this stays false and the
// env variable AUTOPUS_COMMAND_GUARD_MODE is consulted.
var orchestraEnvModeApplied bool

// resolveOrchestraCommandGuardModeFromEnv returns the effective mode under the
// B1 dual-flag policy. Recognized: empty/unset/"disabled"/unknown -> disabled;
// "dry-run" -> dry-run; "enforce" -> enforce ONLY when AUTOPUS_COMMAND_GUARD_ENFORCE=1
// is also set. MODE=enforce alone or ENFORCE=1 alone falls back to disabled.
func resolveOrchestraCommandGuardModeFromEnv() commandGuardMode {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AUTOPUS_COMMAND_GUARD_MODE")))
	if raw == "dry-run" {
		return guardModeDryRun
	}
	if raw == "enforce" && strings.TrimSpace(os.Getenv("AUTOPUS_COMMAND_GUARD_ENFORCE")) == "1" {
		return guardModeEnforce
	}
	return guardModeDisabled
}

// currentCommandGuardMode returns the effective mode: test-setter value if
// applied, otherwise the env-resolved value (dry-run-only policy).
func currentCommandGuardMode() commandGuardMode {
	if orchestraEnvModeApplied {
		return commandGuardHook.mode
	}
	return resolveOrchestraCommandGuardModeFromEnv()
}

// lastCommandGuardDecision exposes the most recent decision for tests/logging.
var lastCommandGuardDecision guard.CommandGuardDecision

// commandGuardCheck evaluates a to-be-created command via the guard facade.
//
// SCOPE (P8b step1 — newCommand hook): only command-string gates are effective
// here. M5 git_gate and M6 script_inspector run on {name,args} / a reconstructed
// raw string. M2 denylist is wired through the facade but inert without an
// injected rule set. M3/M4/M7/M8 are NOT enforced at this point because
// newCommand has NO provider/profile context — they remain facade-only and their
// actual enforcement is a follow-up step.
//
// COVERAGE GAP: the worker path (pkg/worker/loop_subprocess BuildCommand->Start)
// does NOT pass through newCommand and is intentionally NOT covered here.
func commandGuardCheck(name string, args []string) (denied bool, blocked command) {
	mode := currentCommandGuardMode()
	if mode == guardModeDisabled {
		return false, nil
	}
	dec := commandGuardDecisionForExec(name, args)
	lastCommandGuardDecision = dec
	emitOrchestraTelemetry(mode, name, args, dec)
	if mode == guardModeEnforce && !dec.Allowed {
		return true, &deniedCommand{reason: dec.Reason, phase: string(dec.Phase)}
	}
	return false, nil
}

// emitOrchestraTelemetry builds and forwards a telemetry record for the just-
// evaluated newCommand input. M3/M4 are always marked inert here because the
// orchestra hook has no provider/profile context (see SCOPE note above).
// Emit success/failure does not affect the hook decision or process creation.
func emitOrchestraTelemetry(mode commandGuardMode, name string, args []string, dec guard.CommandGuardDecision) {
	if mode == guardModeDisabled {
		return
	}
	raw := strings.TrimSpace(name + " " + strings.Join(args, " "))
	normalized := guard.NormalizeCommand(name, args).CompareString
	rec, ok := telemetry.Build(telemetry.BuildInput{
		Mode:              orchestraTelemetryMode(mode),
		Source:            "orchestra",
		NormalizedCommand: normalized,
		CommandPreviewRaw: raw,
		Decision:          dec,
		M3M4Inert:         true,
		SourceFile:        "pkg/orchestra/command_guard_hook.go",
		SourceFunction:    "commandGuardCheck",
	})
	if !ok {
		return
	}
	_ = telemetry.Emit(rec)
}

func orchestraTelemetryMode(m commandGuardMode) telemetry.Mode {
	switch m {
	case guardModeDryRun:
		return telemetry.ModeDryRun
	case guardModeEnforce:
		return telemetry.ModeEnforce
	}
	return telemetry.ModeDisabled
}

// commandGuardDecisionForExec builds a facade request from exec inputs only.
// Provider/profile are deliberately absent (newCommand has no such context).
func commandGuardDecisionForExec(name string, args []string) guard.CommandGuardDecision {
	raw := strings.TrimSpace(name + " " + strings.Join(args, " "))
	return guard.EvaluateCommandGuard(guard.CommandGuardRequest{
		Executable: name,
		Args:       args,
		RawScript:  raw,
	})
}

// setCommandGuardHookModeForTest sets the mode and marks env override as
// applied (env bypassed for the duration), returning a restore func.
func setCommandGuardHookModeForTest(m commandGuardMode) func() {
	prevMode := commandGuardHook.mode
	prevApplied := orchestraEnvModeApplied
	commandGuardHook.mode = m
	orchestraEnvModeApplied = true
	return func() {
		commandGuardHook.mode = prevMode
		orchestraEnvModeApplied = prevApplied
	}
}

// deniedCommand satisfies the command interface but refuses to run: Start/Wait
// error out and NO OS process is ever created.
type deniedCommand struct {
	reason string
	phase  string
}

func (d *deniedCommand) err() error {
	return fmt.Errorf("command blocked by guard (phase=%s): %s", d.phase, d.reason)
}

func (d *deniedCommand) StdinPipe() (io.WriteCloser, error) { return nil, d.err() }
func (d *deniedCommand) SetStdin(io.Reader)                 {}
func (d *deniedCommand) SetStdout(io.Writer)                {}
func (d *deniedCommand) SetStderr(io.Writer)                {}
func (d *deniedCommand) SetDir(string)                      {}
func (d *deniedCommand) Start() error                       { return d.err() }
func (d *deniedCommand) Wait() error                        { return d.err() }
func (d *deniedCommand) ExitCode() int                      { return -1 }
func (d *deniedCommand) Terminate(string) error             { return nil }
