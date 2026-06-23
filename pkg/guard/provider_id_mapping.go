// Package guard: SB8 provider-id mapping (pure decision/helper).
//
// Normalizes a runtime adapter provider id (claude/codex/gemini) to the YAML
// pipeline provider id (claude-code/codex/gemini-cli/opencode). PURE helper: it
// performs no provider execution, creates no adapter, injects no
// ProfileSet/ProviderBindingSet, and does NOT enable M3/M4 enforcement. The
// facade only runs M4 when a ProviderBindingSet is present (a later step).
package guard

import "strings"

// adapterToYAML maps runtime adapter ids to YAML pipeline provider ids.
var adapterToYAML = map[string]string{
	"claude": "claude-code",
	"codex":  "codex",
	"gemini": "gemini-cli",
}

// yamlProviderIDs is the set of known YAML pipeline provider ids.
var yamlProviderIDs = map[string]bool{
	"claude-code": true,
	"codex":       true,
	"gemini-cli":  true,
	"opencode":    true,
}

// yamlProvidersWithoutAdapter lists YAML provider ids that have NO runtime adapter.
var yamlProvidersWithoutAdapter = map[string]bool{
	"opencode": true,
}

// ProviderIDMappingDecision is the result of normalizing a provider id.
type ProviderIDMappingDecision struct {
	Input          string
	YAMLProviderID string
	Mapped         bool // input resolved to a known YAML provider id
	AdapterPresent bool // a runtime adapter exists for this provider (opencode=false)
	Reason         string
}

// IsKnownAdapterProviderID reports whether id is a known runtime adapter id.
func IsKnownAdapterProviderID(id string) bool {
	_, ok := adapterToYAML[strings.ToLower(strings.TrimSpace(id))]
	return ok
}

// IsKnownYAMLProviderID reports whether id is a known YAML pipeline provider id.
func IsKnownYAMLProviderID(id string) bool {
	return yamlProviderIDs[strings.ToLower(strings.TrimSpace(id))]
}

// MapAdapterProviderID maps an adapter id to its YAML id. ok=false if unknown.
func MapAdapterProviderID(adapterID string) (string, bool) {
	y, ok := adapterToYAML[strings.ToLower(strings.TrimSpace(adapterID))]
	return y, ok
}

// NormalizeProviderID resolves an adapter id OR an already-normalized YAML id to
// the YAML provider id. Empty input is neutral (Mapped=false); unknown input
// fails closed (Mapped=false). It NEVER enables enforcement.
func NormalizeProviderID(input string) ProviderIDMappingDecision {
	n := strings.ToLower(strings.TrimSpace(input))
	if n == "" {
		return ProviderIDMappingDecision{Input: input, Mapped: false, Reason: "empty provider (neutral; M4 skipped)"}
	}
	if y, ok := adapterToYAML[n]; ok {
		return ProviderIDMappingDecision{
			Input: input, YAMLProviderID: y, Mapped: true, AdapterPresent: true,
			Reason: "adapter id mapped to YAML provider id",
		}
	}
	if yamlProviderIDs[n] {
		present := !yamlProvidersWithoutAdapter[n]
		reason := "already a YAML provider id"
		if !present {
			reason = "known YAML provider id but no runtime adapter"
		}
		return ProviderIDMappingDecision{
			Input: input, YAMLProviderID: n, Mapped: true, AdapterPresent: present, Reason: reason,
		}
	}
	return ProviderIDMappingDecision{Input: input, Mapped: false, Reason: "unknown provider (fail-closed/unmapped)"}
}
