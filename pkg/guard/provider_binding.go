// Package guard: SB8 P4 M4 provider binding.
//
// Decision layer ONLY. It validates the providerID + profileID + executable
// allow relationship, then delegates the actual command allow/deny to M3
// EvaluateProfile (which in turn applies M2 denylist deny-first). It does NOT
// call adapter.ResolveBinary, does NOT import the adapter package, does NOT
// trust ProviderConfig.Binary, and does NOT use config / content Profile types.
// Provider identity is the YAML provider id (e.g. claude-code, codex, gemini-cli,
// opencode) — never adapter.Name() ("claude"/"codex"/"gemini").
package guard

import "strings"

// ProviderRule describes what a single provider id is permitted to do.
type ProviderRule struct {
	AllowedProfiles    []string // cli_command_profile_ref ids permitted for this provider
	AllowedExecutables []string // normalized executables permitted for this provider
}

// ProviderBindingSet maps a YAML provider id to its rule.
type ProviderBindingSet map[string]ProviderRule

// ProviderDecision is the result of evaluating a command under a provider+profile.
type ProviderDecision struct {
	Allowed     bool
	ProviderID  string
	ProfileID   string
	Executable  string // normalized executable (M1)
	MatchedRule string // allowed/denied rule surfaced by M3
	Reason      string
}

// MatchProviderExecutable reports whether the (already M1-normalized) executable
// is in the provider's allow list. Comparison is exact on normalized names.
func MatchProviderExecutable(normExec string, allowed []string) bool {
	for _, a := range allowed {
		if normExec == strings.TrimSpace(a) {
			return true
		}
	}
	return false
}

func providerAllowsProfile(rule ProviderRule, profileID string) bool {
	for _, p := range rule.AllowedProfiles {
		if p == profileID {
			return true
		}
	}
	return false
}

// EvaluateProviderBinding validates provider+profile+executable, then delegates
// the command decision to M3 EvaluateProfile (M2 denylist wins). Unknown
// provider, a profile not allowed for the provider, or an executable not allowed
// for the provider all fail closed. The denylist (via M3->M2) takes precedence
// over the provider allow list.
func EvaluateProviderBinding(providerID, profileID, executable string, args []string, pbs ProviderBindingSet, ps ProfileSet) ProviderDecision {
	rule, ok := pbs[providerID]
	if !ok {
		return ProviderDecision{Allowed: false, ProviderID: providerID, ProfileID: profileID, Reason: "unknown provider (fail-closed)"}
	}
	if !providerAllowsProfile(rule, profileID) {
		return ProviderDecision{Allowed: false, ProviderID: providerID, ProfileID: profileID, Reason: "profile not allowed for provider (fail-closed)"}
	}

	normExec := NormalizeExecutable(executable)
	if !MatchProviderExecutable(normExec, rule.AllowedExecutables) {
		return ProviderDecision{Allowed: false, ProviderID: providerID, ProfileID: profileID, Executable: normExec, Reason: "executable not allowed for provider (fail-closed)"}
	}

	pd := EvaluateProfile(profileID, executable, args, ps)
	if !pd.Allowed {
		return ProviderDecision{
			Allowed:     false,
			ProviderID:  providerID,
			ProfileID:   profileID,
			Executable:  normExec,
			MatchedRule: pd.MatchedRule,
			Reason:      pd.Reason,
		}
	}

	return ProviderDecision{
		Allowed:     true,
		ProviderID:  providerID,
		ProfileID:   profileID,
		Executable:  normExec,
		MatchedRule: pd.MatchedRule,
		Reason:      "allowed",
	}
}
