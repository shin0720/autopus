package orchestra

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

// naCtx exercises the production bootstrap chain in a CWD-isolated tempDir for
// N-α-0 safety shim. names passed to newCommand are tracked for provider CLI
// non-entry assertion. The returned command is never Start/Wait-ed.
type naCtx struct {
	t          *testing.T
	tempDir    string
	absTempDir string
	origCWD    string
	names      []string
}

var nAlphaProviderBinaries = []string{"claude", "gemini", "codex", "opencode"}

func newOrchestraNACtx(t *testing.T) *naCtx {
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

func setupOrchestraNADryRun(t *testing.T) *naCtx {
	restore := setCommandGuardHookModeForTest(guardModeDryRun)
	t.Cleanup(restore)
	return newOrchestraNACtx(t)
}

// invoke creates a command via newCommand (which fires the hook + emit) and
// tracks the name. The returned command is discarded — never Start/Wait-ed.
func (n *naCtx) invoke(name string, args ...string) {
	n.t.Helper()
	n.names = append(n.names, name)
	_ = newCommand(context.Background(), name, args...)
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
	for i, name := range n.names {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(name), ".exe"))
		for _, p := range nAlphaProviderBinaries {
			if base == p {
				n.t.Errorf("name[%d] %q is a provider binary", i, base)
			}
		}
	}
	absDefault, err := filepath.Abs(telemetry.DefaultDir())
	if err != nil {
		n.t.Errorf("abs(DefaultDir): %v", err)
	} else if !strings.HasPrefix(absDefault, n.absTempDir) {
		n.t.Errorf("DefaultDir %q lacks tempDir prefix %q", absDefault, n.absTempDir)
	}
	_ = filepath.WalkDir(n.tempDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		nm := strings.ToLower(d.Name())
		if strings.HasSuffix(nm, ".pid") || strings.HasSuffix(nm, ".sock") || strings.Contains(nm, "a2a") {
			n.t.Errorf("unexpected boundary-violating file at %s", p)
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

func assertOrchestraNARecord(t *testing.T, rec telemetry.Record, exp naExp) {
	t.Helper()
	if rec.SchemaVersion != 1 || rec.Mode != "dry-run" || rec.Source != "orchestra" ||
		rec.SourceFile != "pkg/orchestra/command_guard_hook.go" ||
		rec.SourceFunction != "commandGuardCheck" {
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

// ON-01: orchestra startup-adjacent boundary safe.
func TestN_Alpha_0_Orchestra_StartupAdjacentBoundarySafe(t *testing.T) {
	n := setupOrchestraNADryRun(t)
	n.invoke("git", "status", "-sb")
	assertOrchestraNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// ON-02: orchestra command boundary safe.
func TestN_Alpha_0_Orchestra_CommandBoundarySafe(t *testing.T) {
	n := setupOrchestraNADryRun(t)
	n.invoke("git", "status")
	assertOrchestraNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// ON-03: orchestra command boundary dangerous.
func TestN_Alpha_0_Orchestra_CommandBoundaryDangerous(t *testing.T) {
	n := setupOrchestraNADryRun(t)
	n.invoke("git", "add", ".")
	assertOrchestraNARecord(t, n.mustOneRecord(), naExp{allowed: false, guardID: "M5", inert: true})
}

// ON-04: orchestra provider CLI non-entry.
func TestN_Alpha_0_Orchestra_ProviderCLINonEntry(t *testing.T) {
	n := setupOrchestraNADryRun(t)
	n.invoke("git", "log")
	assertOrchestraNARecord(t, n.mustOneRecord(), naExp{allowed: true, guardID: "P8a", inert: true})
}

// ON-05: orchestra DefaultDir / telemetry writer t.TempDir prefix.
func TestN_Alpha_0_Orchestra_DefaultDirTelemetryWriterTempDirSample(t *testing.T) {
	n := setupOrchestraNADryRun(t)
	n.invoke("git", "status", "-sb")
	_ = n.mustOneRecord()
}
