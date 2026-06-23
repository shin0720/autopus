package worker

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// evCtx wires a t.TempDir()-backed Writer into the telemetry emitter slot so
// the hook's emit path writes NDJSON only inside the test's temp directory.
type evCtx struct {
	t      *testing.T
	dir    string
	writer *telemetry.Writer
	today  string
}

func newWorkerEvCtx(t *testing.T) *evCtx {
	t.Helper()
	dir := t.TempDir()
	w := telemetry.NewWriter(dir)
	prev := telemetry.GetEmitter()
	telemetry.SetEmitter(w)
	t.Cleanup(func() { telemetry.SetEmitter(prev) })
	return &evCtx{t: t, dir: dir, writer: w, today: time.Now().UTC().Format("2006-01-02")}
}

func (e *evCtx) ndjsonPath() string { return filepath.Join(e.dir, e.today+".ndjson") }

func (e *evCtx) rawContents() string {
	data, err := os.ReadFile(e.ndjsonPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ""
		}
		e.t.Fatalf("read NDJSON: %v", err)
	}
	return string(data)
}

func (e *evCtx) records() []telemetry.Record {
	raw := e.rawContents()
	if raw == "" {
		return nil
	}
	var out []telemetry.Record
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var rec telemetry.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			e.t.Fatalf("unmarshal %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func (e *evCtx) assertNoFile() {
	e.t.Helper()
	if _, err := os.Stat(e.ndjsonPath()); !errors.Is(err, fs.ErrNotExist) {
		e.t.Errorf("expected no NDJSON file at %s, got err=%v", e.ndjsonPath(), err)
	}
}

func (e *evCtx) assertWriterNoErrors() {
	e.t.Helper()
	if n := e.writer.WriteErrors(); n != 0 {
		e.t.Errorf("WriteErrors=%d, want 0", n)
	}
}

func (e *evCtx) assertWorkspaceNotPolluted() {
	e.t.Helper()
	if _, err := os.Stat(filepath.Join(".autopus", "telemetry")); err == nil {
		e.t.Errorf("workspace .autopus/telemetry/ must not be created by fixture")
	}
}

func (e *evCtx) assertNoSubstring(forbidden ...string) {
	e.t.Helper()
	raw := e.rawContents()
	for _, s := range forbidden {
		if s != "" && strings.Contains(raw, s) {
			e.t.Errorf("NDJSON contains forbidden substring %q", s)
		}
	}
}

// expectedRecord captures the minimal worker-side expectations per scenario.
type expectedRecord struct {
	allowed bool
	guardID string
	t12     bool
	inert   bool
}

func assertWorkerRecord(t *testing.T, rec telemetry.Record, exp expectedRecord) {
	t.Helper()
	if rec.SchemaVersion != 1 {
		t.Errorf("schema_version=%d want 1", rec.SchemaVersion)
	}
	if rec.Mode != "dry-run" {
		t.Errorf("mode=%q want dry-run", rec.Mode)
	}
	if rec.Source != "worker" {
		t.Errorf("source=%q want worker", rec.Source)
	}
	if rec.SourceFile != "pkg/worker/command_guard_hook.go" {
		t.Errorf("source_file=%q", rec.SourceFile)
	}
	if rec.SourceFunction != "workerCommandGuardCheck" {
		t.Errorf("source_function=%q", rec.SourceFunction)
	}
	if !rec.NoSecretRawArgs {
		t.Errorf("no_secret_raw_args must be true")
	}
	if got := len([]rune(rec.CommandPreview)); got > 120 {
		t.Errorf("command_preview rune length=%d > 120", got)
	}
	if rec.DecisionAllowed != exp.allowed {
		t.Errorf("decision_allowed=%v want %v", rec.DecisionAllowed, exp.allowed)
	}
	if rec.WouldBlockInEnforce == rec.DecisionAllowed {
		t.Errorf("would_block_in_enforce must be inverse of decision_allowed")
	}
	if rec.GuardID != exp.guardID {
		t.Errorf("guard_id=%q want %q", rec.GuardID, exp.guardID)
	}
	if rec.T12FailClosed != exp.t12 {
		t.Errorf("t12_fail_closed=%v want %v", rec.T12FailClosed, exp.t12)
	}
	if rec.M3M4Inert != exp.inert {
		t.Errorf("m3_m4_inert=%v want %v", rec.M3M4Inert, exp.inert)
	}
}

// W-01: safe command -> PhaseAllow -> guard_id "P8a", allowed.
func TestEvidenceFixture_Worker_AllowSafeCommand(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "", ""); err != nil {
		t.Fatalf("safe command must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: true, guardID: "P8a", t12: false, inert: true})
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-02: dangerous git -> M5 git_gate deny.
func TestEvidenceFixture_Worker_DenyDangerousGitAdd(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: false, guardID: "M5", t12: false, inert: true})
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-03: T12 dangerous Makefile recipe — Phase=ScriptInspector -> guard_id "M6", t12=true.
func TestEvidenceFixture_Worker_T12DangerousMakefileRecipe(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push origin main\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: false, guardID: "M6", t12: true, inert: true})
	if recs[0].MakeTarget != "release" {
		t.Errorf("make_target=%q want release", recs[0].MakeTarget)
	}
	if recs[0].MakefileStatus != "Makefile" {
		t.Errorf("makefile_status=%q want Makefile", recs[0].MakefileStatus)
	}
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-04: T12 missing Makefile fail-closed; MakefileStatus is empty.
func TestEvidenceFixture_Worker_T12MissingMakefileFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	dir := t.TempDir() // no Makefile written
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: false, guardID: "M6", t12: true, inert: true})
	if recs[0].MakefileStatus != "" {
		t.Errorf("makefile_status=%q want empty (no Makefile)", recs[0].MakefileStatus)
	}
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-05: secret in args is redacted; NDJSON contains no raw token substring.
func TestEvidenceFixture_Worker_RedactionSecretInArgs(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	secret := "ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII"
	url := "https://oauth2:" + secret + "@example/x.git"
	if err := workerCommandGuardCheck(mkCmd("git", "push", url), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: false, guardID: "M5", t12: false, inert: true})
	if !recs[0].Redacted {
		t.Errorf("redacted flag must be true for secret-containing input")
	}
	ev.assertNoSubstring(secret)
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-06: nil rulesets -> m3_m4_inert=true.
func TestEvidenceFixture_Worker_M3M4InertTrue(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "ccp_readonly", "claude"); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: true, guardID: "P8a", t12: false, inert: true})
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-07: both rulesets injected -> m3_m4_inert=false.
func TestEvidenceFixture_Worker_M3M4InertFalse(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	defer setWorkerCommandGuardRulesetsForTest(injectProfiles(), injectProviders())()
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "ccp_readonly", "claude"); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	recs := ev.records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(recs))
	}
	assertWorkerRecord(t, recs[0], expectedRecord{allowed: true, guardID: "P8a", t12: false, inert: false})
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-08: disabled mode (no setter, env unset) -> no NDJSON file created.
func TestEvidenceFixture_Worker_DisabledNoFileCreate(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("disabled must not block: %v", err)
	}
	ev.assertNoFile()
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}

// W-09: env=enforce -> resolveFromEnv returns disabled fallback -> no NDJSON.
func TestEvidenceFixture_Worker_EnvEnforceDisabledFallback(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	ev := newWorkerEvCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("env=enforce must fall back to disabled: %v", err)
	}
	ev.assertNoFile()
	ev.assertWriterNoErrors()
	ev.assertWorkspaceNotPolluted()
}
