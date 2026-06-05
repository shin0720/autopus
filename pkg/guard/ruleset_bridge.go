// Package guard: SB8 ruleset bridge (pure helper).
//
// Converts already-decoded in-memory specs into a ProfileSet / ProviderBindingSet.
// PURE: it reads NO files, imports NO yaml package, calls NO yaml.Unmarshal /
// os.ReadFile, performs NO execution, and does NOT enable enforcement or change
// any hook default. The YAML->spec decode is a separate (later) step; this bridge
// only transforms structs the caller already holds. allowed_providers are routed
// through NormalizeProviderID. Validation failures fail closed via RulesetBridgeError.
package guard

import "strings"

// BridgeCommandProfileSpec is an in-memory representation of one cli_command_profile (ccp_*).
type BridgeCommandProfileSpec struct {
	ID              string
	AllowedCommands []string
	DeniedExact     []string
	DeniedRegex     []string
}

// BridgeWorkerCommandProfileSpec is an in-memory representation of one cli_worker_profile (cwp_W##).
type BridgeWorkerCommandProfileSpec struct {
	ID                 string
	AllowedProviders   []string // adapter or YAML provider ids (routed via NormalizeProviderID)
	CommandProfileRef  string   // cli_command_profile_ref; must match a BridgeCommandProfileSpec.ID
	AllowedExecutables []string
}

// RulesetBridgeSpec is the full in-memory spec the bridge transforms.
type RulesetBridgeSpec struct {
	CommandProfiles []BridgeCommandProfileSpec
	WorkerProfiles  []BridgeWorkerCommandProfileSpec
}

// RulesetBridgeError is a fail-closed validation error.
type RulesetBridgeError struct{ Reason string }

func (e *RulesetBridgeError) Error() string { return "ruleset bridge: " + e.Reason }

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// BuildProfileSet converts command-profile specs into a ProfileSet. An empty
// AllowedCommands is preserved (M3 fail-closes on it). Duplicate ids fail closed.
func BuildProfileSet(spec RulesetBridgeSpec) (ProfileSet, error) {
	ps := ProfileSet{}
	for _, cp := range spec.CommandProfiles {
		id := strings.TrimSpace(cp.ID)
		if id == "" {
			return nil, &RulesetBridgeError{Reason: "empty command profile id"}
		}
		if _, dup := ps[id]; dup {
			return nil, &RulesetBridgeError{Reason: "duplicate command profile id: " + id}
		}
		ps[id] = DenyRuleSet{
			AllowedCommands: cp.AllowedCommands,
			DeniedExact:     cp.DeniedExact,
			DeniedRegex:     cp.DeniedRegex,
		}
	}
	return ps, nil
}

// BuildProviderBindingSet converts worker-profile specs into a ProviderBindingSet.
// Each allowed provider is normalized via NormalizeProviderID; an unmapped/unknown
// provider, a missing/dangling command_profile_ref, empty allowed_providers, or a
// duplicate worker id all fail closed. opencode is preserved (known YAML id);
// its runtime adapter absence is handled later at the hook, not here.
func BuildProviderBindingSet(spec RulesetBridgeSpec) (ProviderBindingSet, error) {
	known := make(map[string]bool, len(spec.CommandProfiles))
	for _, cp := range spec.CommandProfiles {
		known[strings.TrimSpace(cp.ID)] = true
	}

	seen := make(map[string]bool, len(spec.WorkerProfiles))
	pbs := ProviderBindingSet{}
	for _, w := range spec.WorkerProfiles {
		wid := strings.TrimSpace(w.ID)
		if wid == "" {
			return nil, &RulesetBridgeError{Reason: "empty worker profile id"}
		}
		if seen[wid] {
			return nil, &RulesetBridgeError{Reason: "duplicate worker profile id: " + wid}
		}
		seen[wid] = true

		ref := strings.TrimSpace(w.CommandProfileRef)
		if ref == "" {
			return nil, &RulesetBridgeError{Reason: "missing cli_command_profile_ref for " + wid}
		}
		if !known[ref] {
			return nil, &RulesetBridgeError{Reason: "unknown cli_command_profile_ref " + ref + " for " + wid}
		}
		if len(w.AllowedProviders) == 0 {
			return nil, &RulesetBridgeError{Reason: "empty allowed_providers for " + wid}
		}

		for _, p := range w.AllowedProviders {
			m := NormalizeProviderID(p)
			if !m.Mapped {
				return nil, &RulesetBridgeError{Reason: "unknown provider " + p + " for " + wid}
			}
			rule := pbs[m.YAMLProviderID]
			rule.AllowedProfiles = appendUnique(rule.AllowedProfiles, ref)
			for _, e := range w.AllowedExecutables {
				rule.AllowedExecutables = appendUnique(rule.AllowedExecutables, e)
			}
			pbs[m.YAMLProviderID] = rule
		}
	}
	return pbs, nil
}

// BuildGuardRulesets builds both rule sets, failing closed on the first error.
func BuildGuardRulesets(spec RulesetBridgeSpec) (ProfileSet, ProviderBindingSet, error) {
	ps, err := BuildProfileSet(spec)
	if err != nil {
		return nil, nil, err
	}
	pbs, err := BuildProviderBindingSet(spec)
	if err != nil {
		return nil, nil, err
	}
	return ps, pbs, nil
}

// ValidateRulesetBridgeSpec reports the first validation error, if any.
func ValidateRulesetBridgeSpec(spec RulesetBridgeSpec) error {
	_, _, err := BuildGuardRulesets(spec)
	return err
}
