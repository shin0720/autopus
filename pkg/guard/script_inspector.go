// Package guard: SB8 P5b M6 script inspector.
//
// Decision layer ONLY, raw shell/script string based. It denies dangerous
// script STRUCTURES — PowerShell ExecutionPolicy Bypass, install.ps1 execution,
// and download-pipe-execute (iwr/irm/curl/wget | iex/bash/sh) — by reusing M1
// ContainsPipe/DetectShellMetacharacters and the M2 ReDoS-safe denylist engine.
// It NEVER executes powershell/pwsh/cmd/sh/bash/iwr/irm/curl/wget, never reads
// or edits install.ps1, does NOT implement network egress (M8), and does NOT
// integrate any runner hook (M9). Structured command-category gating lives in
// P5a M5 (git_gate) and is not duplicated here. A final "allow" is M3/M4/M5/M9's
// call: a non-dangerous string is reported neutral / allow candidate.
package guard

import "strings"

// ScriptInspectionDecision is the result of inspecting a raw script string.
type ScriptInspectionDecision struct {
	Allowed      bool   // false => dangerous structure denied
	RiskCategory string // malformed | powershell_bypass | install_script | pipe_execution | none
	MatchedRule  string // denied pattern when denied
	Reason       string
}

// ScriptInspectionRule groups the deny pattern sets the inspector applies.
type ScriptInspectionRule struct {
	PowerShellBypass []string
	InstallScript    []string
	PipeExecution    []string
}

// DefaultScriptInspectionRule returns the SB8 raw-script deny patterns. All are
// compiled and matched through the M2 ReDoS-safe engine (RE2 + 100ms timeout).
func DefaultScriptInspectionRule() ScriptInspectionRule {
	return ScriptInspectionRule{
		PowerShellBypass: []string{
			`(?i)(powershell|pwsh)(\.exe)?\s+.*-executionpolicy\s+bypass`,
			`(?i)(powershell|pwsh)(\.exe)?\s+.*-ep\s+bypass`,
		},
		InstallScript: []string{`(?i)(^|[\s/\\])install\.ps1\b`},
		PipeExecution: []string{`(?i)(iwr|irm|curl|wget|invoke-webrequest|invoke-restmethod)\s+.+\|\s*(iex|bash|sh|invoke-expression)`},
	}
}

// isMalformedScript flags NUL and disallowed control characters (tab/newline/CR
// are allowed). Such input is fail-closed denied before any pattern match.
func isMalformedScript(s string) bool {
	for _, r := range s {
		if r == 0 {
			return true
		}
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// denyScan applies the deny patterns through M2 EvaluateDenylist. The first
// token seeds AllowedCommands so the M2 allow baseline always passes, leaving
// only the deny patterns to decide. A baseline rejection (empty MatchedPattern)
// is NOT treated as a match.
func denyScan(input string, patterns []string) (bool, string) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return false, ""
	}
	rs := DenyRuleSet{
		AllowedCommands: []string{fields[0]},
		DeniedRegex:     patterns,
	}
	d := EvaluateDenylist(input, rs)
	if !d.Allowed && d.MatchedPattern != "" {
		return true, d.MatchedPattern
	}
	return false, ""
}

// HasPowerShellBypass reports whether s contains a PowerShell ExecutionPolicy
// Bypass invocation.
func HasPowerShellBypass(s string) (bool, string) {
	return denyScan(s, DefaultScriptInspectionRule().PowerShellBypass)
}

// HasInstallScriptExecution reports whether s executes install.ps1.
func HasInstallScriptExecution(s string) (bool, string) {
	return denyScan(s, DefaultScriptInspectionRule().InstallScript)
}

// HasPipeExecution reports whether s is a download-pipe-execute. It first uses
// M1 ContainsPipe as a cheap gate before the M2 regex.
func HasPipeExecution(s string) (bool, string) {
	if !ContainsPipe(s) {
		return false, ""
	}
	return denyScan(s, DefaultScriptInspectionRule().PipeExecution)
}

// ClassifyScriptRisk returns the risk category of a raw script string.
func ClassifyScriptRisk(s string) string {
	return InspectScriptString(s).RiskCategory
}

// InspectScriptString inspects a raw script string for dangerous structures.
// Malformed input fails closed. PowerShell bypass / install.ps1 / pipe-execute
// are denied. Otherwise the string is neutral (allow candidate; the final allow
// is decided by the profile/provider/git gate, not here).
func InspectScriptString(s string) ScriptInspectionDecision {
	if isMalformedScript(s) {
		return ScriptInspectionDecision{Allowed: false, RiskCategory: "malformed", Reason: "malformed shell string (NUL/control char) — fail-closed"}
	}
	if ok, p := HasPowerShellBypass(s); ok {
		return ScriptInspectionDecision{Allowed: false, RiskCategory: "powershell_bypass", MatchedRule: p, Reason: "PowerShell ExecutionPolicy Bypass denied"}
	}
	if ok, p := HasInstallScriptExecution(s); ok {
		return ScriptInspectionDecision{Allowed: false, RiskCategory: "install_script", MatchedRule: p, Reason: "install.ps1 execution denied"}
	}
	if ok, p := HasPipeExecution(s); ok {
		return ScriptInspectionDecision{Allowed: false, RiskCategory: "pipe_execution", MatchedRule: p, Reason: "download-pipe-execute pattern denied"}
	}

	reason := "no dangerous script structure (out of P5b scope; allow candidate)"
	if metas := DetectShellMetacharacters(s); len(metas) > 0 {
		reason = "shell metacharacters " + strings.Join(metas, ",") + " present but no deny pattern (allow candidate; gate/profile decides)"
	}
	return ScriptInspectionDecision{Allowed: true, RiskCategory: "none", Reason: reason}
}
