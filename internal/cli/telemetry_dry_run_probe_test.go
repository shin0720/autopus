package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gt "github.com/shin0720/auto-adk/pkg/guard/telemetry"
)

func TestTelemetryDryRunProbe_CommandShape(t *testing.T) {
	cmd := newTelemetryDryRunProbeCmd()
	if cmd.Use != "dry-run-probe" {
		t.Errorf("Use=%q want dry-run-probe", cmd.Use)
	}
	if cmd.Short == "" {
		t.Errorf("Short must not be empty")
	}
	if !cmd.Flags().HasFlags() {
		t.Errorf("must declare flags (--max-records, --dir, --json-summary)")
	}
}

func TestTelemetryDryRunProbe_MaxRecordsOutOfRange(t *testing.T) {
	var buf bytes.Buffer
	if err := runTelemetryDryRunProbe(&buf, probeFlags{maxRecords: 5, dir: t.TempDir()}); err == nil {
		t.Error("max-records=5 must be rejected")
	}
	buf.Reset()
	if err := runTelemetryDryRunProbe(&buf, probeFlags{maxRecords: 100, dir: t.TempDir()}); err == nil {
		t.Error("max-records=100 must be rejected")
	}
}

func TestTelemetryDryRunProbe_NoProviderCLIInSource(t *testing.T) {
	data, err := os.ReadFile("telemetry_dry_run_probe.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(data)
	for _, p := range []string{`"claude"`, `"gemini"`, `"codex"`, `"opencode"`} {
		if strings.Contains(s, p) {
			t.Errorf("probe source must not reference provider binary %s", p)
		}
	}
}

func TestTelemetryDryRunProbe_NoStartRunInSource(t *testing.T) {
	data, err := os.ReadFile("telemetry_dry_run_probe.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(data)
	for _, c := range []string{".Start()", ".Run()", ".Output()", ".CombinedOutput()", "exec.Command"} {
		if strings.Contains(s, c) {
			t.Errorf("probe source must not contain %q", c)
		}
	}
}

func runProbeInTempDir(t *testing.T, max int, asJSON bool) (string, []gt.Record, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	prev := gt.GetEmitter()
	t.Cleanup(func() { gt.SetEmitter(prev) })
	var buf bytes.Buffer
	if err := runTelemetryDryRunProbe(&buf, probeFlags{maxRecords: max, dir: dir, jsonSummary: asJSON}); err != nil {
		t.Fatalf("probe: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	var content string
	var recs []gt.Record
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".ndjson") {
			continue
		}
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		content += string(b)
	}
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		if line == "" {
			continue
		}
		var rec gt.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		recs = append(recs, rec)
	}
	return content, recs, &buf
}

func TestTelemetryDryRunProbe_WorkspaceTelemetryOutput(t *testing.T) {
	_, recs, _ := runProbeInTempDir(t, 16, false)
	if len(recs) != 14 {
		t.Fatalf("expected 14 records, got %d", len(recs))
	}
	var worker, orch, allow, deny, t12, redacted, inert int
	for _, rec := range recs {
		if rec.SchemaVersion != 1 || rec.Mode != "dry-run" || !rec.NoSecretRawArgs {
			t.Errorf("invariant: schema=%d mode=%q nsra=%v", rec.SchemaVersion, rec.Mode, rec.NoSecretRawArgs)
		}
		if rec.Source == "worker" {
			worker++
		} else if rec.Source == "orchestra" {
			orch++
		}
		if rec.DecisionAllowed {
			allow++
		} else {
			deny++
		}
		if rec.T12FailClosed {
			t12++
		}
		if rec.Redacted {
			redacted++
		}
		if rec.M3M4Inert {
			inert++
		}
	}
	if worker != 8 || orch != 6 {
		t.Errorf("worker/orch: %d/%d want 8/6", worker, orch)
	}
	if allow < 4 || deny < 4 {
		t.Errorf("allow/deny: %d/%d need >=4 each", allow, deny)
	}
	if t12 < 1 {
		t.Errorf("t12: %d need >=1", t12)
	}
	if redacted < 2 {
		t.Errorf("redacted: %d need >=2", redacted)
	}
	if inert != 14 {
		t.Errorf("m3_m4_inert: %d need 14", inert)
	}
}

func TestTelemetryDryRunProbe_RawLeakZero(t *testing.T) {
	content, _, buf := runProbeInTempDir(t, 16, false)
	for _, sub := range probeForbidden {
		if strings.Contains(content, sub) {
			t.Errorf("NDJSON contains forbidden substring %q", sub)
		}
	}
	if !strings.Contains(buf.String(), "PASS") {
		t.Errorf("summary must contain raw leak check PASS, got: %s", buf.String())
	}
}

func TestTelemetryDryRunProbe_RedactedSummaryStdout(t *testing.T) {
	_, _, buf := runProbeInTempDir(t, 16, false)
	out := buf.String()
	for _, sub := range []string{`"schema_version":`, `"command_preview":`, `"normalized_command":`} {
		if strings.Contains(out, sub) {
			t.Errorf("stdout must not contain raw NDJSON field %q, got: %s", sub, out)
		}
	}
	for _, h := range []string{"total records", "allow/deny", "raw leak check"} {
		if !strings.Contains(out, h) {
			t.Errorf("summary must include %q", h)
		}
	}
}

func TestTelemetryDryRunProbe_JSONSummary(t *testing.T) {
	_, _, buf := runProbeInTempDir(t, 14, true)
	var sum probeSummary
	if err := json.Unmarshal(buf.Bytes(), &sum); err != nil {
		t.Errorf("json unmarshal: %v\nout: %s", err, buf.String())
	}
	if sum.TotalRecords != 14 {
		t.Errorf("json summary total=%d want 14", sum.TotalRecords)
	}
	if sum.RawLeakCheck != "PASS" {
		t.Errorf("raw_leak_check=%q want PASS", sum.RawLeakCheck)
	}
}

func TestTelemetryDryRunProbe_GitignoreAppliesToTelemetryDir(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".gitignore"))
	if err != nil {
		t.Skipf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), ".autopus/telemetry/") {
		t.Errorf(".gitignore must contain `.autopus/telemetry/` rule")
	}
}

func TestTelemetryDryRunProbe_RegisteredUnderTelemetry(t *testing.T) {
	root := newTelemetryCmd()
	for _, sub := range root.Commands() {
		if sub.Use == "dry-run-probe" {
			return
		}
	}
	t.Errorf("dry-run-probe must be registered as a subcommand of telemetry")
}
