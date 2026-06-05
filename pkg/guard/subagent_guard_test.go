package guard

import "testing"

func baseSubagentRule() SubagentCapabilityRule {
	return SubagentCapabilityRule{
		KnownSubagents:         []string{"executor", "tester", "reviewer", "W06"},
		WriteEditBashForbidden: true,
	}
}

func TestSubagent_WriteDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"write"}, RequestedTool: "Write"}, baseSubagentRule())
	if d.Allowed || d.Tool != "Write" || d.Reason != ReasonNonEscalation {
		t.Errorf("write delegation must deny as guard_non_escalation, got %+v", d)
	}
}

func TestSubagent_EditDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"edit"}, RequestedTool: "Edit"}, baseSubagentRule())
	if d.Allowed || d.Tool != "Edit" {
		t.Errorf("edit delegation must deny, got %+v", d)
	}
}

func TestSubagent_BashDeny(t *testing.T) {
	// SPEC T10 log contract: tool=Bash reason=guard_non_escalation.
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"bash"}, RequestedTool: "Bash"}, baseSubagentRule())
	if d.Allowed || d.Tool != "Bash" || d.Reason != ReasonNonEscalation {
		t.Errorf("bash delegation must deny tool=Bash reason=guard_non_escalation, got %+v", d)
	}
}

func TestSubagent_ShellDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"shell"}, RequestedTool: "Shell"}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("shell delegation must deny, got %+v", d)
	}
}

func TestSubagent_CommandExecDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"command"}}, baseSubagentRule())
	if d.Allowed || d.Reason != ReasonNonEscalation {
		t.Errorf("command execution delegation must deny, got %+v", d)
	}
}

func TestSubagent_ReadOnlyAllowCandidate(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "reviewer", RequestedCapabilities: []string{"read", "grep"}, RequestedTool: "Read"}, baseSubagentRule())
	if !d.Allowed || d.SubagentID != "reviewer" {
		t.Errorf("read-only delegation should be allow candidate, got %+v", d)
	}
}

func TestSubagent_UnknownDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "ghost", RequestedCapabilities: []string{"read"}, RequestedTool: "Read"}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("unknown subagent must fail-closed deny, got %+v", d)
	}
}

func TestSubagent_EmptyCapabilityDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: nil}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("empty capability list must fail-closed deny, got %+v", d)
	}
}

func TestSubagent_MixedReadWriteDeny(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"read", "write"}, RequestedTool: "Write"}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("mixed read+write must deny, got %+v", d)
	}
}

func TestSubagent_UnknownCapabilityFailClosed(t *testing.T) {
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "executor", RequestedCapabilities: []string{"telepathy"}}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("unknown capability must fail-closed deny, got %+v", d)
	}
}

func TestSubagent_ToolEscalationDefense(t *testing.T) {
	// Capabilities look read-only but the tool itself escalates (Bash via M1 norm).
	d := EvaluateSubagentDelegation(SubagentDelegationRequest{SubagentID: "reviewer", RequestedCapabilities: []string{"read"}, RequestedTool: "Bash.exe"}, baseSubagentRule())
	if d.Allowed {
		t.Errorf("tool-name escalation must deny, got %+v", d)
	}
}

func TestSubagent_CapabilityClassifiers(t *testing.T) {
	if !IsMutatingCapability("Bash") || !IsMutatingCapability("write") {
		t.Error("Bash/write should be mutating")
	}
	if !IsReadOnlyCapability("read") || !IsReadOnlyCapability("Grep") {
		t.Error("read/Grep should be read-only")
	}
	if ClassifyCapability("edit") != "mutating" || ClassifyCapability("status") != "readonly" || ClassifyCapability("xyz") != "unknown" {
		t.Error("classification mismatch")
	}
}
