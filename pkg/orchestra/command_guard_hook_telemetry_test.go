package orchestra

import (
	"context"
	"testing"

	"github.com/insajin/autopus-adk/pkg/guard/telemetry"
)

// installCapture wires a CaptureEmitter for the duration of the test and
// restores the previous emitter on cleanup.
func installCapture(t *testing.T) *telemetry.CaptureEmitter {
	t.Helper()
	prev := telemetry.GetEmitter()
	cap := &telemetry.CaptureEmitter{}
	telemetry.SetEmitter(cap)
	t.Cleanup(func() { telemetry.SetEmitter(prev) })
	return cap
}

func TestOrchestraTelemetry_DisabledModeEmitsZero(t *testing.T) {
	cap := installCapture(t)
	// default disabled, env unset -> disabled.
	c := newCommand(context.Background(), "git", "add", ".")
	if _, isDenied := c.(*deniedCommand); isDenied {
		t.Fatal("disabled must not block")
	}
	if cap.Len() != 0 {
		t.Errorf("disabled mode must emit 0 records, got %d", cap.Len())
	}
}

func TestOrchestraTelemetry_DryRunDangerousGitEmitsOne(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeDryRun)()
	cap := installCapture(t)
	c := newCommand(context.Background(), "git", "add", ".")
	if _, isDenied := c.(*deniedCommand); isDenied {
		t.Fatal("dry-run must not block")
	}
	if cap.Len() != 1 {
		t.Fatalf("dry-run dangerous git must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if rec.Source != "orchestra" {
		t.Errorf("source=%q want orchestra", rec.Source)
	}
	if rec.Mode != string(telemetry.ModeDryRun) {
		t.Errorf("mode=%q want dry-run", rec.Mode)
	}
	if rec.DecisionAllowed {
		t.Errorf("dangerous git must record decision_allowed=false")
	}
	if !rec.WouldBlockInEnforce {
		t.Errorf("would_block_in_enforce must be true for denied decision")
	}
	if rec.GuardID != "M5" {
		t.Errorf("guard_id=%q want M5", rec.GuardID)
	}
	if !rec.M3M4Inert {
		t.Errorf("orchestra hook has no provider/profile context; m3_m4_inert must be true")
	}
}

func TestOrchestraTelemetry_DryRunSafeCommandEmitsOne(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeDryRun)()
	cap := installCapture(t)
	c := newCommand(context.Background(), "git", "status", "-sb")
	if _, isDenied := c.(*deniedCommand); isDenied {
		t.Fatal("safe command must not be denied")
	}
	if cap.Len() != 1 {
		t.Fatalf("safe command must emit 1 record, got %d", cap.Len())
	}
	rec := cap.Records()[0]
	if !rec.DecisionAllowed {
		t.Errorf("safe command must record decision_allowed=true")
	}
}

func TestOrchestraTelemetry_EnforceBlockedStillEmits(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeEnforce)()
	cap := installCapture(t)
	c := newCommand(context.Background(), "git", "add", ".")
	if _, isDenied := c.(*deniedCommand); !isDenied {
		t.Fatal("enforce must block git add")
	}
	if cap.Len() != 1 {
		t.Errorf("enforce-blocked path must still emit 1 record, got %d", cap.Len())
	}
}

func TestOrchestraTelemetry_NoEmitterDoesNotBlock(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeDryRun)()
	prev := telemetry.GetEmitter()
	telemetry.SetEmitter(nil)
	t.Cleanup(func() { telemetry.SetEmitter(prev) })
	c := newCommand(context.Background(), "git", "add", ".")
	if _, isDenied := c.(*deniedCommand); isDenied {
		t.Fatal("dry-run must not block even when no emitter is installed")
	}
	// nothing to assert about emit (no emitter); the hook simply must not error.
}

func TestOrchestraTelemetry_RedactionFailureSkipsEmit(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeDryRun)()
	cap := installCapture(t)
	prev := telemetry.RedactionFailureProbe()
	telemetry.SetRedactionFailureProbe(true)
	defer telemetry.SetRedactionFailureProbe(prev)
	c := newCommand(context.Background(), "git", "status", "-sb")
	if _, isDenied := c.(*deniedCommand); isDenied {
		t.Error("redaction failure must not affect hook decision")
	}
	if cap.Len() != 0 {
		t.Errorf("redaction failure must skip emit, got %d records", cap.Len())
	}
}

func TestOrchestraTelemetry_WriteFailureNoEffectOnDecision(t *testing.T) {
	defer setCommandGuardHookModeForTest(guardModeEnforce)()
	prev := telemetry.GetEmitter()
	telemetry.SetEmitter(failingEmitter{})
	t.Cleanup(func() { telemetry.SetEmitter(prev) })

	c := newCommand(context.Background(), "git", "add", ".")
	if _, isDenied := c.(*deniedCommand); !isDenied {
		t.Error("write failure must not mask enforce deny")
	}

	c2 := newCommand(context.Background(), "git", "status", "-sb")
	if _, isDenied := c2.(*deniedCommand); isDenied {
		t.Error("write failure must not introduce a new deny on safe command")
	}
}

type failingEmitter struct{}

func (failingEmitter) Append(_ telemetry.Record) bool { return false }
