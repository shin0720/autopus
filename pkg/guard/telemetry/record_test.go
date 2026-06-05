package telemetry

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shin0720/auto-adk/pkg/guard"
)

func baseInput() BuildInput {
	return BuildInput{
		Mode:              ModeDryRun,
		Source:            "worker",
		Provider:          "claude",
		NormalizedCommand: "git status",
		CommandPreviewRaw: "git status -sb",
		Decision: guard.CommandGuardDecision{
			Phase: guard.PhaseAllow, Allowed: true, Reason: "ok",
		},
		SourceFile:     "pkg/worker/command_guard_hook.go",
		SourceFunction: "workerCommandGuardCheck",
	}
}

func TestBuild_RequiredFields(t *testing.T) {
	rec, ok := Build(baseInput())
	if !ok {
		t.Fatal("Build must succeed for safe input")
	}
	if rec.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version=%d want %d", rec.SchemaVersion, SchemaVersion)
	}
	if !rec.NoSecretRawArgs {
		t.Errorf("no_secret_raw_args must be true on every emitted record")
	}
	if _, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err != nil {
		t.Errorf("timestamp not RFC3339Nano: %v", err)
	}
	if rec.RunID == "" {
		t.Errorf("run_id must be non-empty")
	}
	if rec.GuardID != "P8a" {
		t.Errorf("guard_id for PhaseAllow=%q want P8a", rec.GuardID)
	}
}

func TestBuild_WouldBlockInEnforceFollowsDecision(t *testing.T) {
	in := baseInput()
	in.Decision = guard.CommandGuardDecision{Phase: guard.PhaseGitGate, Allowed: false, Reason: "git_dangerous"}
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if !rec.WouldBlockInEnforce {
		t.Errorf("would_block_in_enforce must be true when Allowed=false")
	}
	if rec.GuardID != "M5" {
		t.Errorf("guard_id for PhaseGitGate=%q want M5", rec.GuardID)
	}
}

func TestBuild_RedactionFailureReturnsNotOk(t *testing.T) {
	redactionFailureProbeForTest = true
	defer func() { redactionFailureProbeForTest = false }()
	_, ok := Build(baseInput())
	if ok {
		t.Error("Build must return ok=false when redaction fails")
	}
}

func TestBuild_PreviewIsRedacted(t *testing.T) {
	in := baseInput()
	in.CommandPreviewRaw = "deploy AKIAABCDEFGHIJKLMNOP --target prod"
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if strings.Contains(rec.CommandPreview, "AKIAABCDEFGHIJKLMNOP") {
		t.Errorf("AWS key must be redacted in preview, got %q", rec.CommandPreview)
	}
	if !rec.Redacted {
		t.Errorf("redacted flag must be true when content changed")
	}
}

func TestValidateRequired_RejectsMissingFields(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Record)
	}{
		{"schema_version", func(r *Record) { r.SchemaVersion = 0 }},
		{"timestamp", func(r *Record) { r.Timestamp = "" }},
		{"run_id", func(r *Record) { r.RunID = "" }},
		{"source", func(r *Record) { r.Source = "" }},
		{"mode", func(r *Record) { r.Mode = "" }},
		{"guard_id", func(r *Record) { r.GuardID = "" }},
		{"source_file", func(r *Record) { r.SourceFile = "" }},
		{"source_function", func(r *Record) { r.SourceFunction = "" }},
		{"no_secret_raw_args", func(r *Record) { r.NoSecretRawArgs = false }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := Build(baseInput())
			if !ok {
				t.Fatal("Build must succeed")
			}
			tc.mut(&rec)
			if err := rec.ValidateRequired(); err == nil {
				t.Errorf("ValidateRequired must reject missing %s", tc.name)
			}
		})
	}
}

func TestGuardIDFromPhase_AllPhases(t *testing.T) {
	cases := map[guard.CommandGuardPhase]string{
		guard.PhaseScriptInspector: "M6",
		guard.PhaseNonStructured:   "T02",
		guard.PhaseDenylist:        "M2",
		guard.PhaseGitGate:         "M5",
		guard.PhaseProfile:         "M3",
		guard.PhaseProviderBinding: "M4",
		guard.PhaseSubagent:        "M7",
		guard.PhaseEgress:          "M8",
		guard.PhaseAllow:           "P8a",
	}
	for ph, want := range cases {
		if got := GuardIDFromPhase(ph); got != want {
			t.Errorf("GuardIDFromPhase(%q)=%q want %q", ph, got, want)
		}
	}
}

func TestGuardIDFromPhase_UnknownFallsBackToP8a(t *testing.T) {
	if got := GuardIDFromPhase(guard.CommandGuardPhase("unknown")); got != "P8a" {
		t.Errorf("unknown phase must fall back to P8a, got %q", got)
	}
}

func TestRecord_JSONShapeContainsAllFields(t *testing.T) {
	rec, ok := Build(baseInput())
	if !ok {
		t.Fatal("Build must succeed")
	}
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	required := []string{
		"schema_version", "timestamp", "run_id", "session_id", "source",
		"provider", "worker_id", "command_preview", "normalized_command",
		"mode", "decision_allowed", "reason_code", "guard_id", "matched_rule",
		"profile_id", "provider_id", "make_target", "makefile_status",
		"t12_fail_closed", "m3_m4_inert", "would_block_in_enforce",
		"redacted", "no_secret_raw_args", "source_file", "source_function",
	}
	js := string(b)
	for _, k := range required {
		if !strings.Contains(js, `"`+k+`":`) {
			t.Errorf("JSON missing required field %q: %s", k, js)
		}
	}
}

func TestBuild_NormalizedCommandRedacted(t *testing.T) {
	in := baseInput()
	secret := "ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII"
	in.NormalizedCommand = "git push " + secret
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if strings.Contains(rec.NormalizedCommand, secret) {
		t.Errorf("normalized_command must be redacted, got %q", rec.NormalizedCommand)
	}
	if !rec.Redacted {
		t.Error("redacted flag must be true when normalized_command was redacted")
	}
}

func TestBuild_ReasonCodeRedacted(t *testing.T) {
	in := baseInput()
	secret := "AKIAABCDEFGHIJKLMNOP"
	in.Decision.Reason = "deny because of token " + secret
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if strings.Contains(rec.ReasonCode, secret) {
		t.Errorf("reason_code must be redacted, got %q", rec.ReasonCode)
	}
	if !rec.Redacted {
		t.Error("redacted flag must be true when reason_code was redacted")
	}
}

func TestBuild_MatchedRuleRedacted(t *testing.T) {
	in := baseInput()
	secret := "sk-abcdef0123456789ABCDEFXYZ"
	in.Decision.MatchedRule = "rule with token " + secret
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if strings.Contains(rec.MatchedRule, secret) {
		t.Errorf("matched_rule must be redacted, got %q", rec.MatchedRule)
	}
	if !rec.Redacted {
		t.Error("redacted flag must be true when matched_rule was redacted")
	}
}

func TestBuild_RedactedFlagAccumulatesAcrossFields(t *testing.T) {
	in := baseInput()
	in.CommandPreviewRaw = "git status -sb"
	in.NormalizedCommand = "git push glpat-ABCDEFGHIJ1234567890"
	rec, ok := Build(in)
	if !ok {
		t.Fatal("Build must succeed")
	}
	if !rec.Redacted {
		t.Error("redacted flag must accumulate (true when any field is redacted)")
	}
	if strings.Contains(rec.NormalizedCommand, "glpat-ABCDEFGHIJ1234567890") {
		t.Errorf("normalized_command must be redacted, got %q", rec.NormalizedCommand)
	}
}

func TestBuild_AllCleanFieldsRedactedFalse(t *testing.T) {
	rec, ok := Build(baseInput())
	if !ok {
		t.Fatal("Build must succeed")
	}
	if rec.Redacted {
		t.Error("redacted flag must be false when no field needed redaction")
	}
}

func TestGetRunID_StableAcrossCalls(t *testing.T) {
	a := GetRunID()
	b := GetRunID()
	if a != b {
		t.Errorf("run_id must be stable per process, got %q vs %q", a, b)
	}
	if a == "" {
		t.Errorf("run_id must be non-empty")
	}
}
