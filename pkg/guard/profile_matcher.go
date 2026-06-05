// Package guard: SB8 P3 M3 profile matcher.
//
// Selects a command profile (profileID -> DenyRuleSet) and delegates the actual
// allow/deny decision to M2 EvaluateDenylist (deny-first, fail-closed), which in
// turn reuses pkg/worker/security. No provider binding / ResolveBinary / adapter
// / config / content profile types are used here (reserved for M4+).
package guard

import "strings"

// ProfileSet maps a profile id (e.g. ccp_readonly) to its DenyRuleSet.
type ProfileSet map[string]DenyRuleSet

// ProfileDecision is the result of evaluating a command under a named profile.
type ProfileDecision struct {
	Allowed     bool
	ProfileID   string
	MatchedRule string // allowed rule when allowed; denied pattern when denied
	Reason      string
}

// MatchAllowedRule returns the allowed entry that matches input, using the same
// semantics as security (exact, or prefix followed by a space). "" if none.
func MatchAllowedRule(input string, allowed []string) string {
	for _, a := range allowed {
		t := strings.TrimSpace(a)
		if t == "" {
			continue
		}
		if input == t || strings.HasPrefix(input, t+" ") {
			return a
		}
	}
	return ""
}

// EvaluateProfile normalizes the command (M1), then applies the named profile's
// rules via M2 EvaluateDenylist. Deny takes precedence; unknown / empty profiles
// fail closed.
func EvaluateProfile(profileID, executable string, args []string, ps ProfileSet) ProfileDecision {
	rs, ok := ps[profileID]
	if !ok {
		return ProfileDecision{Allowed: false, ProfileID: profileID, Reason: "unknown profile (fail-closed)"}
	}
	if len(rs.AllowedCommands) == 0 {
		return ProfileDecision{Allowed: false, ProfileID: profileID, Reason: "empty profile: no allowed commands (fail-closed)"}
	}

	nc := NormalizeCommand(executable, args)
	d := EvaluateDenylist(nc.CompareString, rs)
	if !d.Allowed {
		return ProfileDecision{
			Allowed:     false,
			ProfileID:   profileID,
			MatchedRule: d.MatchedPattern,
			Reason:      d.Reason,
		}
	}

	return ProfileDecision{
		Allowed:     true,
		ProfileID:   profileID,
		MatchedRule: MatchAllowedRule(nc.CompareString, rs.AllowedCommands),
	}
}
