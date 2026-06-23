// Package guard: SB8 P2 M2 denylist + regex matcher.
//
// Wraps pkg/worker/security ValidateCommandSafe (ReDoS-safe, timeout-protected,
// compile-validated) — it does NOT reimplement the regex engine. The denylist
// input is the M1 NormalizedCommand.CompareString. SecurityPolicy is used here
// only for command allow/deny decisions; AllowFS/AllowedPaths/AllowNetwork are
// intentionally left to M5/M8.
package guard

import (
	"regexp"

	"github.com/shin0720/auto-adk/pkg/worker/security"
)

// DenyRuleSet describes the SB8 command policy for denylist evaluation.
type DenyRuleSet struct {
	AllowedCommands []string // ccp.allowed_commands (exact or prefix)
	DeniedExact     []string // literal forbidden commands
	DeniedRegex     []string // forbidden regex patterns
}

// DenyDecision is the result of evaluating a command against a DenyRuleSet.
type DenyDecision struct {
	Allowed        bool
	MatchedPattern string
	Reason         string
}

// deniedPatterns converts exact denies into anchored literal regexes and
// appends the regex denies, producing the SecurityPolicy.DeniedPatterns slice.
func (rs DenyRuleSet) deniedPatterns() []string {
	out := make([]string, 0, len(rs.DeniedExact)+len(rs.DeniedRegex))
	for _, e := range rs.DeniedExact {
		out = append(out, "^"+regexp.QuoteMeta(e)+`\b`)
	}
	out = append(out, rs.DeniedRegex...)
	return out
}

// BuildSecurityPolicyFromRules maps a DenyRuleSet to a security.SecurityPolicy
// using ONLY command allow/deny fields (AllowFS/AllowedPaths/AllowNetwork are
// left zero, reserved for M5/M8).
func BuildSecurityPolicyFromRules(rs DenyRuleSet) security.SecurityPolicy {
	return security.SecurityPolicy{
		AllowedCommands: rs.AllowedCommands,
		DeniedPatterns:  rs.deniedPatterns(),
	}
}

// ValidateDeniedPatterns reuses security pattern validation (compile + length).
func ValidateDeniedPatterns(rs DenyRuleSet) error {
	p := BuildSecurityPolicyFromRules(rs)
	return p.ValidateDeniedPatterns()
}

// EvaluateDenylist evaluates a normalized command string against the rule set.
// Deny takes precedence. matched_pattern is determined at the guard level by
// testing patterns individually (no reason-string parsing). ReDoS/timeout/
// compile safety is delegated to security.ValidateCommandSafe.
func EvaluateDenylist(input string, rs DenyRuleSet) DenyDecision {
	// fail-closed on malformed / oversized patterns
	if err := ValidateDeniedPatterns(rs); err != nil {
		return DenyDecision{Allowed: false, Reason: "invalid denied pattern (fail-closed): " + err.Error()}
	}

	// allow-list baseline (no denied patterns). Empty AllowedCommands => fail-closed.
	allowOnly := security.SecurityPolicy{AllowedCommands: rs.AllowedCommands}
	baseAllowed, baseReason := allowOnly.ValidateCommandSafe(input, "")
	if !baseAllowed {
		return DenyDecision{Allowed: false, Reason: baseReason}
	}

	// per-pattern evaluation surfaces the matched pattern at the guard level.
	for _, p := range rs.deniedPatterns() {
		single := security.SecurityPolicy{
			AllowedCommands: rs.AllowedCommands,
			DeniedPatterns:  []string{p},
		}
		ok, reason := single.ValidateCommandSafe(input, "")
		if !ok {
			return DenyDecision{Allowed: false, MatchedPattern: p, Reason: reason}
		}
	}
	return DenyDecision{Allowed: true}
}

// EvaluateCommand normalizes (M1) then evaluates. OriginalArgs is preserved by
// NormalizeCommand; CompareString is used as the denylist input.
func EvaluateCommand(executable string, args []string, rs DenyRuleSet) DenyDecision {
	nc := NormalizeCommand(executable, args)
	return EvaluateDenylist(nc.CompareString, rs)
}
