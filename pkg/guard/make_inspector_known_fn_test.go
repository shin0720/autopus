// Known-false-negative TESTS-FIRST regression for SPEC-OI-001-known-fn-must-fix.
//
// These tests assert the FUTURE desired behavior of the make recipe inspector after
// the must-fix (limited variable expansion + deny-on-uncertain fallback). Until the
// implementation turn lands, the OI-020 / OI-021 / OI-037 cases are EXPECTED-TO-FAIL
// (red phase): the current guard.InspectMakeTarget does NOT resolve make variable
// indirection and therefore still ALLOWS variable-expanded pushes (documented
// limitation in make_inspector.go). The OI-034 case is a false-positive control that
// MUST keep allowing a read-only help/echo target.
//
// SCOPE: pure guard.InspectMakeTarget with in-memory Makefile strings ONLY.
//
//	NO actual make, NO shell, NO git, NO exec.Command, NO subprocess, NO network,
//	NO env=enforce, NO dataset CSV access, NO file I/O.
//
// This file does NOT change source behavior. It is NOT acceptance evidence, FP_REVIEW
// is NOT resolved, SB8 is NOT cleared, and this is NOT a release/enforce gate. go test
// is NOT run in the turn that creates this file.
//
// NOTE: the separate make_inspector_t12_c_test.go intentionally LOCKS the current
// allow behavior of $(GIT) push as a regression baseline. That is the present-state
// baseline; the cases here are the opposite target (future deny) and must not be
// reconciled by weakening either side outside the implementation turn.
package guard

import "testing"

// assertMakeDeny fails when the inspector ALLOWS a recipe that must be denied or
// fail-closed. A pass (allow) is treated as a test failure, per the must-fix SPEC.
func assertMakeDeny(t *testing.T, sampleID, makefile, target string) {
	t.Helper()
	d := InspectMakeTarget(makefile, target)
	if d.Allowed {
		t.Fatalf("%s: expected deny/fail-closed for make target %q but got ALLOW "+
			"(expected-to-fail until known-FN must-fix is implemented); reason=%q",
			sampleID, target, d.Reason)
	}
}

// assertMakeAllow fails when the inspector DENIES a clearly safe read-only recipe.
// This is the false-positive control: deny-on-uncertain must not overblock it.
func assertMakeAllow(t *testing.T, sampleID, makefile, target string) {
	t.Helper()
	d := InspectMakeTarget(makefile, target)
	if !d.Allowed {
		t.Fatalf("%s: expected allow for read-only make target %q but got DENY "+
			"(false-positive regression); reason=%q matched=%q",
			sampleID, target, d.Reason, d.MatchedRule)
	}
}

// OI-020 (must-deny): variable-expanded git push must be denied/fail-closed after the
// must-fix. Currently ALLOWED (KNOWN_FALSE_NEGATIVE) -> expected-to-fail until then.
func TestInspectMakeTarget_KnownFN_VarGitPush_Deny(t *testing.T) {
	mk := "GIT=git\nrelease:\n\t$(GIT) push\n"
	assertMakeDeny(t, "OI-020", mk, "release")
}

// OI-021 (must-deny): nested make variable indirection push must be denied/fail-closed.
// No actual make is executed; the recipe text is inspected statically only.
func TestInspectMakeTarget_KnownFN_NestedMakeVarPush_Deny(t *testing.T) {
	mk := "TOOL=$(MAKE)\nship:\n\t$(TOOL) -C sub push\n"
	assertMakeDeny(t, "OI-021", mk, "ship")
}

// OI-034 (false-positive control): a read-only help/echo target with no dangerous
// token (no push/force/curl/wget) MUST stay allowed and not be over-blocked.
func TestInspectMakeTarget_FPControl_HelpEcho_Allow(t *testing.T) {
	mk := "help:\n\t@echo available targets\n"
	assertMakeAllow(t, "OI-034", mk, "help")
}

// OI-037 (must-deny / follow-up): variable-expanded force push must be denied/
// fail-closed. Authored as an ACTIVE deny assertion. If the inspector ultimately
// cannot express this within scope, the implementation turn may downgrade this to a
// t.Skip with an explicit follow-up note -- it must NOT be silently weakened here.
func TestInspectMakeTarget_KnownFN_VarForcePush_DenyOrFollowup(t *testing.T) {
	mk := "GIT=git\nrelease:\n\t$(GIT) push --force\n"
	assertMakeDeny(t, "OI-037", mk, "release")
}

// Out-of-scope tracking (NOT active tests in this make-inspector file):
//   - OI-033 (git stash list), OI-035 (git config --get user.name),
//     OI-036 (git remote -v): unaffected-allow. They belong to the git gate
//     (EvaluateGitGate / phase M5), not the make inspector, and must keep allowing
//     read-only commands. They are deliberately not asserted here.
//   - OI-038 (git -c alias.x=push x): git alias indirection is OUT OF SCOPE for the
//     make-variable must-fix and is tracked under a separate alias-indirection
//     follow-up SPEC.
