package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shin0720/auto-adk/pkg/guard"
)

func validRecordForWrite() Record {
	rec, ok := Build(BuildInput{
		Mode:              ModeDryRun,
		Source:            "worker",
		NormalizedCommand: "git status",
		CommandPreviewRaw: "git status -sb",
		Decision: guard.CommandGuardDecision{
			Phase: guard.PhaseAllow, Allowed: true, Reason: "ok",
		},
		SourceFile:     "pkg/guard/telemetry/writer_test.go",
		SourceFunction: "validRecordForWrite",
	})
	if !ok {
		panic("Build failed in test fixture")
	}
	return rec
}

func TestWriter_AppendCreatesFile(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	if !w.Append(validRecordForWrite()) {
		t.Fatal("Append failed")
	}
	today := time.Now().UTC().Format("2006-01-02")
	p := filepath.Join(dir, today+".ndjson")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Errorf("file must end with newline: %q", string(data))
	}
}

func TestWriter_AppendMultipleLines(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	for i := 0; i < 3; i++ {
		if !w.Append(validRecordForWrite()) {
			t.Fatalf("Append %d failed", i)
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	f, err := os.Open(filepath.Join(dir, today+".ndjson"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	count := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var rec Record
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Errorf("unmarshal line %d: %v", count, err)
		}
		if rec.SchemaVersion != SchemaVersion {
			t.Errorf("line %d schema_version=%d", count, rec.SchemaVersion)
		}
		count++
	}
	if count != 3 {
		t.Errorf("want 3 lines, got %d", count)
	}
}

func TestWriter_ValidationFailureIncrementsCounter(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	bad := validRecordForWrite()
	bad.NoSecretRawArgs = false // violates invariant
	if w.Append(bad) {
		t.Error("Append must return false for invalid record")
	}
	if w.WriteErrors() == 0 {
		t.Error("write error counter must increment on validation failure")
	}
	// no file should be created
	today := time.Now().UTC().Format("2006-01-02")
	if _, err := os.Stat(filepath.Join(dir, today+".ndjson")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("no file expected on validation failure")
	}
}

func TestWriter_RotateOnSizeOverflow(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	w.SetMaxFileBytes(200) // tiny: one line will already approach the limit
	for i := 0; i < 6; i++ {
		if !w.Append(validRecordForWrite()) {
			t.Fatalf("Append %d failed", i)
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	base := filepath.Join(dir, today+".ndjson")
	if _, err := os.Stat(base); err != nil {
		t.Errorf("base file missing: %v", err)
	}
	// at least one rotated file must exist
	found := false
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), today+"-") && strings.HasSuffix(e.Name(), ".ndjson") {
			found = true
		}
	}
	if !found {
		t.Error("rotate did not produce a suffixed file")
	}
}

func TestWriter_RetentionRemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	// create a file 10 days old by filename date
	oldDate := time.Now().UTC().AddDate(0, 0, -10).Format("2006-01-02")
	oldPath := filepath.Join(dir, oldDate+".ndjson")
	if err := os.WriteFile(oldPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// a recent file (2 days old)
	recentDate := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")
	recentPath := filepath.Join(dir, recentDate+".ndjson")
	if err := os.WriteFile(recentPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	w := NewWriter(dir)
	if !w.Append(validRecordForWrite()) {
		t.Fatal("Append failed")
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("old file (10d) must be swept, still present: %v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Errorf("recent file (2d) must remain: %v", err)
	}
}

func TestWriter_RetentionIgnoresNonNDJSON(t *testing.T) {
	dir := t.TempDir()
	// a file with old date but not NDJSON
	oldDate := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	keepPath := filepath.Join(dir, oldDate+".txt")
	if err := os.WriteFile(keepPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	w := NewWriter(dir)
	if !w.Append(validRecordForWrite()) {
		t.Fatal("Append failed")
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Errorf("non-.ndjson must be untouched: %v", err)
	}
}

func TestWriter_MkdirAllowsDeepDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "deep", "nested", "telemetry")
	w := NewWriter(dir)
	if !w.Append(validRecordForWrite()) {
		t.Fatal("Append failed")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("dir must be created: %v", err)
	}
}

// --- package-level Emit / Emitter -----------------------------------------

func TestEmit_NoEmitterReturnsFalse(t *testing.T) {
	SetEmitter(nil)
	if Emit(validRecordForWrite()) {
		t.Error("Emit must return false when no emitter installed")
	}
}

func TestEmit_ForwardsToInstalledEmitter(t *testing.T) {
	cap := &CaptureEmitter{}
	SetEmitter(cap)
	t.Cleanup(func() { SetEmitter(nil) })
	if !Emit(validRecordForWrite()) {
		t.Error("Emit must return true for installed emitter accepting record")
	}
	if cap.Len() != 1 {
		t.Errorf("capture len=%d want 1", cap.Len())
	}
}

func TestCaptureEmitter_RejectsInvalidRecord(t *testing.T) {
	cap := &CaptureEmitter{}
	bad := validRecordForWrite()
	bad.GuardID = ""
	if cap.Append(bad) {
		t.Error("CaptureEmitter must reject invalid record")
	}
	if cap.Rejected() == 0 {
		t.Error("rejected counter must increment")
	}
	if cap.Len() != 0 {
		t.Errorf("captured len=%d want 0", cap.Len())
	}
}

func TestSetEmitter_Roundtrip(t *testing.T) {
	original := GetEmitter()
	t.Cleanup(func() { SetEmitter(original) })

	cap := &CaptureEmitter{}
	SetEmitter(cap)
	if GetEmitter() != cap {
		t.Error("GetEmitter must return the installed emitter")
	}
	SetEmitter(nil)
	if GetEmitter() != nil {
		t.Error("SetEmitter(nil) must clear the emitter")
	}
}

func TestDefaultDir_PointsAtGitignoredLocation(t *testing.T) {
	got := DefaultDir()
	if !strings.HasPrefix(filepath.ToSlash(got), ".autopus/telemetry/") {
		t.Errorf("DefaultDir must live under .autopus/telemetry/, got %q", got)
	}
}
