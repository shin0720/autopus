package worker

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/insajin/autopus-adk/pkg/guard/telemetry"
)

// naCtx exercises the production bootstrap chain (telemetry.EnsureDefault) in
// a CWD-isolated environment for the N-α-0 safety shim: every created exec.Cmd
// is tracked so we can verify Process / ProcessState stay nil (no Start/Run),
// and no provider binary (claude / gemini / codex / opencode) is resolved.
type naCtx struct {
	t          *testing.T
	tempDir    string
	absTempDir string
	origCWD    string
	cmds       []*exec.Cmd
}

var nAlphaProviderBinaries = []string{"claude", "gemini", "codex", "opencode"}

func newWorkerNACtx(t *testing.T) *naCtx {
	t.Helper()
	origCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
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
	return &naCtx{t: t, tempDir: tempDir, absTempDir: absTempDir, origCWD: origCWD}
}

func setupWorkerNADryRun(t *testing.T) *naCtx {
	restore := setWorkerCommandGuardHookModeForTest(workerGuardDryRun)
	t.Cleanup(restore)
	return newWorkerNACtx(t)
}

func (n *naCtx) track(c *exec.Cmd) *exec.Cmd {
	n.cmds = append(n.cmds, c)
	return c
}

func (n *naCtx) tdir() string {
	return filepath.Join(n.tempDir, ".autopus", "telemetry", "command_guard")
}

func (n *naCtx) rawContents() string {
	entries, err := os.ReadDir(n.tdir())
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ndjson") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(n.tdir(), e.Name()))
		if rerr != nil {
			n.t.Fatalf("read: %v", rerr)
		}
		sb.Write(b)
	}
	return sb.String()
}

func (n *naCtx) records() []telemetry.Record {
	var out []telemetry.Record
	for _, line := range strings.Split(strings.TrimSpace(n.rawContents()), "\n") {
		if line == "" {
			continue
		}
		var rec telemetry.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			n.t.Fatalf("unmarshal: %v", err)
		}
		out = append(out, rec)
	}
	return out
}

func (n *naCtx) mustOneRecord() telemetry.Record {
	n.t.Helper()
	recs := n.records()
	if len(recs) != 1 {
		n.t.Fatalf("want 1 record, got %d", len(recs))
	}
	n.checkInvariants()
	return recs[0]
}

func (n *naCtx) checkInvariants() {
	n.t.Helper()
	w, ok := telemetry.GetEmitter().(*telemetry.Writer)
	if !ok {
		n.t.Errorf("emitter must be *Writer, got %T", telemetry.GetEmitter())
	} else if errs := w.WriteErrors(); errs != 0 {
		n.t.Errorf("WriteErrors=%d want 0", errs)
	}
	if _, err := os.Stat(filepath.Join(n.origCWD, ".autopus", "telemetry")); err == nil {
		n.t.Errorf("workspace .autopus/telemetry at %s must not be created", n.origCWD)
	}
	for i, c := range n.cmds {
		if c.Process != nil || c.ProcessState != nil {
			n.t.Errorf("cmd[%d] %v: Process/ProcessState != nil (Start/Run called)", i, c.Args)
		}
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(c.Path), ".exe"))
		for _, p := range nAlphaProviderBinaries {
			if base == p {
				n.t.Errorf("cmd[%d] resolves to provider binary %q", i, base)
			}
		}
	}
	absDefault, err := filepath.Abs(telemetry.DefaultDir())
	if err != nil {
		n.t.Errorf("abs(DefaultDir): %v", err)
	} else if !strings.HasPrefix(absDefault, n.absTempDir) {
		n.t.Errorf("DefaultDir %q lacks tempDir prefix %q", absDefault, n.absTempDir)
	}
}

func (n *naCtx) assertNoSubstring(forbidden ...string) {
	n.t.Helper()
	raw := n.rawContents()
	for _, s := range forbidden {
		if s != "" && strings.Contains(raw, s) {
			n.t.Errorf("NDJSON contains forbidden substring %q", s)
		}
	}
}

func (n *naCtx) assertNoFileMatching(match func(string) bool, label string) {
	n.t.Helper()
	_ = filepath.WalkDir(n.tempDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if match(strings.ToLower(d.Name())) {
			n.t.Errorf("unexpected %s file at %s", label, p)
		}
		return nil
	})
}

type naExp struct {
	allowed bool
	guardID string
	t12     bool
	inert   bool
}

func assertWorkerNARecord(t *testing.T, rec telemetry.Record, exp naExp) {
	t.Helper()
	if rec.SchemaVersion != 1 || rec.Mode != "dry-run" || rec.Source != "worker" ||
		rec.SourceFile != "pkg/worker/command_guard_hook.go" ||
		rec.SourceFunction != "workerCommandGuardCheck" {
		t.Errorf("header mismatch: %+v", rec)
	}
	if !rec.NoSecretRawArgs || len([]rune(rec.CommandPreview)) > 120 {
		t.Errorf("invariant: nsra=%v preview=%d", rec.NoSecretRawArgs, len([]rune(rec.CommandPreview)))
	}
	if rec.DecisionAllowed != exp.allowed || rec.WouldBlockInEnforce == rec.DecisionAllowed {
		t.Errorf("decision=%v would_block=%v want %v", rec.DecisionAllowed, rec.WouldBlockInEnforce, exp.allowed)
	}
	if rec.GuardID != exp.guardID || rec.T12FailClosed != exp.t12 || rec.M3M4Inert != exp.inert {
		t.Errorf("guard=%q t12=%v inert=%v want %+v", rec.GuardID, rec.T12FailClosed, rec.M3M4Inert, exp)
	}
}

// WN-01: startup-adjacent boundary safe — cmd.Process/State nil pre and post hook.
func TestN_Alpha_0_Worker_StartupAdjacentBoundarySafe(t *testing.T) {
	n := setupWorkerNADryRun(t)
	cmd := n.track(mkCmd("git", "status", "-sb"))
	if cmd.Process != nil || cmd.ProcessState != nil {
		t.Errorf("cmd Process/State must be nil before hook")
	}
	if err := workerCommandGuardCheck(cmd, "", ""); err != nil {
		t.Fatalf("safe must not block: %v", err)
	}
	assertWorkerNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// WN-02: command boundary safe.
func TestN_Alpha_0_Worker_CommandBoundarySafe(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "status")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	assertWorkerNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// WN-03: command boundary dangerous.
func TestN_Alpha_0_Worker_CommandBoundaryDangerous(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "add", ".")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	assertWorkerNARecord(t, n.mustOneRecord(), naExp{allowed: false, guardID: "M5", inert: true})
}

// WN-04: provider CLI non-entry — input is git, not provider binary.
func TestN_Alpha_0_Worker_ProviderCLINonEntry(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "log")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	assertWorkerNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// WN-05: PID lock non-entry — no .pid file in t.TempDir().
func TestN_Alpha_0_Worker_PIDLockNonEntryAssertion(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "status", "-sb")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	_ = n.mustOneRecord()
	n.assertNoFileMatching(func(nm string) bool { return strings.HasSuffix(nm, ".pid") }, ".pid")
}

// WN-06: a2a server non-entry — no .sock or a2a-named file in t.TempDir().
func TestN_Alpha_0_Worker_A2AServerNonEntryAssertion(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "status", "-sb")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	_ = n.mustOneRecord()
	n.assertNoFileMatching(func(nm string) bool {
		return strings.HasSuffix(nm, ".sock") || strings.Contains(nm, "a2a")
	}, "a2a/socket")
}

// WN-07: T12 fail-closed dangerous Makefile recipe.
func TestN_Alpha_0_Worker_T12FailClosedSample(t *testing.T) {
	n := setupWorkerNADryRun(t)
	mkDir := writeTempMakefile(t, "Makefile", "release:\n\tgit push origin main\n")
	if err := workerCommandGuardCheck(n.track(mkMakeCmd(mkDir, "release")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := n.mustOneRecord()
	assertWorkerNARecord(t, rec, naExp{allowed: false, guardID: "M6", t12: true, inert: true})
	if rec.MakeTarget != "release" || rec.MakefileStatus != "Makefile" {
		t.Errorf("T12 fields: target=%q status=%q", rec.MakeTarget, rec.MakefileStatus)
	}
}

// WN-08: DefaultDir absolute path has t.TempDir() prefix — verified inside checkInvariants.
func TestN_Alpha_0_Worker_DefaultDirTelemetryWriterTempDirSample(t *testing.T) {
	n := setupWorkerNADryRun(t)
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "status", "-sb")), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	_ = n.mustOneRecord()
}

// WN-09: raw secret/home/env leak 0건 — token + Windows home + env value substrings absent.
func TestN_Alpha_0_Worker_RawSecretHomeEnvLeakZero(t *testing.T) {
	n := setupWorkerNADryRun(t)
	secret := "ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII"
	user := "shin0720unique"
	env := "verysecretval123"
	input := "https://oauth2:" + secret + `@example/C:\Users\` + user + `\repo MY_TOKEN=` + env
	if err := workerCommandGuardCheck(n.track(mkCmd("git", "push", input)), "", ""); err != nil {
		t.Fatalf("dry-run must not block: %v", err)
	}
	rec := n.mustOneRecord()
	if !rec.Redacted {
		t.Errorf("redacted must be true for token+home+env input")
	}
	n.assertNoSubstring(secret, user, env)
}
