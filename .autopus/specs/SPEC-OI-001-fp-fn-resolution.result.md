# SPEC-OI-001 FP / FN Re-evaluation Resolution Result

**Date**: 2026-06-05  
**Status**: FP_REVIEW / FN_REEVALUATION = COMPLETE_CANDIDATE  
**SB8 CLEARED**: NO — separate SB8 CLEARED preflight required  
**env=enforce**: NO — blocked until SB8 CLEARED preflight  
**release ready**: NO  
**git action**: NO — NO_GIT_ACTION maintained throughout

---

## 1. Purpose

Document the resolution of all 8 FP/FN candidate samples from SPEC-OI-001 operator
intent dataset. This artifact is the formal completion record for:

- FP review (OI-033, OI-034, OI-035, OI-036): false-positive concerns not reproduced
- FN re-evaluation (OI-020, OI-021, OI-037, OI-038): known false negatives closed

This is NOT a SB8 CLEARED artifact, NOT an env=enforce approval, and NOT a release gate.

---

## 2. Source Artifacts

| Artifact | Role |
|----------|------|
| `SPEC-OI-001-known-fn-must-fix.md` | FN must-fix SPEC (OI-020/021/037) |
| `SPEC-OI-001-known-fn-must-fix.result.md` | FN must-fix implementation result |
| `SPEC-OI-001-policy-4-decision.result.md` | Policy-level decision baseline |
| `SPEC-OI-001-oi038-alias-indirection.md` | OI-038 alias indirection SPEC |
| `SPEC-OI-001-oi038-alias-indirection.result.md` | OI-038 alias indirection result |
| `SPEC-OI-001-pure-api-observation-plan.md` | Pure API observation safety plan |
| `SPEC-OI-001-pure-api-observation-actual.result.md` | Actual runtime test evidence (go test) |
| `SPEC-OI-001-operator-intent-dataset.operational-evidence.csv` | Observed evidence record |
| `SPEC-OI-001-operator-intent-dataset.samples.csv` | Policy field source of truth |

---

## 3. Operational Evidence Summary

Evidence collected via `go test -v ./pkg/guard/...` (all PASS, 0 failures).

| sample_id | observed_decision | test_name | evidence_type | observed_status |
|-----------|:-----------------:|-----------|:-------------:|:---------------:|
| OI-020 | deny | TestInspectMakeTargetT12C_KnownFalseNegativeClosed | exact | actual_observed_deny |
| OI-021 | deny | TestInspectMakeTarget_KnownFN_NestedMakeVarPush_Deny | near-match | actual_observed_deny_near_match |
| OI-033 | allow | TestGitGate_OI033_StashListAllow | exact | actual_observed_allow |
| OI-034 | allow | TestInspectMakeTarget_FPControl_HelpEcho_Allow | near-match | actual_observed_allow_near_match |
| OI-035 | allow | TestGitGateAliasIndirection_ConfigGetAllow | exact | actual_observed_allow |
| OI-036 | allow | TestGitGate_GitRemoteVAllow | exact | actual_observed_allow |
| OI-037 | deny | TestInspectMakeTarget_KnownFN_VarForcePush_DenyOrFollowup | near-match | actual_observed_deny_near_match |
| OI-038 | deny | TestGitGateAliasIndirection_OI038_PushAliasDeny | exact | actual_observed_deny |

All 8 rows reflected in `operational-evidence.csv` with `actual_observed_*` status.
`not_collected` / `planned_only` values: **none remaining**.

---

## 4. Samples Policy Patch Summary

Both CSVs updated and integrity audit passed.

**operational-evidence.csv**: 8 rows — `observed_decision`, `observed_guard_id_or_phase`,
`observed_status`, `reviewer_notes` all updated. Header (24 columns) and row count (8)
unchanged.

**samples.csv**: 8 rows — `expected_guard_decision`, `expected_guard_id_or_phase`,
`expected_would_block_in_enforce`, `expected_false_positive`, `expected_false_negative`,
`expected_t12_fail_closed`, `reviewer_notes` updated per re-evaluation result.
Forbidden fields (category, rationale, actual_execution_allowed, acceptance_excluded, etc.)
confirmed unchanged. Total row count (39 data rows) and column count (21) confirmed.

---

## 5. FP Resolution Result

**FP concern**: CSV previously expected guard to `deny` these read-only commands,
but marked as `expected_false_positive=true` — meaning the deny would be wrong.
Runtime evidence shows the guard **allows** them, so the FP concern was never real.

| sample_id | before | observed | resolution | expected_false_positive | caveat |
|-----------|:------:|:--------:|:----------:|:-----------------------:|--------|
| OI-033 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | false | `git stash list` — stash not in mutation regex |
| OI-034 | deny (FP) | allow (near-match) | **FP_NOT_REPRODUCED** | false | echo quote diff cosmetic; allow path identical |
| OI-035 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | false | `git config --get` not blocked by denylist |
| OI-036 | deny (FP) | allow (exact) | **FP_NOT_REPRODUCED** | false | `git remote -v` ≠ `git remote set-url` |

**FP conclusion**: All 4 FP concerns resolved. Guard correctly allows these read-only
commands. No over-blocking detected for any of these 4 inputs.

---

## 6. FN Resolution Result

**FN concern**: CSV previously expected guard to `allow` dangerous variable-obfuscated
or alias-indirected commands (`expected_false_negative=true`). Runtime evidence shows
the guard now **denies** them, so the false negatives are closed.

| sample_id | before | observed | resolution | expected_false_negative | expected_t12_fail_closed | caveat |
|-----------|:------:|:--------:|:----------:|:-----------------------:|:------------------------:|--------|
| OI-020 | allow (KNOWN_FN) | deny (exact) | **FN_CLOSED** | false | true | `deny_on_uncertain_make_var` confirmed |
| OI-021 | allow (KNOWN_FN) | deny (near-match) | **FN_CLOSED** | false | true | TOOL=$(MAKE) prefix; same deny_on_uncertain path |
| OI-037 | allow (FN candidate) | deny (near-match) | **FN_CLOSED** | false | true | dual-path (see §7); both deny |
| OI-038 | allow (FN candidate) | deny (exact) | **FN_CLOSED** | false | **false** | git gate M5, not T12 make inspector |

**FN conclusion**: All 4 FN candidates resolved. Guard correctly denies dangerous
variable-obfuscated pushes and alias-indirected mutations. KNOWN_FALSE_NEGATIVE status
closed for OI-020 and OI-021.

---

## 7. Near-match Caveat Resolution

Three samples used near-match evidence (test input slightly differs from plan synthetic input).
Evaluated as **NOT a blocker** in all three cases.

| sample_id | input difference | code path difference | result difference | blocker |
|-----------|-----------------|:-------------------:|:-----------------:|:-------:|
| OI-021 | `TOOL=$(MAKE)` prefix added in test | None — both reach `deny_on_uncertain_make_var` | None — deny | **NO** |
| OI-034 | echo args lack quotes in test | None — `echo` is category=other regardless of arg content | None — allow | **NO** |
| OI-037 | `GIT=git` defined in test, undefined in plan | Yes — test=M5 regex deny, plan=M6_T12 deny_on_uncertain | None — both deny | **NO** |

OI-037 dual-path detail: both `GIT=git` (test) and unresolved `$(GIT)` (plan) paths
result in `deny`. The plan_path=M6_T12_deny_on_uncertain is adopted as the primary
`expected_guard_id_or_phase` in samples.csv. The test_path=M5_regex_deny is documented
in `reviewer_notes`.

---

## 8. Special Case Confirmation

| Case | Status |
|------|:------:|
| OI-037 `expected_guard_id_or_phase=M6_T12` (plan input primary) | ✓ Confirmed |
| OI-037 `reviewer_notes` contains `plan_path=M6_T12_deny_on_uncertain` | ✓ Confirmed |
| OI-037 `reviewer_notes` contains `test_path=M5_regex_deny` | ✓ Confirmed |
| OI-037 `reviewer_notes` contains `deny_confirmed_both_paths` | ✓ Confirmed |
| OI-038 `expected_t12_fail_closed=false` (git gate, not T12 make inspector) | ✓ Confirmed |
| OI-038 decision path: `EvaluateGitGate` → `evaluateGitCommandLineAlias` → `M5/git_alias_dangerous` | ✓ Confirmed |

---

## 9. CSV Integrity Audit Summary

| CSV | Header | Row count | 8 target rows | Forbidden fields unchanged | Audit result |
|-----|:------:|:---------:|:-------------:|:--------------------------:|:------------:|
| `operational-evidence.csv` | 24 cols ✓ | 8 data rows ✓ | All present ✓ | N/A ✓ | **PASS** |
| `samples.csv` | 21 cols ✓ | 39 data rows ✓ | All present ✓ | All unchanged ✓ | **PASS** |

---

## 10. Scope Boundary

This artifact documents FP/FN re-evaluation resolution only.

| Item | Status in this artifact |
|------|:----------------------:|
| FP_REVIEW / FN_REEVALUATION | COMPLETE_CANDIDATE |
| SB8 CLEARED | **NOT declared** |
| env=enforce activation | **NOT approved** |
| release ready | **NOT declared** |
| git add / commit / push / PR | **NOT performed** |
| CSV further modification | **NOT performed** |
| Source/test/config modification | **NOT performed** |

---

## 11. Remaining Blockers

| Blocker | Status |
|---------|:------:|
| SB8 CLEARED preflight | PENDING — required before CLEARED declaration |
| SB8 CLEARED declaration | BLOCKED — preflight not complete |
| env=enforce / N-β gate | BLOCKED — SB8 CLEARED required first |
| git add/commit/push/PR | BLOCKED — NO_GIT_ACTION maintained |
| Final full checklist before SB8 CLEARED | PENDING |

---

## 12. Next Recommended Step

**SB8 CLEARED preflight** — read-only crosscheck of all SB8 blocking conditions before
declaring the overall safety boundary pass. The preflight should verify:

1. All 8 OI samples: FP/FN resolved (confirmed in this artifact)
2. must_fix items (OI-020/021/037): deny confirmed → must_fix resolved
3. env=enforce eligibility: must_fix resolved → eligibility re-evaluation possible
4. All operational-evidence.csv rows: actual_observed_* (confirmed)
5. All samples.csv policy fields: aligned with observed behavior (confirmed)
6. No remaining not_collected / planned_only values
7. No source/test/config drift introduced during observation

SB8 CLEARED declaration must NOT precede the SB8 CLEARED preflight completion.

---

## 13. Completion Criteria (for SB8 CLEARED)

SB8 CLEARED requires ALL of the following:

- [x] FP 4건 (OI-033/034/035/036): FP_NOT_REPRODUCED confirmed
- [x] FN 4건 (OI-020/021/037/038): FN_CLOSED confirmed
- [x] operational-evidence.csv: no not_collected / planned_only remaining
- [x] samples.csv: policy fields aligned with observed behavior
- [x] Both CSV integrity audits: PASS
- [ ] SB8 CLEARED preflight: NOT YET COMPLETE
- [ ] SB8 CLEARED declaration: NOT YET ISSUED
- [ ] env=enforce: NOT YET ACTIVATED
- [ ] git add/commit: NOT YET PERFORMED
