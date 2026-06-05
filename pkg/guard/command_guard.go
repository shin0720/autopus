// Package guard: SB8 P8a M9 command guard facade (decision layer).
//
// EvaluateCommandGuard composes M1~M8 into a single deny-first decision. It is a
// FACADE / DECISION layer only — it performs NO actual enforcement: it does not
// run commands or subprocesses, makes no network call, spawns no subagent, and
// integrates no runner hook. Wiring this decision into the execution path (e.g.
// orchestra/command.go newCommand) is P8b actual hook integration (separate
// approval). Every "allow" is an allow CANDIDATE; the real block happens only
// once the P8b hook calls this and refuses to Start() on deny.
package guard

import "strings"

// CommandGuardPhase identifies which gate produced the decision.
type CommandGuardPhase string

const (
	PhaseScriptInspector CommandGuardPhase = "script_inspector"
	PhaseNonStructured   CommandGuardPhase = "non_structured"
	PhaseDenylist        CommandGuardPhase = "denylist"
	PhaseGitGate         CommandGuardPhase = "git_gate"
	PhaseProfile         CommandGuardPhase = "profile"
	PhaseProviderBinding CommandGuardPhase = "provider_binding"
	PhaseSubagent        CommandGuardPhase = "subagent"
	PhaseEgress          CommandGuardPhase = "egress"
	PhaseAllow           CommandGuardPhase = "allow"
)

// CommandGuardRequest carries everything the facade may evaluate. Each section
// is optional: a gate is applied only when its inputs are present.
type CommandGuardRequest struct {
	Executable string
	Args       []string
	RawScript  string

	ProviderID string
	ProfileID  string

	DenyRules DenyRuleSet
	Profiles  ProfileSet
	Providers ProviderBindingSet

	Subagent     *SubagentDelegationRequest
	SubagentRule SubagentCapabilityRule

	Egress      *EgressRequest
	EgressRules EgressRuleSet
}

// CommandGuardDecision is the standardized facade decision / log record.
type CommandGuardDecision struct {
	Phase       CommandGuardPhase
	Allowed     bool
	Command     string // M1 normalized compare string
	Tool        string // blocked tool/host/executable when relevant
	MatchedRule string // denied pattern/rule when relevant
	Reason      string
}

type gateResult struct {
	applicable bool
	allowed    bool
	tool       string
	matched    string
	reason     string
}

type gatePhase struct {
	name CommandGuardPhase
	eval func() gateResult
}

func denyRulesActive(rs DenyRuleSet) bool {
	return len(rs.AllowedCommands) > 0 || len(rs.DeniedExact) > 0 || len(rs.DeniedRegex) > 0
}

// normalizeDecisionReason preserves a sub-gate reason (e.g. guard_non_escalation,
// api_use_forbidden) and supplies a default only when empty.
func normalizeDecisionReason(phase CommandGuardPhase, reason string) string {
	if r := strings.TrimSpace(reason); r != "" {
		return r
	}
	return string(phase) + " denied"
}

func mkDeny(phase CommandGuardPhase, cmd, tool, matched, reason string) CommandGuardDecision {
	return CommandGuardDecision{
		Phase:       phase,
		Allowed:     false,
		Command:     cmd,
		Tool:        tool,
		MatchedRule: matched,
		Reason:      normalizeDecisionReason(phase, reason),
	}
}

// firstDeny returns the first applicable gate that denies, in order.
func firstDeny(cmd string, phases []gatePhase) (CommandGuardDecision, bool) {
	for _, p := range phases {
		r := p.eval()
		if r.applicable && !r.allowed {
			return mkDeny(p.name, cmd, r.tool, r.matched, r.reason), true
		}
	}
	return CommandGuardDecision{}, false
}

// EvaluateCommandGuard runs M1 normalization then the applicable gates deny-first:
// M6 script inspector -> M2 denylist -> M5 git gate -> M3 profile -> M4 provider
// binding -> M7 subagent -> M8 egress. The first deny wins; otherwise an allow
// candidate is returned.
func EvaluateCommandGuard(req CommandGuardRequest) CommandGuardDecision {
	nc := NormalizeCommand(req.Executable, req.Args)
	cmd := nc.CompareString

	phases := []gatePhase{
		{PhaseScriptInspector, func() gateResult {
			if req.RawScript == "" {
				return gateResult{applicable: false, allowed: true}
			}
			d := InspectScriptString(req.RawScript)
			return gateResult{true, d.Allowed, "", d.MatchedRule, d.Reason}
		}},
		// T02 non_structured gate: M6-specific reasons (powershell/install/pipe)
		// already short-circuit above; here we deny raw shell / metachar input that
		// M6 left neutral. M1 IsStructuredCommand is the only check (no policy
		// change to M1~M8). Reason is the exact SPEC log contract "non_structured".
		{PhaseNonStructured, func() gateResult {
			if req.Executable == "" && req.RawScript == "" {
				return gateResult{applicable: false, allowed: true}
			}
			if req.Executable == "" {
				// RawScript-only and M6 neutral => raw shell input.
				return gateResult{applicable: true, allowed: false, reason: "non_structured"}
			}
			if IsStructuredCommand(req.Executable, req.Args) {
				return gateResult{applicable: false, allowed: true}
			}
			return gateResult{applicable: true, allowed: false, reason: "non_structured"}
		}},
		{PhaseDenylist, func() gateResult {
			if !denyRulesActive(req.DenyRules) {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateDenylist(cmd, req.DenyRules)
			return gateResult{true, d.Allowed, "", d.MatchedPattern, d.Reason}
		}},
		{PhaseGitGate, func() gateResult {
			if req.Executable == "" {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateGitGate(req.Executable, req.Args)
			return gateResult{true, d.Allowed, "", d.MatchedRule, d.Reason}
		}},
		{PhaseProfile, func() gateResult {
			if req.ProfileID == "" || req.Profiles == nil {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateProfile(req.ProfileID, req.Executable, req.Args, req.Profiles)
			return gateResult{true, d.Allowed, "", d.MatchedRule, d.Reason}
		}},
		{PhaseProviderBinding, func() gateResult {
			if req.ProviderID == "" || req.Providers == nil {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateProviderBinding(req.ProviderID, req.ProfileID, req.Executable, req.Args, req.Providers, req.Profiles)
			return gateResult{true, d.Allowed, d.Executable, d.MatchedRule, d.Reason}
		}},
		{PhaseSubagent, func() gateResult {
			if req.Subagent == nil {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateSubagentDelegation(*req.Subagent, req.SubagentRule)
			return gateResult{true, d.Allowed, d.Tool, "", d.Reason}
		}},
		{PhaseEgress, func() gateResult {
			if req.Egress == nil {
				return gateResult{applicable: false, allowed: true}
			}
			d := EvaluateEgress(*req.Egress, req.EgressRules)
			return gateResult{true, d.Allowed, d.Host, "", d.Reason}
		}},
	}

	if dec, denied := firstDeny(cmd, phases); denied {
		return dec
	}
	return CommandGuardDecision{
		Phase:   PhaseAllow,
		Allowed: true,
		Command: cmd,
		Reason:  "all applicable guards passed (allow candidate; actual enforcement requires P8b hook)",
	}
}
