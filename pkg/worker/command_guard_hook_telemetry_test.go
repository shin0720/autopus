package worker

import (
	"testing"

	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// installCapture wires a CaptureEmitter for the duration of the test and
// restores the previous emitter on cleanup. The default emitter is nil in
// production builds, so the previous emitter is typically nil — preserving
// the "no file pollution by default" invariant for unrelated tests.
func installCapture(t *testing.T) *telemetry.CaptureEmitter {
	t.Helper()
	prev := telemetry.GetEmitter()
	cap := &telemetry.CaptureEmitter{}
	telemetry.SetEmitter(cap)
	t.Cleanup(func() { telemetry.SetEmitter(prev) })
	return cap
}

func TestWorkerTelemetry_DisabledModeEmitsZero(t *testing.T) {
	cap := installCapture(t)
	// no test setter: default disabled, env unset -> disabled.
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("disabled hook must not block: %v", err)
	}
	if cap.Len() != 0 {
		t.Errorf("disabled mode must emit 0 records, got %d", cap.Len())
	}
}

func TestWorkerTelemetry_DryRunDangerousGitEmitsOne(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	if cap.Len() != 1 {
		t.Fatalf("dry-run dangerous git must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if rec.Source != "worker" {
		t.Errorf("source=%q want worker", rec.Source)
	}
	if rec.Mode != string(telemetry.ModeDryRun) {
		t.Errorf("mode=%q want dry-run", rec.Mode)
	}
	if rec.DecisionAllowed {
		t.Errorf("dangerous git must be denied; decision_allowed=true")
	}
	if !rec.WouldBlockInEnforce {
		t.Errorf("would_block_in_enforce must be true for denied decision")
	}
	if rec.GuardID != "M5" {
		t.Errorf("guard_id=%q want M5 (git_gate)", rec.GuardID)
	}
}

func TestWorkerTelemetry_DryRunSafeCommandEmitsOne(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "", ""); err != nil {
		t.Fatalf("safe command must not block: %v", err)
	}
	if cap.Len() != 1 {
		t.Fatalf("safe command must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if !rec.DecisionAllowed {
		t.Errorf("safe command must record decision_allowed=true")
	}
	if rec.WouldBlockInEnforce {
		t.Errorf("safe command must not have would_block_in_enforce=true")
	}
}

func TestWorkerTelemetry_DryRunT12DangerousMakeEmitsOne(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push origin main\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run T12 must not block: %v", err)
	}
	if cap.Len() != 1 {
		t.Fatalf("T12 dangerous make must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if !rec.T12FailClosed {
		t.Errorf("t12_fail_closed must be true for T12 deny path")
	}
	if rec.MakeTarget != "release" {
		t.Errorf("make_target=%q want release", rec.MakeTarget)
	}
	if rec.MakefileStatus != "Makefile" {
		t.Errorf("makefile_status=%q want Makefile", rec.MakefileStatus)
	}
	if !rec.WouldBlockInEnforce {
		t.Errorf("T12 deny must set would_block_in_enforce=true")
	}
}

func TestWorkerTelemetry_DryRunMissingMakefileEmitsOne(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	dir := t.TempDir() // no Makefile created -> fail-closed
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run must not block on missing Makefile: %v", err)
	}
	if cap.Len() != 1 {
		t.Fatalf("missing Makefile must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if !rec.T12FailClosed {
		t.Errorf("t12_fail_closed must be true on missing Makefile")
	}
	if rec.MakefileStatus != "" {
		t.Errorf("makefile_status must be empty when no file matched, got %q", rec.MakefileStatus)
	}
}

func TestWorkerTelemetry_M3M4InertFlagWhenRulesetsNil(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	_ = workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "ccp_readonly", "claude")
	if cap.Len() != 1 {
		t.Fatalf("want 1 record, got %d", cap.Len())
	}
	if !cap.Records()[0].M3M4Inert {
		t.Errorf("m3_m4_inert must be true when ProfileSet/ProviderBindingSet are nil")
	}
}

func TestWorkerTelemetry_M3M4InertFalseWhenBothInjected(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	defer setWorkerCommandGuardRulesetsForTest(injectProfiles(), injectProviders())()
	cap := installCapture(t)
	_ = workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "ccp_readonly", "claude")
	if cap.Len() != 1 {
		t.Fatalf("want 1 record, got %d", cap.Len())
	}
	if cap.Records()[0].M3M4Inert {
		t.Errorf("m3_m4_inert must be false when both rulesets are injected")
	}
}

func TestWorkerTelemetry_EnforceBlockedStillEmits(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	cap := installCapture(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err == nil {
		t.Error("enforce must block git add")
	}
	if cap.Len() != 1 {
		t.Errorf("enforce-blocked path must still emit telemetry, got %d", cap.Len())
	}
}

func TestWorkerTelemetry_RedactionFailureSkipsEmit(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	cap := installCapture(t)
	// Force redaction to fail; emit must skip and hook decision must be unaffected.
	prev := redactionFailureProbeProxyForWorkerTest(true)
	defer redactionFailureProbeProxyForWorkerTest(prev)
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "", ""); err != nil {
		t.Errorf("redaction failure must not affect hook decision, got %v", err)
	}
	if cap.Len() != 0 {
		t.Errorf("redaction failure must skip emit, got %d records", cap.Len())
	}
}

// redactionFailureProbeProxyForWorkerTest toggles the telemetry package's
// redaction-failure probe via the shared test-only setter. Defined here to
// avoid leaking the toggle into the telemetry package's public surface.
func redactionFailureProbeProxyForWorkerTest(v bool) bool {
	prev := telemetry.RedactionFailureProbe()
	telemetry.SetRedactionFailureProbe(v)
	return prev
}

func TestWorkerTelemetry_WriteFailureNoEffectOnDecision(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	// install an emitter that always rejects (forced write failure)
	prev := telemetry.GetEmitter()
	telemetry.SetEmitter(failingEmitter{})
	t.Cleanup(func() { telemetry.SetEmitter(prev) })
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err == nil {
		t.Error("write failure must not mask enforce deny")
	}
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "", ""); err != nil {
		t.Errorf("write failure must not introduce a new error on safe command, got %v", err)
	}
}

type failingEmitter struct{}

func (failingEmitter) Append(_ telemetry.Record) bool { return false }

// mkCmd / writeTempMakefile / mkMakeCmd / injectProfiles / injectProviders are
// defined in sibling _test.go files in this package and reused here.
