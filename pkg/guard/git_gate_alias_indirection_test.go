package guard

import "testing"

func assertAliasGateDeny(t *testing.T, caseID string, args []string) {
	t.Helper()
	d := EvaluateGitGate("git", args)
	if d.Allowed {
		t.Fatalf("%s expected deny/fail-closed, got allow: %+v", caseID, d)
	}
	if d.Category != "git" {
		t.Fatalf("%s expected git category, got %+v", caseID, d)
	}
}

func assertAliasGateAllow(t *testing.T, caseID string, args []string) {
	t.Helper()
	d := EvaluateGitGate("git", args)
	if !d.Allowed {
		t.Fatalf("%s expected allow candidate, got deny: %+v", caseID, d)
	}
	if d.Category != "git" {
		t.Fatalf("%s expected git category, got %+v", caseID, d)
	}
}

// TestGitGateAliasIndirection_OI038_PushAliasDeny pins the future desired
// behavior from SPEC-OI-001-oi038-alias-indirection. It is expected-to-fail
// until command-line alias expansion / deny-on-uncertain is implemented.
func TestGitGateAliasIndirection_OI038_PushAliasDeny(t *testing.T) {
	assertAliasGateDeny(t, "OI-038", []string{"-c", "alias.x=push", "x"})
}

func TestGitGateAliasIndirection_DirectPushStillDeny(t *testing.T) {
	assertAliasGateDeny(t, "direct git push", []string{"push"})
}

func TestGitGateAliasIndirection_ConfigGetAllow(t *testing.T) {
	assertAliasGateAllow(t, "git config --get", []string{"config", "--get", "user.name"})
}

func TestGitGateAliasIndirection_RemoteVAllow(t *testing.T) {
	assertAliasGateAllow(t, "git remote -v", []string{"remote", "-v"})
}

func TestGitGateAliasIndirection_ReadOnlyAliasAllow(t *testing.T) {
	assertAliasGateAllow(t, "benign alias candidate", []string{"-c", "alias.l=status", "l"})
}

// TestGitGateAliasIndirection_DangerousAliasDeny is expected-to-fail until
// command-line alias expansion / deny-on-uncertain is implemented.
func TestGitGateAliasIndirection_DangerousAliasDeny(t *testing.T) {
	assertAliasGateDeny(t, "dangerous alias candidate", []string{"-c", "alias.ship=push", "ship"})
}

// TestGitGateAliasIndirection_BangAliasDeny is expected-to-fail until bang
// aliases are treated as fail-closed by the git gate.
func TestGitGateAliasIndirection_BangAliasDeny(t *testing.T) {
	assertAliasGateDeny(t, "bang alias candidate", []string{"-c", "alias.x=!sh", "x"})
}

// TestGitGateAliasIndirection_AmbiguousAliasDeny is expected-to-fail until
// ambiguous command-line aliases fail closed before enforce.
func TestGitGateAliasIndirection_AmbiguousAliasDeny(t *testing.T) {
	assertAliasGateDeny(t, "ambiguous alias candidate", []string{"-c", "alias.x=$(unknown)", "x"})
}
