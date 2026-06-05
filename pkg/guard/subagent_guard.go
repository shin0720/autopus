// Package guard: SB8 P6 M7 subagent capability guard.
//
// Decision layer ONLY. Given a subagent delegation request, it decides whether a
// parent node may delegate a tool/capability to a subagent under the
// non-escalation principle (a subagent must not acquire a capability the parent
// guard forbids). It NEVER spawns or runs a subagent, NEVER executes a tool,
// does NOT rewrite prompts, does NOT integrate a runner hook (M9), and does NOT
// implement network egress (M8). Capability classification is bespoke here;
// M1 NormalizeExecutable is reused only to canonicalize the tool name. The M2
// command denylist is intentionally NOT applied to capability enums.
package guard

import "strings"

// ReasonNonEscalation is the SPEC T10 deny reason for blocked delegation
// (log contract: "DENY delegation tool=<Tool> reason=guard_non_escalation").
const ReasonNonEscalation = "guard_non_escalation"

// SubagentDelegationRequest is a parent node's request to delegate to a subagent.
type SubagentDelegationRequest struct {
	SubagentID            string
	RequestedCapabilities []string
	RequestedTool         string
}

// SubagentCapabilityRule models the YAML subagent_capability_guard data.
type SubagentCapabilityRule struct {
	KnownSubagents         []string // recognized subagent ids
	WriteEditBashForbidden bool     // YAML write_edit_bash_delegation: forbidden
}

// SubagentGuardDecision is the result of evaluating a delegation request.
type SubagentGuardDecision struct {
	Allowed    bool
	SubagentID string
	Tool       string // blocked capability/tool name when denied (e.g. "Bash")
	Reason     string
}

var mutatingCaps = map[string]bool{
	"write": true, "edit": true, "bash": true, "shell": true, "command": true,
}

var readOnlyCaps = map[string]bool{
	"read": true, "check": true, "grep": true, "glob": true, "inspect": true, "status": true,
}

func normalizeCapability(c string) string {
	return strings.ToLower(strings.TrimSpace(c))
}

// IsMutatingCapability reports whether a capability mutates state / executes.
func IsMutatingCapability(c string) bool { return mutatingCaps[normalizeCapability(c)] }

// IsReadOnlyCapability reports whether a capability is read-only / check-only.
func IsReadOnlyCapability(c string) bool { return readOnlyCaps[normalizeCapability(c)] }

// ClassifyCapability returns mutating | readonly | unknown.
func ClassifyCapability(c string) string {
	n := normalizeCapability(c)
	switch {
	case mutatingCaps[n]:
		return "mutating"
	case readOnlyCaps[n]:
		return "readonly"
	default:
		return "unknown"
	}
}

func knownSubagent(rule SubagentCapabilityRule, id string) bool {
	for _, s := range rule.KnownSubagents {
		if s == id {
			return true
		}
	}
	return false
}

func deny(req SubagentDelegationRequest, tool string) SubagentGuardDecision {
	return SubagentGuardDecision{Allowed: false, SubagentID: req.SubagentID, Tool: tool, Reason: ReasonNonEscalation}
}

// EvaluateSubagentDelegation decides a delegation request under non-escalation.
// Unknown subagent, empty capabilities, an unknown capability, and mixed
// read+write capabilities all fail closed. A mutating delegation is denied when
// the guard forbids write/edit/bash. An all-read-only request is an allow
// candidate (final allow is the profile/provider/git gate's call, not here).
func EvaluateSubagentDelegation(req SubagentDelegationRequest, rule SubagentCapabilityRule) SubagentGuardDecision {
	if !knownSubagent(rule, req.SubagentID) {
		return deny(req, req.RequestedTool)
	}
	if len(req.RequestedCapabilities) == 0 {
		return deny(req, req.RequestedTool)
	}

	hasMutating, hasReadOnly := false, false
	firstMutating := ""
	for _, c := range req.RequestedCapabilities {
		switch ClassifyCapability(c) {
		case "mutating":
			hasMutating = true
			if firstMutating == "" {
				firstMutating = c
			}
		case "readonly":
			hasReadOnly = true
		default:
			return deny(req, c) // unknown capability fails closed
		}
	}

	// Defense against tool/capability mismatch: the requested tool name itself
	// may escalate even if the capability list looks benign.
	if toolIsMutating(req.RequestedTool) {
		hasMutating = true
		if firstMutating == "" {
			firstMutating = req.RequestedTool
		}
	}

	if hasMutating && hasReadOnly {
		return deny(req, blockedTool(req, firstMutating))
	}
	if hasMutating && rule.WriteEditBashForbidden {
		return deny(req, blockedTool(req, firstMutating))
	}

	return SubagentGuardDecision{
		Allowed:    true,
		SubagentID: req.SubagentID,
		Tool:       req.RequestedTool,
		Reason:     "read-only delegation (allow candidate)",
	}
}

// toolIsMutating reports whether the requested tool name is itself a mutating
// tool, after M1 NormalizeExecutable canonicalization ("Bash.exe"/"BASH" -> "bash").
func toolIsMutating(tool string) bool {
	if tool == "" {
		return false
	}
	return mutatingCaps[NormalizeExecutable(tool)]
}

// blockedTool prefers the explicit RequestedTool, falling back to the offending
// capability name.
func blockedTool(req SubagentDelegationRequest, fallbackCap string) string {
	if req.RequestedTool != "" {
		return req.RequestedTool
	}
	return fallbackCap
}
