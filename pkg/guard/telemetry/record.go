// Package telemetry: SB8 minimal dry-run telemetry record + builder.
//
// This package is a LOCAL evidence channel only — it performs no network call,
// no external upload, and no actual command execution. It is wired by the
// worker/orchestra command-guard hooks and never alters their decisions: the
// emit path is fire-and-forget (write failures and validation failures are
// silently counted, never returned to the caller).
//
// The default emitter is a no-op until SetEmitter is called explicitly. This
// keeps existing tests free of file-system side effects and lets production
// startup decide where (or whether) to persist records.
package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/shin0720/auto-adk/pkg/guard"
)

// SchemaVersion is the frozen schema id for SB8 dry-run telemetry v1.
const SchemaVersion = 1

// Mode is the canonical telemetry mode string for the "mode" field.
type Mode string

const (
	ModeDisabled Mode = "disabled"
	ModeDryRun   Mode = "dry-run"
	ModeEnforce  Mode = "enforce"
)

// Record is the v1 SB8 dry-run telemetry schema. Field order matches the SPEC
// minimum-field list; JSON tags use snake_case for NDJSON readability.
type Record struct {
	SchemaVersion       int    `json:"schema_version"`
	Timestamp           string `json:"timestamp"`
	RunID               string `json:"run_id"`
	SessionID           string `json:"session_id"`
	Source              string `json:"source"`
	Provider            string `json:"provider"`
	WorkerID            string `json:"worker_id"`
	CommandPreview      string `json:"command_preview"`
	NormalizedCommand   string `json:"normalized_command"`
	Mode                string `json:"mode"`
	DecisionAllowed     bool   `json:"decision_allowed"`
	ReasonCode          string `json:"reason_code"`
	GuardID             string `json:"guard_id"`
	MatchedRule         string `json:"matched_rule"`
	ProfileID           string `json:"profile_id"`
	ProviderID          string `json:"provider_id"`
	MakeTarget          string `json:"make_target"`
	MakefileStatus      string `json:"makefile_status"`
	T12FailClosed       bool   `json:"t12_fail_closed"`
	M3M4Inert           bool   `json:"m3_m4_inert"`
	WouldBlockInEnforce bool   `json:"would_block_in_enforce"`
	Redacted            bool   `json:"redacted"`
	NoSecretRawArgs     bool   `json:"no_secret_raw_args"`
	SourceFile          string `json:"source_file"`
	SourceFunction      string `json:"source_function"`
}

// ValidateRequired rejects records missing the SPEC-mandated minimum invariants.
// no_secret_raw_args MUST be true — false is treated as a redaction-policy bug
// and prevents the record from reaching the disk.
func (r *Record) ValidateRequired() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %d, got %d", SchemaVersion, r.SchemaVersion)
	}
	if r.Timestamp == "" {
		return fmt.Errorf("timestamp required")
	}
	if r.RunID == "" {
		return fmt.Errorf("run_id required")
	}
	if r.Source == "" {
		return fmt.Errorf("source required")
	}
	if r.Mode == "" {
		return fmt.Errorf("mode required")
	}
	if r.GuardID == "" {
		return fmt.Errorf("guard_id required")
	}
	if r.SourceFile == "" {
		return fmt.Errorf("source_file required")
	}
	if r.SourceFunction == "" {
		return fmt.Errorf("source_function required")
	}
	if !r.NoSecretRawArgs {
		return fmt.Errorf("no_secret_raw_args must be true")
	}
	return nil
}

// GuardIDFromPhase maps a CommandGuardPhase to a stable, human-readable guard
// id used in evidence aggregations. Unknown phases fall back to the facade
// label "P8a" so the record always carries a non-empty guard_id.
func GuardIDFromPhase(p guard.CommandGuardPhase) string {
	switch p {
	case guard.PhaseScriptInspector:
		return "M6"
	case guard.PhaseNonStructured:
		return "T02"
	case guard.PhaseDenylist:
		return "M2"
	case guard.PhaseGitGate:
		return "M5"
	case guard.PhaseProfile:
		return "M3"
	case guard.PhaseProviderBinding:
		return "M4"
	case guard.PhaseSubagent:
		return "M7"
	case guard.PhaseEgress:
		return "M8"
	case guard.PhaseAllow:
		return "P8a"
	}
	return "P8a"
}

// BuildInput collects all hook-side inputs required to build a Record. The
// caller (worker/orchestra hook) supplies raw fields; Build orchestrates the
// redaction step and the timestamp/run_id capture.
type BuildInput struct {
	Mode              Mode
	Source            string
	Provider          string
	WorkerID          string
	SessionID         string
	NormalizedCommand string
	CommandPreviewRaw string
	Decision          guard.CommandGuardDecision
	ProfileID         string
	ProviderID        string
	MakeTarget        string
	MakefileStatus    string
	T12FailClosed     bool
	M3M4Inert         bool
	SourceFile        string
	SourceFunction    string
}

// Build redacts the preview, the normalized command, the reason code, and the
// matched rule via RedactPreview before assembling a Record. Returns
// (zero, false) when any of the four redaction calls fails — the caller MUST
// treat that as emit-skip. The Redacted flag is the OR of the four per-field
// results, so a single redacted field marks the whole record as redacted.
func Build(in BuildInput) (Record, bool) {
	preview, redactedPreview, ok := RedactPreview(in.CommandPreviewRaw)
	if !ok {
		return Record{}, false
	}
	normalized, redactedNorm, ok := RedactPreview(in.NormalizedCommand)
	if !ok {
		return Record{}, false
	}
	reason, redactedReason, ok := RedactPreview(in.Decision.Reason)
	if !ok {
		return Record{}, false
	}
	matched, redactedMatched, ok := RedactPreview(in.Decision.MatchedRule)
	if !ok {
		return Record{}, false
	}
	rec := Record{
		SchemaVersion:       SchemaVersion,
		Timestamp:           time.Now().UTC().Format(time.RFC3339Nano),
		RunID:               GetRunID(),
		SessionID:           in.SessionID,
		Source:              in.Source,
		Provider:            in.Provider,
		WorkerID:            in.WorkerID,
		CommandPreview:      preview,
		NormalizedCommand:   normalized,
		Mode:                string(in.Mode),
		DecisionAllowed:     in.Decision.Allowed,
		ReasonCode:          reason,
		GuardID:             GuardIDFromPhase(in.Decision.Phase),
		MatchedRule:         matched,
		ProfileID:           in.ProfileID,
		ProviderID:          in.ProviderID,
		MakeTarget:          in.MakeTarget,
		MakefileStatus:      in.MakefileStatus,
		T12FailClosed:       in.T12FailClosed,
		M3M4Inert:           in.M3M4Inert,
		WouldBlockInEnforce: !in.Decision.Allowed,
		Redacted:            redactedPreview || redactedNorm || redactedReason || redactedMatched,
		NoSecretRawArgs:     true,
		SourceFile:          in.SourceFile,
		SourceFunction:      in.SourceFunction,
	}
	return rec, true
}

// run_id is generated lazily on first access and shared for the lifetime of
// the process. crypto/rand fallback to nanosecond timestamp keeps the
// invariant "non-empty" even on rare entropy errors.
var (
	runIDOnce sync.Once
	runIDVal  string
)

// GetRunID returns the per-process run id (initialized on first call).
func GetRunID() string {
	runIDOnce.Do(func() {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			runIDVal = fmt.Sprintf("run-%d", time.Now().UnixNano())
			return
		}
		runIDVal = hex.EncodeToString(b)
	})
	return runIDVal
}
