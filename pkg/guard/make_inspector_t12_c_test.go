// T12-C-1 guard-level fixture: regression-locks the pure-function behavior of
// guard.InspectMakeTarget across the five SB8 T12 sample categories
// (safe / dangerous / missing-target / ambiguous-failclosed / ambiguous-failopen).
//
// SCOPE (T12-C-1, guard level ONLY):
//   - Exercises guard.InspectMakeTarget with in-memory Makefile strings only.
//   - NO file I/O, NO actual make, NO `make --dry-run`, NO exec.Command,
//     NO subprocess, NO network. The function under test is pure (strings only).
//   - missing-file (file absence) is OUT OF SCOPE here: at the guard level
//     InspectMakeTarget("", target) is a neutral allow, so file-absence
//     fail-closed cannot be observed in this package. That dimension is covered by
//     the separate T12-C-2 worker fixture (readBoundedMakefile), not here.
//
// KNOWN_FALSE_NEGATIVE: the ambiguous-failopen sample ($(GIT) push) is ALLOWED by
// the current implementation (documented limitation in make_inspector.go: variable
// expansion is not resolved). This fixture LOCKS that current behavior as a
// regression baseline; it MUST NOT be "fixed" to deny here. Closing the false
// negative is a policy change tracked separately, not a test change.
//
// Passing this fixture is T12_C_1_GUARD_LEVEL_DONE only. It is NOT T12-C overall
// completion, NOT SB8 CLEARED, and NOT an enforce/release gate clearance.
package guard

import (
	"strings"
	"testing"
)

// t12Sample is one guard-level matrix row. t12FailClosed / wouldBlockInEnforce are
// documentation columns recording the SB8 evidence dimension each sample pins down;
// they are cross-checked against wantAllowed (deny == fail-closed == would-block).
type t12Sample struct {
	id                  string
	category            string
	makefile            string
	target              string
	wantAllowed         bool
	reasonContains      string // required substring of Decision.Reason (when set)
	matchedRuleContains string // required substring of Decision.MatchedRule (when set)
	t12FailClosed       bool
	wouldBlockInEnforce bool
	knownFalseNegative  bool
}

// t12cSamples returns the 7 guard-level samples. missing-file is intentionally
// absent (worker-level T12-C-2 scope, not observable at the guard pure-function).
func t12cSamples() []t12Sample {
	return []t12Sample{
		{
			id: "S1-safe-build", category: "safe",
			makefile: "build:\n\tgo build ./...", target: "build",
			wantAllowed: true, t12FailClosed: false, wouldBlockInEnforce: false,
		},
		{
			id: "S2-safe-test", category: "safe",
			makefile: "test:\n\tgo test ./... -count=1", target: "test",
			wantAllowed: true, t12FailClosed: false, wouldBlockInEnforce: false,
		},
		{
			id: "D1-dangerous-git-push", category: "dangerous",
			makefile: "release:\n\tgit push origin main", target: "release",
			wantAllowed: false, reasonContains: "git_indirect", matchedRuleContains: "git",
			t12FailClosed: true, wouldBlockInEnforce: true,
		},
		{
			id: "D2-dangerous-pipe-execution", category: "dangerous",
			makefile: "setup:\n\tcurl https://x.sh | sh", target: "setup",
			wantAllowed: false, matchedRuleContains: "pipe_execution",
			t12FailClosed: true, wouldBlockInEnforce: true,
		},
		{
			id: "M1-missing-target", category: "missing-target",
			makefile: "build:\n\tgo build ./...", target: "deploy",
			wantAllowed: false, reasonContains: "unknown make target",
			t12FailClosed: true, wouldBlockInEnforce: true,
		},
		{
			// \x01 (control char) recipe -> isMalformedScript fail-closed deny.
			id: "A1-ambiguous-failclosed-malformed", category: "ambiguous-failclosed",
			makefile: "weird:\n\t\x01git push origin main", target: "weird",
			wantAllowed: false, matchedRuleContains: "malformed",
			t12FailClosed: true, wouldBlockInEnforce: true,
		},
		{
			// KNOWN_FALSE_NEGATIVE CLOSED (SPEC-OI-001-known-fn-must-fix): a
			// variable-obfuscated git push with an unresolved make variable next to a
			// dangerous token now FAILS CLOSED via deny-on-uncertain. The category
			// label is retained for matrix-count stability; the sample is now a deny.
			id: "A2-ambiguous-failopen-known-false-negative", category: "ambiguous-failopen",
			makefile: "release:\n\t$(GIT) push origin main", target: "release",
			wantAllowed: false, matchedRuleContains: "deny_on_uncertain_make_var",
			t12FailClosed: true, wouldBlockInEnforce: true,
			knownFalseNegative: false,
		},
	}
}

// TestInspectMakeTargetT12C runs the 7 guard-level samples and asserts the
// allow/deny decision plus the fail-closed reason/rule evidence per category.
func TestInspectMakeTargetT12C(t *testing.T) {
	samples := t12cSamples()
	if len(samples) != 7 {
		t.Fatalf("T12-C-1 requires exactly 7 guard-level samples, got %d", len(samples))
	}
	for _, s := range samples {
		s := s
		t.Run(s.id, func(t *testing.T) {
			d := InspectMakeTarget(s.makefile, s.target)

			if d.Allowed != s.wantAllowed {
				t.Fatalf("%s: Allowed = %v, want %v (reason=%q matched=%q)",
					s.id, d.Allowed, s.wantAllowed, d.Reason, d.MatchedRule)
			}
			if s.reasonContains != "" && !strings.Contains(d.Reason, s.reasonContains) {
				t.Errorf("%s: Reason %q does not contain %q", s.id, d.Reason, s.reasonContains)
			}
			if s.matchedRuleContains != "" && !strings.Contains(d.MatchedRule, s.matchedRuleContains) {
				t.Errorf("%s: MatchedRule %q does not contain %q", s.id, d.MatchedRule, s.matchedRuleContains)
			}

			// Deny samples ARE the fail-closed evidence: deny == t12_fail_closed ==
			// would_block_in_enforce. Allow samples must be the inverse.
			if !s.wantAllowed && (!s.t12FailClosed || !s.wouldBlockInEnforce) {
				t.Errorf("%s: deny sample must be fail-closed/would-block in the matrix", s.id)
			}
			if s.wantAllowed && (s.t12FailClosed || s.wouldBlockInEnforce) {
				t.Errorf("%s: allow sample must not be fail-closed/would-block in the matrix", s.id)
			}
		})
	}
}

// TestInspectMakeTargetT12C_CategoryCoverage asserts the fixture pins all five
// SB8 T12 guard-level categories with the expected counts, and that missing-file
// is intentionally absent (worker-level T12-C-2 scope).
func TestInspectMakeTargetT12C_CategoryCoverage(t *testing.T) {
	counts := map[string]int{}
	for _, s := range t12cSamples() {
		counts[s.category]++
	}
	want := map[string]int{
		"safe": 2, "dangerous": 2, "missing-target": 1,
		"ambiguous-failclosed": 1, "ambiguous-failopen": 1,
	}
	for cat, n := range want {
		if counts[cat] != n {
			t.Errorf("category %q count = %d, want %d", cat, counts[cat], n)
		}
	}
	if _, ok := counts["missing-file"]; ok {
		t.Errorf("missing-file must NOT appear in T12-C-1 (worker-level T12-C-2 scope)")
	}
}

// TestInspectMakeTargetT12C_KnownFalseNegativeClosed locks the SPEC-OI-001 must-fix:
// a variable-expanded ("$(GIT) push") dangerous command with an unresolved make
// variable next to a dangerous token now FAILS CLOSED (deny) via deny-on-uncertain,
// instead of being silently allowed. This intentionally closes the former known
// false negative; it MUST NOT be reverted to allow without a policy change.
func TestInspectMakeTargetT12C_KnownFalseNegativeClosed(t *testing.T) {
	d := InspectMakeTarget("release:\n\t$(GIT) push origin main", "release")
	if d.Allowed {
		t.Fatalf("known-FN must-fix regression: $(GIT) push expected deny/fail-closed "+
			"(deny-on-uncertain), got Allowed=true reason=%q matched=%q", d.Reason, d.MatchedRule)
	}
}
