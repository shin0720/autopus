# SPEC-OI-001 Pure API Observation — Result

**Status**: OBSERVED_EVIDENCE_CANDIDATE — NOT A SB8 CLEAR ARTIFACT  
**Date**: 2026-06-04  
**Method**: Static code trace (pure/deterministic functions; `go run`/`go test` prohibited; trace is equivalent to direct call for side-effect-free functions)  
**Related**: SPEC-OI-001-pure-api-observation-plan.md, operational-evidence.csv, samples.csv

---

## 1. Purpose

Collect `observed_decision` / `observed_guard_id_or_phase` / `observed_status` for 8 planned
samples (OI-020, OI-021, OI-033, OI-034, OI-035, OI-036, OI-037, OI-038) by tracing through
pure/static guard functions with synthetic inputs.

Quality signal and operational evidence are kept strictly separate:
- This artifact = **operational evidence candidate** (observed behavior record)
- CSV update = **not performed in this artifact** (requires separate CSV update preflight)

---

## 2. Method

All 7 candidate functions were confirmed as pure/static in the dry-run preflight:

| Function | File | No subprocess | No network | No shell | No file I/O |
|----------|------|:---:|:---:|:---:|:---:|
| `EvaluateCommandGuard` | `pkg/guard/command_guard.go:112` | ✓ | ✓ | ✓ | ✓ |
| `EvaluateGitGate` | `pkg/guard/git_gate.go:228` | ✓ | ✓ | ✓ | ✓ |
| `InspectMakeTarget` | `pkg/guard/make_inspector.go:180` | ✓ | ✓ | ✓ | ✓ |
| `InspectScriptString` | `pkg/guard/script_inspector.go:107` | ✓ | ✓ | ✓ | ✓ |
| `EvaluateDenylist` | `pkg/guard/denylist.go:61` | ✓ | ✓ | ✓ | ✓ |
| `EvaluateEgress` | `pkg/guard/egress_gate.go:109` | ✓ | ✓ | ✓ (URL parse only) | ✓ |
| `EvaluateSubagentDelegation` | `pkg/guard/subagent_guard.go:89` | ✓ | ✓ | ✓ | ✓ |

**Key normalizer trace** (critical for `$(GIT)` / `$(MAKE)` cases):
- `NormalizeExecutable("$(GIT)")` → `strings.ToLower(filepath.Base("$(GIT)"))` = `"$(git)"` (no `/`, no ext)
- `ClassifyCommandCategory("$(git)")` → `"other"` (not in `"git"/"gh"/"auto"/"doctor"` switch)
- Therefore `EvaluateGitGate("$(GIT)", ...)` returns `{Allowed:true, Category:"other", Reason:"out of git-gate scope (neutral)"}` — the dangerous command check is bypassed
- However `make_inspector.go:204` applies `hasUnresolvedMakeVar && hasDangerousContext` fail-closed path afterward

---

## 3. Safety Boundary

| Boundary | Status |
|----------|--------|
| `go run` / `go test` / `go env` / `go list` / `go mod` | NOT EXECUTED |
| subprocess invocation | NOT INVOKED |
| network / DNS / HTTP | NOT INVOKED |
| actual `git` / `make` / `sh` / `bash` | NOT INVOKED |
| hook / probe path | NOT INVOKED |
| provider CLI | NOT INVOKED |
| `git add` / `commit` / `push` / `PR` | NOT PERFORMED |
| `env=enforce` | NOT ACTIVATED |
| existing CSV / SPEC / result modification | NOT PERFORMED |

`no_subprocess_evidence: true` (precondition confirmed — static trace only)  
`no_network_evidence: true` (precondition confirmed — static trace only)

---

## 4. Observation Result Table

| sample_id | source_function | synthetic_input_summary | observed_decision | observed_guard_id_or_phase | observed_status | matched_rule_or_reason |
|-----------|-----------------|------------------------|-------------------|---------------------------|-----------------|------------------------|
| OI-020 | `InspectMakeTarget` | `makefileText="release:\n\t$(GIT) push origin main"` target=`"release"` | **deny** | `M6_T12/deny_on_uncertain_make_var` | `delta_fn_resolved_quality_improvement` | `deny_on_uncertain_make_var` — `hasUnresolvedMakeVar("$(GIT) push origin main")=true` AND `hasDangerousContext` matched `"push"` |
| OI-021 | `InspectMakeTarget` | `makefileText="ship:\n\t$(MAKE) -C sub push"` target=`"ship"` | **deny** | `M6_T12/deny_on_uncertain_make_var` | `delta_fn_resolved_quality_improvement` | `deny_on_uncertain_make_var` — `hasUnresolvedMakeVar("$(MAKE) -C sub push")=true` AND `hasDangerousContext` matched `"push"` |
| OI-033 | `EvaluateGitGate` | `executable="git"` args=`["stash","list"]` | **allow** | `P8a_git_gate_allow_candidate` | `fp_concern_not_reproduced_allow_confirmed` | No match in DeniedRegex: `^git\s+(add\|commit\|push\|merge\|rebase\|reset\|clean)\b` — `stash` not listed |
| OI-034 | `InspectMakeTarget` | `makefileText="help:\n\t@echo \"Available targets\""` target=`"help"` | **allow** | `P8a_make_inspector_allow_candidate` | `fp_concern_not_reproduced_allow_confirmed` | `@` stripped → `echo "Available targets"` → `EvaluateGitGate("echo", ...)` → category=other → allow; no unresolved var |
| OI-035 | `EvaluateGitGate` | `executable="git"` args=`["config","--get","user.name"]` | **allow** | `P8a_git_gate_allow_candidate` | `fp_concern_not_reproduced_allow_confirmed` | No regex match; no alias `-c` flag in args; `config` not a mutation verb |
| OI-036 | `EvaluateGitGate` | `executable="git"` args=`["remote","-v"]` | **allow** | `P8a_git_gate_allow_candidate` | `fp_concern_not_reproduced_allow_confirmed` | `^git\s+remote\s+set-url\b` does NOT match `"git remote -v"`; no other match |
| OI-037 | `InspectMakeTarget` | `makefileText="release:\n\t$(GIT) push --force origin main"` target=`"release"` | **deny** | `M6_T12/deny_on_uncertain_make_var` | `delta_fn_resolved_quality_improvement` | `deny_on_uncertain_make_var` — `hasUnresolvedMakeVar=true` AND `hasDangerousContext` matched `"push"` + `"force"` |
| OI-038 | `EvaluateGitGate` | `executable="git"` args=`["-c","alias.x=push","x"]` | **deny** | `M5_git_alias_dangerous` | `delta_fn_resolved_quality_improvement` | `evaluateGitCommandLineAlias`: alias value `"push"` → `isMutationGitVerb("push")=true` → `classifyGitAliasValue` = dangerous → `(true,"git_alias_dangerous","dangerous git command-line alias denied (fail-closed)")` |

---

## 5. Expected-vs-Observed Delta Table

| sample_id | expected_from_samples_csv | observed_decision | delta_from_samples_csv | interpretation |
|-----------|--------------------------|-------------------|------------------------|----------------|
| OI-020 | `allow` (expected_false_negative=true, KNOWN_FALSE_NEGATIVE) | **deny** | DELTA: csv=allow, observed=deny | `deny_on_uncertain_make_var` logic (make_inspector.go:204) resolves the known FN. Implementation quality improvement. |
| OI-021 | `allow` (expected_false_negative=true, KNOWN_FALSE_NEGATIVE) | **deny** | DELTA: csv=allow, observed=deny | Same as OI-020. `$(MAKE)` + "push" context → fail-closed. Implementation quality improvement. |
| OI-033 | `deny` (expected_false_positive=true, FP candidate) | **allow** | DELTA: csv=deny (FP), observed=allow | FP concern does not exist in current code. `stash` is not in the mutation regex. Current implementation is correct. |
| OI-034 | `deny` (expected_false_positive=true, FP candidate) | **allow** | DELTA: csv=deny (FP), observed=allow | FP concern does not exist. `echo` recipe → category "other" → allow. Current implementation is correct. |
| OI-035 | `deny` (expected_false_positive=true, FP candidate) | **allow** | DELTA: csv=deny (FP), observed=allow | FP concern does not exist. `config --get` not in mutation denylist. Current implementation is correct. |
| OI-036 | `deny` (expected_false_positive=true, FP candidate) | **allow** | DELTA: csv=deny (FP), observed=allow | FP concern does not exist. `remote -v` ≠ `remote set-url`. Current implementation is correct. |
| OI-037 | `allow` (expected_false_negative=true, FN candidate) | **deny** | DELTA: csv=allow (FN), observed=deny | `deny_on_uncertain_make_var` handles `$(GIT) push --force`. Implementation quality improvement. |
| OI-038 | `allow` (expected_false_negative=true, FN candidate) | **deny** | DELTA: csv=allow (FN), observed=deny | Alias indirection `alias.x=push` → mutation verb detection → fail-closed. Implementation quality improvement. |

---

## 6. Trace Detail: Key Decision Points

### OI-020 / OI-021 / OI-037 — deny_on_uncertain_make_var path

```
make_inspector.go:180 InspectMakeTarget(makefileText, target)
  ↓ ExtractMakeTargetRecipe → recipe = ["$(GIT) push origin main"]  [or similar]
  ↓ parseSimpleMakeVars(makefileText) → vars = {}  (GIT/MAKE not defined in makefile text)
  ↓ for raw in recipe:
      expanded = expandSimpleMakeVarsOnce({}, "$(GIT) push origin main")
             = "$(GIT) push origin main"  ← no substitution (vars empty)
      inspectMakeRecipeLine("$(GIT) push origin main")
        ↓ InspectScriptString → Allowed=true (no pipe, no bypass, no install.ps1)
        ↓ fields[0] = "$(GIT)"
          EvaluateGitGate("$(GIT)", ["push","origin","main"])
            NormalizeExecutable("$(GIT)") = "$(git)"
            ClassifyCommandCategory("$(git)") = "other"  ← not "git"
            → {Allowed:true, Category:"other"}  ← BYPASS (not dangerous in M5 scope)
        → inspectMakeRecipeLine returns (false, "", "")
      hasUnresolvedMakeVar("$(GIT) push origin main") = true   ← "$(GIT)" still present
      hasDangerousContext("$(GIT) push origin main") = true    ← "push" matched
      → DENY: deny_on_uncertain_make_var
```

### OI-033 / OI-035 / OI-036 — git gate allow path

```
git_gate.go:228 EvaluateGitGate("git", args)
  ↓ NormalizeExecutable("git") = "git"
  ↓ ClassifyCommandCategory("git") = "git"
  ↓ evaluateGitCommandLineAlias(args):
      OI-033: args=["stash","list"]   → no "-c" flag → aliases={} → return (false,"","")
      OI-035: args=["config","--get","user.name"] → no "-c" → aliases={} → return (false,"","")
      OI-036: args=["remote","-v"]    → no "-c" → aliases={} → return (false,"","")
  ↓ dangerousIn("git", compareString):
      DeniedRegex: "^git\s+(add|commit|push|merge|rebase|reset|clean)\b"
      OI-033: "git stash list"  → "stash" NOT in group → no match
      OI-035: "git config --get user.name" → "config" NOT in group → no match
      OI-036: "git remote -v"   → "^git\s+remote\s+set-url\b" does NOT match "-v"
      → not dangerous
  → {Allowed:true, Category:"git", Reason:"git command, no dangerous pattern (allow candidate)"}
```

### OI-038 — alias indirection deny path

```
git_gate.go evaluateGitCommandLineAlias(["-c", "alias.x=push", "x"])
  ↓ parseGitCommandLineAliases:
      arg="-c" → next="alias.x=push" → prefix "alias." matched
      keyValue="x=push" → name="x", value="push"
      aliases = {"x": "push"}
      next arg = "x" → not "-c" → remaining=["x"]
  ↓ for value "push" in aliases:
      classifyGitAliasValue("push"):
        aliasValueHasShellRisk("push") = false
        fields = ["push"]
        isMutationGitVerb("push") = true  ← "push" in mutation verb set
        → (dangerous=true, uncertain=false)
      → return (true, "git_alias_dangerous", "dangerous git command-line alias denied (fail-closed)")
  → {Allowed:false, MatchedRule:"git_alias_dangerous", ...}
```

---

## 7. CSV Mutation Boundary

**HARD RULE**: operational-evidence.csv and samples.csv were NOT modified in this observation.

Delta summary for CSV update preflight (NOT applied yet):

| sample_id | field: observed_decision | field: observed_guard_id_or_phase | field: observed_status |
|-----------|--------------------------|-----------------------------------|------------------------|
| OI-020 | deny | M6_T12/deny_on_uncertain_make_var | delta_fn_resolved_quality_improvement |
| OI-021 | deny | M6_T12/deny_on_uncertain_make_var | delta_fn_resolved_quality_improvement |
| OI-033 | allow | P8a_git_gate_allow_candidate | fp_concern_not_reproduced_allow_confirmed |
| OI-034 | allow | P8a_make_inspector_allow_candidate | fp_concern_not_reproduced_allow_confirmed |
| OI-035 | allow | P8a_git_gate_allow_candidate | fp_concern_not_reproduced_allow_confirmed |
| OI-036 | allow | P8a_git_gate_allow_candidate | fp_concern_not_reproduced_allow_confirmed |
| OI-037 | deny | M6_T12/deny_on_uncertain_make_var | delta_fn_resolved_quality_improvement |
| OI-038 | deny | M5/git_alias_dangerous | delta_fn_resolved_quality_improvement |

---

## 8. FP_REVIEW Readiness Impact

| sample_id | FP_REVIEW status | evidence basis |
|-----------|-----------------|----------------|
| OI-033 | FP concern does NOT reproduce in current code — `allow` observed | `stash` not in mutation regex; no over-block |
| OI-034 | FP concern does NOT reproduce — `allow` observed | echo recipe → category=other → allow |
| OI-035 | FP concern does NOT reproduce — `allow` observed | `config --get` not blocked |
| OI-036 | FP concern does NOT reproduce — `allow` observed | `remote -v` ≠ `remote set-url` |

**Action required**: FP_REVIEW for OI-033/034/035/036 has observational basis to close as
`fp_concern_not_reproduced`. However, formal FP_REVIEW resolved processing is NOT performed
here — requires separate CSV update preflight approval.

**OI-020/OI-021/OI-037/OI-038**: Were `must_fix_before_enforce` / FN candidates.  
Current code shows `deny` for all 4. This is an implementation quality improvement over the
samples.csv record. `env=enforce` eligibility review requires re-evaluation of `must_fix`
status — NOT performed here.

---

## 9. Remaining Blockers

| Blocker | Status |
|---------|--------|
| operational-evidence.csv not updated | PENDING — CSV update preflight required |
| samples.csv `expected_guard_decision` delta not reconciled | PENDING — CSV update preflight required |
| FP_REVIEW formally resolved | BLOCKED — not performed here |
| SB8 CLEARED | BLOCKED — observation only; not a clear artifact |
| env=enforce | BLOCKED — must_fix re-evaluation not complete |
| git add/commit/push/PR | BLOCKED — NO_GIT_ACTION maintained |

---

## 10. Next Recommended Step

**CSV update preflight** — define exact field-by-field changes to operational-evidence.csv
and samples.csv based on observed evidence above, get approval, then apply.

Sequence:
1. CSV update preflight (define changes, no apply)
2. CSV update apply (apply approved changes)
3. FP_REVIEW readiness re-evaluation (OI-033/034/035/036)
4. must_fix status re-evaluation (OI-020/OI-021/OI-037/OI-038)
5. env=enforce eligibility re-evaluation (after must_fix re-evaluation)

---

## 11. Observation Checklist

- [x] 8/8 samples observed
- [x] `no_subprocess_evidence = true`
- [x] `no_network_evidence = true`
- [x] `go run` / `go test` / `go env` not invoked
- [x] actual `git` / `make` / `sh` not invoked
- [x] `env=enforce` not activated
- [x] `git add` / `commit` / `push` / `PR` not performed
- [x] existing CSV not modified
- [x] FP_REVIEW not resolved
- [x] SB8 CLEARED not declared
