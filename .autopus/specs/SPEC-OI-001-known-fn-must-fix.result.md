# SPEC-OI-001 Known-FN Must-Fix Result

Make Variable Push Detection Quality Signal

## 1. Title

SPEC-OI-001 Known-FN Must-Fix Result.

Make Variable Push Detection Quality Signal.

Records the implementation and test-green quality signal of the make-recipe variable-push must-fix. This is a documentation record, not an operational observation.

## 2. Status

- Documentation artifact only.
- Quality signal only.
- Operational observation evidence: NO.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.
- git action: NO.

## 3. Purpose

- Record the make-variable must-fix implementation and `go test ./pkg/guard` green result.
- Keep the quality signal separate from operational observation evidence.
- Provide a basis for later policy / review decisions without mutating the source CSV datasets.
- Prevent direct edits to the existing operational-intent CSV files.

## 4. Scope

- OI-020 (variable-expanded git push).
- OI-021 (nested make variable push).
- OI-037 (variable force push).
- OI-034 (read-only help target, false-positive control).
- OI-033 / OI-035 / OI-036 (operational-only, unaffected by the make-variable fix).
- OI-038 (alias-indirection follow-up).

## 5. Non-goals

- No FP_REVIEW resolution.
- No SB8 clear.
- No release gate clear.
- No env=enforce.
- No operational evidence collection.
- No CSV relabeling.
- No implementation changes.
- No test execution in this turn.

## 6. Source inputs

- `.autopus/specs/SPEC-OI-001-known-fn-must-fix.md` (the must-fix SPEC).
- `pkg/guard/make_inspector.go` (implementation).
- `pkg/guard/make_inspector_known_fn_test.go` (tests-first regression, now green).
- `pkg/guard/make_inspector_t12_c_test.go` (known-FN baseline flip).
- `pkg/guard/make_inspector_test.go` (third known-FN baseline flip).
- Prior `go test ./pkg/guard` result: ok (previous turn). NOTE: go test is NOT re-run in this turn.

## 7. Implementation summary

- limited one-step make variable expansion (`NAME=value`, `NAME := value` → `$(NAME)`/`${NAME}` expanded exactly once).
- narrow deny-on-uncertain fallback (unresolved variable next to a dangerous token fails closed).
- static-only analysis (in-memory Makefile text).
- no actual make execution.
- no shell execution.
- no git execution.
- no network / provider.
- no full Makefile interpreter (recursive/`$(shell)`/include/conditional remain uncertain → fail-closed when dangerous context is present).

## 8. Test green quality signal

| sample_or_test | expected behavior | quality signal | caveat |
|---|---|---|---|
| OI-020 `$(GIT) push` | deny/fail-closed | green | static/test signal, not observed evidence |
| OI-021 nested `$(TOOL)=$(MAKE)` push | deny/fail-closed | green | static/test signal, not observed evidence |
| OI-037 `$(GIT) push --force` | deny/fail-closed | green | static/test signal, not observed evidence |
| OI-034 help echo | allow/not dangerous | green | FP-control; static/test signal only |
| t12_c baseline flip (A2 / KnownFalseNegativeClosed) | deny | green | intentional known-FN closure |
| make_inspector_test baseline flip (VariableObfuscationNowDetected) | deny | green | intentional known-FN closure |
| `go test ./pkg/guard` | ok | green (prior turn) | not re-run this turn; quality signal only |

## 9. Observed evidence caveat

- operational-evidence.csv `observed_decision` = not_collected.
- operational-evidence.csv `observed_guard_id_or_phase` = not_collected.
- operational-evidence.csv `observed_status` = planned_only.
- Package test green is NOT operational observation evidence.
- Static / test signal is NOT acceptance evidence.
- FP_REVIEW resolved: NO.
- enforce gate allowed: NO.

## 10. Review target status matrix

| sample_id | bucket | quality signal | observed evidence | recommended current classification | remaining blocker | next action |
|---|---|---|---|---|---|---|
| OI-020 | policy-primary | must-fix deny green | not_collected | quality signal complete, policy pending | policy decision | policy 4 preflight |
| OI-021 | policy-primary | must-fix deny green | not_collected | quality signal complete, policy pending | policy decision | policy 4 preflight |
| OI-037 | policy+operational | deny/fail-closed green | not_collected | split still pending | split decision | policy 4 preflight |
| OI-034 | policy+operational | FP-control allow green | not_collected | split still pending | split decision | policy 4 preflight |
| OI-033 | operational-only | unaffected by make-variable fix | not_collected | operational pending | observation deferred (module-graph cost) | operational deferred |
| OI-035 | operational-only | unaffected by make-variable fix | not_collected | operational pending | observation deferred | operational deferred |
| OI-036 | operational-only | unaffected by make-variable fix | not_collected | operational pending | observation deferred | operational deferred |
| OI-038 | follow-up | not covered | not_collected | alias-indirection follow-up | separate SPEC | OI-038 follow-up preflight |

## 11. Policy status

- policy_decision_required=4 (OI-020, OI-021, OI-034, OI-037).
- OI-020 / OI-021 (must_fix_before_enforce): implementation quality signal completed, but the policy decision is still pending.
- OI-034 / OI-037 (split_policy_and_operational): still pending.
- No direct policy.csv modification.
- No resolution.csv modification.
- No review.csv modification.
- No samples.csv modification.

## 12. OI-038 follow-up

- OI-038 is outside the make-variable fix.
- Issue type: git alias / command-line indirection (`git -c alias.x=push x`).
- Requires a separate SPEC or policy track (not covered by make recipe variable expansion).
- No implementation now.
- Kept as a follow-up candidate after this result artifact.

## 13. Remaining blockers

- review_target 8 pending.
- observed evidence not collected.
- policy 4 unresolved.
- OI-038 follow-up unresolved.
- M3 / M4 inert.
- production YAML source / decoder absent.
- UI01–UI11 verified:false.
- env=enforce gate incomplete.
- SB8 CLEARED: NO.
- release gate BLOCKED.

## 14. Enforcement gate blockers

- N-β enforce gate not allowed.
- no env=enforce.
- no SB8 clear.
- no release ready.
- no FP_REVIEW resolved.
- no public / GitHub action.

## 15. Completion criteria

- result artifact created exactly once.
- no source / test / config changes.
- no existing CSV changes.
- no existing SPEC changes.
- no go test / go command.
- no pure API / hook / probe.
- no git add / commit / push / PR.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.

## 16. Next recommended tracks

Comparison / order:

1. **policy 4건 확정 preflight** — OI-020 / OI-021 / OI-034 / OI-037 policy decisions, using this artifact's quality signal as the basis.
2. OI-038 alias-indirection follow-up SPEC preflight.
3. operational-only observation (OI-033 / OI-035 / OI-036) remains blocked / deferred due to the module-graph cost.
4. N-β enforce gate remains blocked.

**Designated next recommended track: policy 4건 확정 preflight.**
Reason: this artifact provides the quality-signal basis, so the policy status of OI-020 / OI-021 / OI-034 / OI-037 can be settled first. The OI-038 follow-up is kept as the subsequent candidate.
