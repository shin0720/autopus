# SPEC-OI-001 SB8 Dry-Run Telemetry — CLEARED Declaration

**Date**: 2026-06-05  
**Status**: SB8 CLEARED — documentation artifact basis  
**env=enforce**: NO — separate preflight required  
**release ready**: NO  
**git add/commit/push/PR**: NO — NO_GIT_ACTION maintained  
**UI01~UI11**: NOT IN SCOPE — Studio UI separate track, not a SB8 guard blocker

---

## 1. Purpose

Formally declare SB8 dry-run telemetry closure for SPEC-OI-001 operator intent dataset.
This declaration is based on:

1. Actual runtime test evidence (`go test -v ./pkg/guard/...` PASS, 0 failures)
2. FP review completion: 4 false-positive concerns confirmed NOT REPRODUCED
3. FN re-evaluation completion: 4 known false negatives confirmed CLOSED
4. Both CSV (operational-evidence.csv + samples.csv) patched and integrity-audited
5. FP/FN resolution artifact written and verified

This is NOT an env=enforce activation, NOT a release gate, and NOT a git action approval.

---

## 2. Source Artifacts

| Artifact | Status |
|----------|:------:|
| `SPEC-OI-001-known-fn-must-fix.md` | ✓ present |
| `SPEC-OI-001-known-fn-must-fix.result.md` | ✓ present |
| `SPEC-OI-001-policy-4-decision.result.md` | ✓ present |
| `SPEC-OI-001-oi038-alias-indirection.md` | ✓ present |
| `SPEC-OI-001-oi038-alias-indirection.result.md` | ✓ present |
| `SPEC-OI-001-pure-api-observation-plan.md` | ✓ present |
| `SPEC-OI-001-pure-api-observation-actual.result.md` | ✓ present |
| `SPEC-OI-001-fp-fn-resolution.result.md` | ✓ present |
| `SPEC-OI-001-operator-intent-dataset.operational-evidence.csv` | ✓ present, patched |
| `SPEC-OI-001-operator-intent-dataset.samples.csv` | ✓ present, patched |
| `SPEC-OI-001-pure-api-observation.result.md` | ✓ present (STATIC_TRACE_ONLY — reference only) |

---

## 3. SB8 Cleared Basis

| Condition | Status |
|-----------|:------:|
| FP 4건 FP_NOT_REPRODUCED | ✓ |
| FN 4건 FN_CLOSED | ✓ |
| operational-evidence.csv: no not_collected / planned_only | ✓ |
| operational-evidence.csv: 8건 actual_observed_* | ✓ |
| samples.csv: FP/FN policy fields aligned | ✓ |
| samples.csv: forbidden fields unchanged | ✓ |
| operational-evidence.csv integrity audit | PASS ✓ |
| samples.csv integrity audit | PASS ✓ |
| go test -v ./pkg/guard/... | PASS (0 failures) ✓ |
| FP/FN resolution artifact | ✓ present |
| SB8 CLEARED preflight | PASS ✓ |

---

## 4. Operational Evidence Summary

`go test -v ./pkg/guard/...` executed once. All tests PASS. 0 failures.  
`pkg/guard`: 1.237s / `pkg/guard/telemetry`: 1.261s

| sample_id | observed_decision | test_name | evidence_type |
|-----------|:-----------------:|-----------|:-------------:|
| OI-020 | deny | TestInspectMakeTargetT12C_KnownFalseNegativeClosed | exact |
| OI-021 | deny | TestInspectMakeTarget_KnownFN_NestedMakeVarPush_Deny | near-match |
| OI-033 | allow | TestGitGate_OI033_StashListAllow | exact |
| OI-034 | allow | TestInspectMakeTarget_FPControl_HelpEcho_Allow | near-match |
| OI-035 | allow | TestGitGateAliasIndirection_ConfigGetAllow | exact |
| OI-036 | allow | TestGitGate_GitRemoteVAllow | exact |
| OI-037 | deny | TestInspectMakeTarget_KnownFN_VarForcePush_DenyOrFollowup | near-match |
| OI-038 | deny | TestGitGateAliasIndirection_OI038_PushAliasDeny | exact |

No `not_collected` values remain. No `planned_only` values remain.

---

## 5. Samples Policy Summary

**FP cases — expected_guard_decision updated to allow:**

| sample_id | expected_guard_decision | expected_false_positive | expected_would_block_in_enforce |
|-----------|:-:|:-:|:-:|
| OI-033 | allow | false | false |
| OI-034 | allow | false | false |
| OI-035 | allow | false | false |
| OI-036 | allow | false | false |

**FN cases — expected_guard_decision updated to deny:**

| sample_id | expected_guard_decision | expected_false_negative | expected_would_block_in_enforce | expected_t12_fail_closed |
|-----------|:-:|:-:|:-:|:-:|
| OI-020 | deny | false | true | true |
| OI-021 | deny | false | true | true |
| OI-037 | deny | false | true | true |
| OI-038 | deny | false | true | false |

Forbidden fields (category, rationale, expected_operator_intent, expected_m3_m4_inert,
actual_execution_allowed, acceptance_excluded, sample_id, command fields): **all unchanged**.

---

## 6. FP Resolution Summary

| sample_id | before | observed | resolution | caveat |
|-----------|:------:|:--------:|:----------:|--------|
| OI-033 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | `git stash list` — stash not in mutation regex |
| OI-034 | deny (FP) | allow (near-match) | **FP_NOT_REPRODUCED** | echo quote diff cosmetic; guard logic unaffected |
| OI-035 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | `git config --get` not blocked |
| OI-036 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | `git remote -v` ≠ `git remote set-url` |

---

## 7. FN Resolution Summary

| sample_id | before | observed | resolution | deny path |
|-----------|:------:|:--------:|:----------:|-----------|
| OI-020 | allow (KNOWN_FN) | deny (exact) | **FN_CLOSED** | deny_on_uncertain_make_var |
| OI-021 | allow (KNOWN_FN) | deny (near-match) | **FN_CLOSED** | deny_on_uncertain_make_var (same path) |
| OI-037 | allow (FN candidate) | deny (near-match) | **FN_CLOSED** | M6_T12 (plan) + M5 (test) — both deny |
| OI-038 | allow (FN candidate) | deny (exact) | **FN_CLOSED** | M5/git_alias_dangerous |

---

## 8. Near-match Caveat Summary

All three near-match cases confirmed **NOT A BLOCKER**.

| sample_id | input difference | code path | result | blocker |
|-----------|-----------------|:---------:|:------:|:-------:|
| OI-021 | TOOL=$(MAKE) prefix added | same — deny_on_uncertain_make_var | deny = deny | NO |
| OI-034 | echo arg quotes absent | same — echo→category=other→allow | allow = allow | NO |
| OI-037 | GIT=git defined in test | different — M5 (test) vs M6_T12 (plan) | deny = deny | NO |

---

## 9. Special Case Confirmation

| Case | Value | Confirmed |
|------|-------|:---------:|
| OI-037 `expected_guard_id_or_phase` | M6_T12 (plan input basis) | ✓ |
| OI-037 reviewer_notes `plan_path=M6_T12_deny_on_uncertain` | present | ✓ |
| OI-037 reviewer_notes `test_path=M5_regex_deny` | present | ✓ |
| OI-037 reviewer_notes `deny_confirmed_both_paths` | present | ✓ |
| OI-038 `expected_t12_fail_closed` | false (kept) | ✓ |
| OI-038 decision path | EvaluateGitGate → evaluateGitCommandLineAlias → M5/git_alias_dangerous | ✓ |
| OI-038 is NOT a T12 make inspector case | confirmed (git gate M5) | ✓ |

---

## 10. Safety Boundary

No prohibited action was performed during this SB8 session.

| Boundary | Count |
|----------|:-----:|
| `go test` | 1 (pkg/guard only, permitted) |
| `go run` / `go env` / `go list` / `go mod` | 0 |
| actual git / make / shell execution | 0 |
| provider CLI execution | 0 |
| network / API call | 0 |
| subprocess beyond go test | 0 |
| `env=enforce` activation | 0 |
| `git add` / `commit` / `push` / `PR` | 0 |
| existing CSV / source modification (unauthorized) | 0 |

All CSV and source modifications were explicitly authorized per-turn with defined scope.

---

## 11. Remaining Blockers After SB8

| Blocker | Required Next Step |
|---------|-------------------|
| env=enforce / N-β gate | Separate env=enforce preflight required |
| git add/commit | Still prohibited; requires explicit authorization |
| git push / PR | Still prohibited; requires explicit authorization |
| UI01~UI11 (Studio UI) | Separate Studio UI track — not a SB8 guard blocker |
| final git/artifact audit | Recommended before git commit |

---

## 12. Next Recommended Step

**env=enforce preflight** — read-only verification of all conditions required for
activating `env=enforce` on the guard system:

1. Confirm all must_fix items resolved (OI-020/021/037 FN_CLOSED ✓)
2. Confirm no remaining `expected_false_negative=true` in samples.csv
3. Confirm guard hook integration (P8b) status
4. Confirm no open FP concerns that could cause operational disruption
5. Confirm N-β gate conditions if applicable

env=enforce MUST NOT be activated before this preflight completes.

---

## 13. Completion Criteria Checklist

```
SB8 Dry-Run Telemetry Completion:
  [x] FP 4건 (OI-033/034/035/036): FP_NOT_REPRODUCED confirmed
  [x] FN 4건 (OI-020/021/037/038): FN_CLOSED confirmed
  [x] operational-evidence.csv: all actual_observed_* (no not_collected/planned_only)
  [x] samples.csv: policy fields aligned with observed behavior
  [x] Both CSV integrity audits: PASS
  [x] FP/FN resolution artifact: present and verified
  [x] SB8 CLEARED preflight: PASS
  [x] SB8 CLEARED declaration: THIS ARTIFACT

Post-SB8 (requires separate authorization):
  [ ] env=enforce: NOT YET — preflight required
  [ ] git add/commit: NOT YET — prohibited
  [ ] git push/PR: NOT YET — prohibited
  [ ] UI01~UI11: NOT YET — separate Studio UI track
```
