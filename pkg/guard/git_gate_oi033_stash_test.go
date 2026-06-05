// OI-033 coverage gap closure: git stash list must be an allow candidate.
//
// SCOPE: pure EvaluateGitGate with synthetic args only.
//
//	NO actual git, NO shell, NO exec.Command, NO subprocess, NO network,
//	NO env=enforce, NO CSV access, NO file I/O.
//
// This file closes the OI-033 direct-assert gap identified in the
// SB8 pure-API observation execution path preflight. go test is NOT run
// in the turn that creates this file.
//
// FP_REVIEW is NOT resolved, SB8 is NOT cleared, and this is NOT a
// release/enforce gate clearance.
package guard

import "testing"

// TestGitGate_OI033_StashListAllow asserts that "git stash list" is an allow
// candidate: stash is a read-only listing subcommand and must not be matched
// by the mutation denylist regex (add|commit|push|merge|rebase|reset|clean).
func TestGitGate_OI033_StashListAllow(t *testing.T) {
	d := EvaluateGitGate("git", []string{"stash", "list"})
	if !d.Allowed {
		t.Fatalf("OI-033: git stash list expected allow candidate, got deny: "+
			"category=%q matched=%q reason=%q",
			d.Category, d.MatchedRule, d.Reason)
	}
	if d.Category != "git" {
		t.Fatalf("OI-033: expected category git, got %q", d.Category)
	}
}

