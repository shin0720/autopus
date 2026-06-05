package worker

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// opCtx exercises the production bootstrap chain (telemetry.EnsureDefault) in
// a CWD-isolated environment: os.Chdir(t.TempDir()) forces DefaultDir() to
// resolve under the test temp dir, so workspace .autopus/telemetry/ stays clean.
type opCtx struct {
	t       *testing.T
	tempDir string
	origCWD string
}

func newWorkerOpCtx(t *testing.T) *opCtx {
	t.Helper()
	origCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	prevEmitter := telemetry.GetEmitter()
	telemetry.SetEmitter(nil)
	telemetry.ResetBootstrapForTest()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origCWD)
		telemetry.SetEmitter(prevEmitter)
		telemetry.ResetBootstrapForTest()
	})
	telemetry.EnsureDefault()
	if telemetry.GetEmitter() == nil {
		t.Fatal("EnsureDefault must install an emitter")
	}
	return &opCtx{t: t, tempDir: tempDir, origCWD: origCWD}
}

// setupWorkerDryRun is the common entry for dry-run scenarios: it installs the
// dry-run mode setter, builds an opCtx, and registers the setter restore on
// test cleanup.
func setupWorkerDryRun(t *testing.T) *opCtx {
	restore := setWorkerCommandGuardHookModeForTest(workerGuardDryRun)
	t.Cleanup(restore)
	return newWorkerOpCtx(t)
}

func (o *opCtx) telemetryDir() string {
	return filepath.Join(o.tempDir, ".autopus", "telemetry", "command_guard")
}

func (o *opCtx) rawContents() string {
	entries, err := os.ReadDir(o.telemetryDir())
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ndjson") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(o.telemetryDir(), e.Name()))
		if rerr != nil {
			o.t.Fatalf("read %s: %v", e.Name(), rerr)
		}
		sb.Write(data)
	}
	return sb.String()
}

func (o *opCtx) records() []telemetry.Record {
	raw := o.rawContents()
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
			o.t.Fatalf("unmarshal %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

// mustOneRecord requires exactly 1 NDJSON line, runs writer + workspace checks,
// and returns the single record.
func (o *opCtx) mustOneRecord() telemetry.Record {
	o.t.Helper()
	recs := o.records()
	if len(recs) != 1 {
		o.t.Fatalf("want 1 record, got %d", len(recs))
	}
	o.checkWriter()
	o.checkWorkspace()
	return recs[0]
}

func (o *opCtx) assertNoFile() {
	o.t.Helper()
	entries, err := os.ReadDir(o.telemetryDir())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			o.checkWriter()
			o.checkWorkspace()
			return
		}
		o.t.Errorf("readdir: %v", err)
		return
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ndjson") {
			o.t.Errorf("expected no NDJSON file, found %s", e.Name())
		}
	}
	o.checkWriter()
	o.checkWorkspace()
}

func (o *opCtx) checkWriter() {
	o.t.Helper()
	w, ok := telemetry.GetEmitter().(*telemetry.Writer)
	if !ok {
		o.t.Errorf("emitter must be *Writer, got %T", telemetry.GetEmitter())
		return
	}
	if n := w.WriteErrors(); n != 0 {
		o.t.Errorf("WriteErrors=%d, want 0", n)
	}
}

func (o *opCtx) checkWorkspace() {
	o.t.Helper()
	if _, err := os.Stat(filepath.Join(o.origCWD, ".autopus", "telemetry")); err == nil {
		o.t.Errorf("workspace .autopus/telemetry at %s must not be created", o.origCWD)
	}
}

func (o *opCtx) assertNoSubstring(forbidden ...string) {
	o.t.Helper()
	raw := o.rawContents()
	for _, s := range forbidden {
		if s != "" && strings.Contains(raw, s) {
			o.t.Errorf("NDJSON contains forbidden substring %q", s)
		}
	}
}

type opExpected struct {
	allowed bool
	guardID string
	t12     bool
	inert   bool
}

func assertWorkerOpRecord(t *testing.T, rec telemetry.Record, exp opExpected) {
	t.Helper()
	if rec.SchemaVersion != 1 || rec.Mode != "dry-run" || rec.Source != "worker" ||
		rec.SourceFile != "pkg/worker/command_guard_hook.go" || rec.SourceFunction != "workerCommandGuardCheck" {
		t.Errorf("header mismatch: %+v", rec)
	}
	if !rec.NoSecretRawArgs || len([]rune(rec.CommandPreview)) > 120 {
		t.Errorf("invariant: no_secret_raw_args=%v preview_len=%d", rec.NoSecretRawArgs, len([]rune(rec.CommandPreview)))
	}
	if rec.DecisionAllowed != exp.allowed || rec.WouldBlockInEnforce == rec.DecisionAllowed {
		t.Errorf("decision_allowed=%v would_block=%v want allowed=%v", rec.DecisionAllowed, rec.WouldBlockInEnforce, exp.allowed)
	}
	if rec.GuardID != exp.guardID || rec.T12FailClosed != exp.t12 || rec.M3M4Inert != exp.inert {
		t.Errorf("guard=%q t12=%v inert=%v want %+v", rec.GuardID, rec.T12FailClosed, rec.M3M4Inert, exp)
	}
}

// WOP-01
func TestOperationalEvidence_Worker_BootstrapAllowSafeCommand(t *testing.T) {
	op := setupWorkerDryRun(t)
	if _, ok := telemetry.GetEmitter().(*telemetry.Writer); !ok {
		t.Errorf("bootstrap must install *Writer")
	}
	if err := workerCommandGuardCheck(mkCmd("git", "status", "-sb"), "", ""); err != nil {
		t.Fatalf("safe must not block: %v", err)
	}
	assertWorkerOpRecord(t, op.mustOneRecord(), opExpected{allowed: true, guardID: "P8a", inert: true})
}

// WOP-02
func TestOperationalEvidence_Worker_BootstrapDenyDangerousGitAdd(t *testing.T) {
	op := setupWorkerDryRun(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	assertWorkerOpRecord(t, op.mustOneRecord(), opExpected{allowed: false, guardID: "M5", inert: true})
}

// WOP-03: T12 dangerous Makefile recipe — Phase=ScriptInspector -> guard_id "M6".
func TestOperationalEvidence_Worker_T12DangerousMakefileRecipe(t *testing.T) {
	op := setupWorkerDryRun(t)
	mkDir := writeTempMakefile(t, "Makefile", "release:\n\tgit push origin main\n")
	if err := workerCommandGuardCheck(mkMakeCmd(mkDir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := op.mustOneRecord()
	assertWorkerOpRecord(t, rec, opExpected{allowed: false, guardID: "M6", t12: true, inert: true})
	if rec.MakeTarget != "release" || rec.MakefileStatus != "Makefile" {
		t.Errorf("T12 fields: target=%q status=%q", rec.MakeTarget, rec.MakefileStatus)
	}
}

// WOP-04
func TestOperationalEvidence_Worker_T12MissingMakefileFailClosed(t *testing.T) {
	op := setupWorkerDryRun(t)
	mkDir := t.TempDir()
	if err := workerCommandGuardCheck(mkMakeCmd(mkDir, "release"), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := op.mustOneRecord()
	assertWorkerOpRecord(t, rec, opExpected{allowed: false, guardID: "M6", t12: true, inert: true})
	if rec.MakefileStatus != "" {
		t.Errorf("makefile_status=%q want empty", rec.MakefileStatus)
	}
}

// WOP-05: token redaction
func TestOperationalEvidence_Worker_RedactionSecretInArgs(t *testing.T) {
	op := setupWorkerDryRun(t)
	secret := "ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII"
	url := "https://oauth2:" + secret + "@example/x.git"
	if err := workerCommandGuardCheck(mkCmd("git", "push", url), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := op.mustOneRecord()
	assertWorkerOpRecord(t, rec, opExpected{allowed: false, guardID: "M5", inert: true})
	if !rec.Redacted {
		t.Errorf("redacted must be true")
	}
	op.assertNoSubstring(secret)
}

// WOP-06: Windows home path redaction
func TestOperationalEvidence_Worker_HomePathRedaction(t *testing.T) {
	op := setupWorkerDryRun(t)
	userSeg := "shin0720"
	if err := workerCommandGuardCheck(mkCmd("git", "status", `C:\Users\`+userSeg+`\workspace`), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := op.mustOneRecord()
	assertWorkerOpRecord(t, rec, opExpected{allowed: true, guardID: "P8a", inert: true})
	if !rec.Redacted {
		t.Errorf("redacted must be true for home path input")
	}
	op.assertNoSubstring(userSeg)
}

// WOP-07: env value redaction via *_TOKEN suffix whitelist
func TestOperationalEvidence_Worker_EnvValueRedaction(t *testing.T) {
	op := setupWorkerDryRun(t)
	val := "verysecretvalue123"
	if err := workerCommandGuardCheck(mkCmd("git", "status", "MY_TOKEN="+val), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := op.mustOneRecord()
	if !rec.Redacted {
		t.Errorf("redacted must be true for env value input")
	}
	op.assertNoSubstring(val)
}

// WOP-08: default disabled -> no NDJSON file in tempDir
func TestOperationalEvidence_Worker_DisabledNoFileCreate(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "")
	op := newWorkerOpCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("disabled must not block: %v", err)
	}
	op.assertNoFile()
}

// WOP-09: env=enforce -> disabled fallback -> no NDJSON file
func TestOperationalEvidence_Worker_EnvEnforceDisabledFallback(t *testing.T) {
	t.Setenv("AUTOPUS_COMMAND_GUARD_MODE", "enforce")
	op := newWorkerOpCtx(t)
	if err := workerCommandGuardCheck(mkCmd("git", "add", "."), "", ""); err != nil {
		t.Fatalf("env=enforce must fall back to disabled: %v", err)
	}
	op.assertNoFile()
}
