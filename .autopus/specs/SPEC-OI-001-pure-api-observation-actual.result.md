# SPEC-OI-001 Actual Runtime Test Evidence — Result

**Status**: ACTUAL_OBSERVED_EVIDENCE — NOT A SB8 CLEAR ARTIFACT  
**Date**: 2026-06-04  
**Method**: `go test -v ./pkg/guard/...` — actual runtime execution of pure guard functions  
**Related**: SPEC-OI-001-pure-api-observation-plan.md, operational-evidence.csv, samples.csv

---

## 1. Purpose

Collect actual runtime test evidence for 8 OI samples (OI-020, OI-021, OI-033, OI-034,
OI-035, OI-036, OI-037, OI-038) via `go test`, replacing the prior static trace artifact
with actual observed decisions.

---

## 2. Method

Single execution: `go test -v ./pkg/guard/...`  
Scope: `pkg/guard` and `pkg/guard/telemetry` packages only.  
No `go test ./...`. No `go run`. No `go env`. No `go list`. No `go mod`.

---

## 3. Safety Boundary

| Boundary | Status |
|----------|--------|
| `go run` / `go env` / `go list` / `go mod` | NOT EXECUTED |
| `go test ./...` (full repo) | NOT EXECUTED — scoped to `./pkg/guard/...` only |
| subprocess within tests | NOT INVOKED (test files explicitly state NO exec.Command) |
| network / DNS / HTTP | NOT INVOKED |
| actual `git` / `make` / `sh` / `bash` | NOT INVOKED |
| hook / probe path | NOT INVOKED |
| provider CLI | NOT INVOKED |
| `git add` / `commit` / `push` / `PR` | NOT PERFORMED |
| `env=enforce` | NOT ACTIVATED |
| existing CSV / SPEC / result modification | NOT PERFORMED |

---

## 4. go test Command and Result

```
Command:  go test -v ./pkg/guard/...
Exit:     0 (all tests passed)
Duration: pkg/guard 1.237s  |  pkg/guard/telemetry 1.261s
Result:   PASS
```

**Full outcome**: `ok github.com/insajin/autopus-adk/pkg/guard 1.237s`  
**Failed tests**: 0  
**Total packages**: 2 (`pkg/guard`, `pkg/guard/telemetry`)

---

## 5. OI Sample → Test Mapping

| sample_id | test_name | test_file | input_match |
|-----------|-----------|-----------|:-----------:|
| OI-020 | `TestInspectMakeTargetT12C_KnownFalseNegativeClosed` | `make_inspector_t12_c_test.go` | **EXACT** |
| OI-020 (2nd) | `TestInspectMakeTargetT12C/A2-ambiguous-failopen-known-false-negative` | `make_inspector_t12_c_test.go` | **EXACT** |
| OI-020 (3rd) | `TestInspectMakeTarget_KnownFN_VarGitPush_Deny` | `make_inspector_known_fn_test.go` | NEAR (GIT=git defined) |
| OI-021 | `TestInspectMakeTarget_KnownFN_NestedMakeVarPush_Deny` | `make_inspector_known_fn_test.go` | NEAR (TOOL=$(MAKE) prefix) |
| OI-033 | `TestGitGate_OI033_StashListAllow` | `git_gate_oi033_stash_test.go` | **EXACT** |
| OI-034 | `TestInspectMakeTarget_FPControl_HelpEcho_Allow` | `make_inspector_known_fn_test.go` | NEAR (no quotes) |
| OI-035 | `TestGitGateAliasIndirection_ConfigGetAllow` | `git_gate_alias_indirection_test.go` | **EXACT** |
| OI-036 | `TestGitGate_GitRemoteVAllow` | `git_gate_test.go` | **EXACT** |
| OI-036 (2nd) | `TestGitGateAliasIndirection_RemoteVAllow` | `git_gate_alias_indirection_test.go` | **EXACT** |
| OI-037 | `TestInspectMakeTarget_KnownFN_VarForcePush_DenyOrFollowup` | `make_inspector_known_fn_test.go` | NEAR (GIT=git defined) |
| OI-038 | `TestGitGateAliasIndirection_OI038_PushAliasDeny` | `git_gate_alias_indirection_test.go` | **EXACT** |

**EXACT**: test synthetic input matches plan input exactly.  
**NEAR**: test uses slightly different makefile text (e.g., variable definition prefix added); expected decision is identical.

---

## 6. Actual Runtime Test Evidence Table

| sample_id | test_name | source_function_under_test | synthetic_input_summary | asserted_decision | runtime_test_result | observed_status | expected_from_samples_csv | delta_from_samples_csv | caveat |
|-----------|-----------|---------------------------|------------------------|-------------------|---------------------|-----------------|--------------------------|------------------------|--------|
| OI-020 | `TestInspectMakeTargetT12C_KnownFalseNegativeClosed` | `InspectMakeTarget` | `"release:\n\t$(GIT) push origin main"`, `"release"` | deny | **PASS** | `actual_observed_deny` | `allow` (KNOWN_FALSE_NEGATIVE) | DELTA: csv=allow(FN) → runtime=deny. Quality improvement confirmed. | EXACT input match. Two additional tests also confirm deny via near-match inputs. |
| OI-021 | `TestInspectMakeTarget_KnownFN_NestedMakeVarPush_Deny` | `InspectMakeTarget` | `"TOOL=$(MAKE)\nship:\n\t$(TOOL) -C sub push\n"`, `"ship"` | deny | **PASS** | `actual_observed_deny_near_match` | `allow` (KNOWN_FALSE_NEGATIVE) | DELTA: csv=allow(FN) → runtime=deny. Quality improvement confirmed. | Test input has `TOOL=$(MAKE)` prefix (near match). Deny via `deny_on_uncertain_make_var` after one-step TOOL expansion leaves `$(MAKE)` unresolved. |
| OI-033 | `TestGitGate_OI033_StashListAllow` | `EvaluateGitGate` | `executable="git"`, `args=["stash","list"]` | allow | **PASS** | `actual_observed_allow` | `deny` (FP candidate) | DELTA: csv=deny(FP) → runtime=allow. FP concern not reproduced in current code. | EXACT input match. `stash` not in mutation denylist regex. |
| OI-034 | `TestInspectMakeTarget_FPControl_HelpEcho_Allow` | `InspectMakeTarget` | `"help:\n\t@echo available targets\n"`, `"help"` | allow | **PASS** | `actual_observed_allow_near_match` | `deny` (FP candidate) | DELTA: csv=deny(FP) → runtime=allow. FP concern not reproduced. | Test input lacks double-quotes around `"Available targets"` (near match). Both forms have no dangerous token; allow confirmed. |
| OI-035 | `TestGitGateAliasIndirection_ConfigGetAllow` | `EvaluateGitGate` | `executable="git"`, `args=["config","--get","user.name"]` | allow | **PASS** | `actual_observed_allow` | `deny` (FP candidate) | DELTA: csv=deny(FP) → runtime=allow. FP concern not reproduced. | EXACT input match. `config` not a mutation verb; no regex match. |
| OI-036 | `TestGitGate_GitRemoteVAllow` | `EvaluateGitGate` | `executable="git"`, `args=["remote","-v"]` | allow | **PASS** | `actual_observed_allow` | `deny` (FP candidate) | DELTA: csv=deny(FP) → runtime=allow. FP concern not reproduced. | EXACT input match. `remote set-url` pattern does NOT match `remote -v`. Second test (`TestGitGateAliasIndirection_RemoteVAllow`) also confirms allow. |
| OI-037 | `TestInspectMakeTarget_KnownFN_VarForcePush_DenyOrFollowup` | `InspectMakeTarget` | `"GIT=git\nrelease:\n\t$(GIT) push --force\n"`, `"release"` | deny | **PASS** | `actual_observed_deny_near_match` | `allow` (FN candidate) | DELTA: csv=allow(FN) → runtime=deny. Quality improvement confirmed. | Test input has `GIT=git` defined (near match). After expansion: `git push --force` → M5 mutation regex match → deny. Plan input (no GIT defined) would deny via `deny_on_uncertain_make_var` instead. Both paths deny. |
| OI-038 | `TestGitGateAliasIndirection_OI038_PushAliasDeny` | `EvaluateGitGate` | `executable="git"`, `args=["-c","alias.x=push","x"]` | deny | **PASS** | `actual_observed_deny` | `allow` (FN candidate) | DELTA: csv=allow(FN) → runtime=deny. Quality improvement confirmed. | EXACT input match. Alias value `"push"` → `isMutationGitVerb=true` → `git_alias_dangerous` → deny fail-closed. |

---

## 7. Expected-vs-Test-Result Caveat

**Classification upgrade from prior static trace artifact:**

| sample_id | prior status (static trace) | current status (runtime test) | upgrade |
|-----------|----------------------------|-------------------------------|---------|
| OI-020 | `static_inferred_decision` | `actual_observed_deny` | ✓ UPGRADED |
| OI-021 | `static_inferred_decision` | `actual_observed_deny_near_match` | ✓ UPGRADED |
| OI-033 | `static_inferred_decision` | `actual_observed_allow` | ✓ UPGRADED |
| OI-034 | `static_inferred_decision` | `actual_observed_allow_near_match` | ✓ UPGRADED |
| OI-035 | `static_inferred_decision` | `actual_observed_allow` | ✓ UPGRADED |
| OI-036 | `static_inferred_decision` | `actual_observed_allow` | ✓ UPGRADED |
| OI-037 | `static_inferred_decision` | `actual_observed_deny_near_match` | ✓ UPGRADED |
| OI-038 | `static_inferred_decision` | `actual_observed_deny` | ✓ UPGRADED |

**Near-match caveat for OI-021/034/037**: Test inputs differ from plan synthetic inputs
but yield identical expected decisions. The near-match inputs provide actual runtime
evidence of the same guard behavior. CSV update preflight must note the input difference.

---

## 8. CSV Mutation Boundary

**HARD RULE**: operational-evidence.csv and samples.csv were NOT modified in this observation.

Candidate CSV field updates (NOT applied — requires separate CSV update preflight):

| sample_id | `observed_decision` | `observed_guard_id_or_phase` | `observed_status` |
|-----------|--------------------|-----------------------------|-------------------|
| OI-020 | deny | M6_T12/deny_on_uncertain_make_var (plan) + M5 (near-match tests) | actual_observed_deny |
| OI-021 | deny | M6_T12/deny_on_uncertain_make_var | actual_observed_deny_near_match |
| OI-033 | allow | P8a_git_gate_allow_candidate | actual_observed_allow |
| OI-034 | allow | P8a_make_inspector_allow_candidate | actual_observed_allow_near_match |
| OI-035 | allow | P8a_git_gate_allow_candidate | actual_observed_allow |
| OI-036 | allow | P8a_git_gate_allow_candidate | actual_observed_allow |
| OI-037 | deny | M5/git_push_mutation (near-match) | actual_observed_deny_near_match |
| OI-038 | deny | M5/git_alias_dangerous | actual_observed_deny |

---

## 9. FP_REVIEW Readiness Impact

| sample_id | FP_REVIEW impact | runtime evidence basis |
|-----------|-----------------|------------------------|
| OI-033 | FP concern CONFIRMED NOT REPRODUCED — `allow` observed at runtime | `TestGitGate_OI033_StashListAllow` PASS |
| OI-034 | FP concern CONFIRMED NOT REPRODUCED — `allow` observed at runtime | `TestInspectMakeTarget_FPControl_HelpEcho_Allow` PASS |
| OI-035 | FP concern CONFIRMED NOT REPRODUCED — `allow` observed at runtime | `TestGitGateAliasIndirection_ConfigGetAllow` PASS |
| OI-036 | FP concern CONFIRMED NOT REPRODUCED — `allow` observed at runtime | `TestGitGate_GitRemoteVAllow` PASS |

FP_REVIEW formal resolution is NOT performed here. Requires separate CSV update preflight approval.

**OI-020/OI-021/OI-037/OI-038**: Were `must_fix_before_enforce` / FN candidates.  
Runtime tests confirm `deny` for all 4. `env=enforce` eligibility re-evaluation and `must_fix`
status update require separate approval. NOT performed here.

---

## 10. Remaining Blockers

| Blocker | Status |
|---------|--------|
| operational-evidence.csv not updated | PENDING — CSV update preflight required |
| samples.csv `expected_guard_decision` delta not reconciled | PENDING — CSV update preflight required |
| FP_REVIEW formally resolved | BLOCKED — not performed here |
| SB8 CLEARED | BLOCKED — evidence record only; not a clear artifact |
| `env=enforce` | BLOCKED — must_fix re-evaluation not complete |
| `git add` / `commit` / `push` / `PR` | BLOCKED — NO_GIT_ACTION maintained |

---

## 11. Next Recommended Step

**CSV update preflight** — define exact field-by-field changes to operational-evidence.csv
and samples.csv based on actual runtime evidence above, confirm approval, then apply.

Sequence:
1. CSV update preflight (define changes, no apply)
2. CSV update apply (apply approved changes only)
3. FP_REVIEW re-evaluation (OI-033/034/035/036) — after CSV update
4. must_fix status re-evaluation (OI-020/021/037/038) — after CSV update
5. `env=enforce` eligibility re-evaluation — after must_fix re-evaluation

---

## 12. Evidence Checklist

- [x] `go test -v ./pkg/guard/...` executed exactly once
- [x] All tests PASS (0 failures)
- [x] 8/8 OI samples covered (exact or near-match)
- [x] No subprocess within tests (confirmed by test file scope comments)
- [x] No network / actual git / make / shell invoked
- [x] `go test ./...` NOT executed (scoped to `./pkg/guard/...` only)
- [x] `go run` / `go env` / `go list` / `go mod` NOT executed
- [x] `env=enforce` NOT activated
- [x] `git add` / `commit` / `push` / `PR` NOT performed
- [x] existing CSV NOT modified
- [x] FP_REVIEW NOT resolved
- [x] SB8 CLEARED NOT declared
