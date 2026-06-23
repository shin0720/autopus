# SPEC-OI-001 Policy 4 Decision Result

Quality Signal Based Policy Decision Proposal

## 1. Title

SPEC-OI-001 Policy 4 Decision Result.

Quality Signal Based Policy Decision Proposal.

## 2. Status

- Documentation artifact only.
- Policy decision proposal only.
- Operational observation evidence: NO.
- CSV mutation: NO.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.
- git action: NO.

## 3. Purpose

- Fix the recommended policy decisions for OI-020 / OI-021 / OI-034 / OI-037 in a reviewable document before any CSV mutation.
- Keep quality signal and observed evidence strictly separated.
- Provide review material before directly modifying existing CSV files.
- Document blockers that prevent moving to the N-beta enforce gate.

## 4. Scope

- OI-020.
- OI-021.
- OI-034.
- OI-037.
- Includes the review_target 8 matrix.
- Includes the OI-038 follow-up position.

## 5. Non-goals

- no CSV mutation.
- no operational observation collection.
- no FP_REVIEW resolution.
- no SB8 clear.
- no release gate clear.
- no env=enforce.
- no implementation changes.
- no test execution in this turn.
- no git add/commit/push/PR.

## 6. Source inputs

- `.autopus/specs/SPEC-OI-001-known-fn-must-fix.result.md`.
- `.autopus/specs/SPEC-OI-001-known-fn-must-fix.md`.
- Previous `go test ./pkg/guard` result: ok.
- Policy 4 decision preflight.
- Read-only git evidence correction result.
- NOTE: `go test` is not re-run in this turn.

## 7. Evidence boundary

- Quality signal available.
- Observed evidence not collected.
- operational-evidence.csv `observed_decision` remains `not_collected`.
- operational-evidence.csv `observed_guard_id_or_phase` remains `not_collected`.
- operational-evidence.csv `observed_status` remains `planned_only`.
- Package test green is not operational observation evidence.
- Static/test signal is not acceptance evidence.
- Enforce gate not allowed.

## 8. Policy target summary

| sample_id | previous_policy_status | bucket | quality_signal | observed_evidence_status | proposed_policy_decision | caveat |
|---|---|---|---|---|---|---|
| OI-020 | must_fix_before_enforce | policy-primary | must-fix implementation/test green | not_collected | ready_to_record_quality_signal_not_observed | observed evidence is not collected |
| OI-021 | must_fix_before_enforce | policy-primary | must-fix implementation/test green | not_collected | ready_to_record_quality_signal_not_observed | observed evidence is not collected |
| OI-034 | split_policy_and_operational | policy+operational | FP-control allow green | not_collected | split_quality_signal_supports_fp_control_policy_pending | operational split unresolved |
| OI-037 | split_policy_and_operational | policy+operational | deny/fail-closed green | not_collected | split_quality_signal_supports_deny_policy_pending | operational split unresolved |

## 9. Policy decision table

| sample_id | recommended_policy_decision | rationale | allowed_next_action | forbidden_action | remaining_blocker |
|---|---|---|---|---|---|
| OI-020 | ready_to_record_quality_signal_not_observed | The `$(GIT) push` make-variable case has deny/fail-closed quality signal complete. | Review this proposal in a separate CSV update preflight. | Relabel observed evidence from test signal. | observed evidence not collected |
| OI-021 | ready_to_record_quality_signal_not_observed | The nested make-variable push case has deny/fail-closed quality signal complete. | Review this proposal in a separate CSV update preflight. | Directly mutate policy/resolution CSV in this turn. | observed evidence not collected |
| OI-034 | split_quality_signal_supports_fp_control_policy_pending | The help echo FP-control case has allow/not dangerous quality signal available. | Keep split policy documented and defer operational observation. | Treat FP-control quality signal as operational evidence. | split policy/operational decision remains pending |
| OI-037 | split_quality_signal_supports_deny_policy_pending | The force-push make-variable case has deny/fail-closed quality signal available. | Keep split policy documented and defer operational observation. | Move to enforce gate from static/test signal. | split policy/operational decision remains pending |

## 10. Review target 8 matrix

| sample_id | bucket | policy_decision_required | quality_signal_status | observed_evidence_status | current_blocker | next_action |
|---|---|---|---|---|---|---|
| OI-020 | policy-primary | YES | quality signal complete; policy decision artifact created | not_collected | observed evidence not collected | CSV policy update preflight candidate |
| OI-021 | policy-primary | YES | quality signal complete; policy decision artifact created | not_collected | observed evidence not collected | CSV policy update preflight candidate |
| OI-033 | operational-only | NO | unaffected by policy 4 result | not_collected | operational observation deferred | keep deferred |
| OI-034 | policy+operational | YES | split policy decision documented; quality signal available | not_collected | operational split unresolved | CSV policy update preflight candidate |
| OI-035 | operational-only | NO | unaffected by policy 4 result | not_collected | operational observation deferred | keep deferred |
| OI-036 | operational-only | NO | unaffected by policy 4 result | not_collected | operational observation deferred | keep deferred |
| OI-037 | policy+operational | YES | split policy decision documented; quality signal available | not_collected | operational split unresolved | CSV policy update preflight candidate |
| OI-038 | follow-up | NO | not covered | not_collected | separate SPEC/policy track needed | alias-indirection SPEC preflight |

## 11. CSV mutation boundary

- no policy.csv modification.
- no resolution.csv modification.
- no review.csv modification.
- no samples.csv modification.
- no operational-evidence.csv modification.
- Any future CSV update must be a separate preflight/patch turn.
- Do not relabel observed evidence from quality signal.

## 12. OI-038 follow-up

- OI-038 is outside make-variable fix.
- Issue type: git alias / command-line indirection.
- Requires separate SPEC or policy track.
- Not implemented now.
- Keep as follow-up after policy 4 result artifact.

## 13. Remaining blockers

- review_target 8 pending.
- observed evidence not collected.
- operational observation deferred.
- OI-038 follow-up unresolved.
- M3/M4 inert.
- production YAML source/decoder absent.
- UI01~UI11 verified:false.
- env=enforce gate incomplete.
- SB8 CLEARED: NO.
- release gate BLOCKED.

## 14. Enforcement gate blockers

- N-beta enforce gate not allowed.
- no env=enforce.
- no SB8 clear.
- no release ready.
- no FP_REVIEW resolved.
- no public/GitHub action.
- no git add/commit/push/PR.

## 15. Next recommended track

| candidate | summary | recommendation |
|---|---|---|
| A. OI-038 alias-indirection SPEC preflight | Follow up the remaining alias / command-line indirection gap outside the make-variable fix. | Recommended |
| B. operational-only observation retry | Would target OI-033 / OI-035 / OI-036, but module graph and observation cost remain blockers. | Defer |
| C. CSV policy update preflight | Can come later, but observed evidence caveats still require careful separation. | Defer |
| D. N-beta enforce gate | Not allowed while blockers remain. | Forbidden |

Recommended next track: A. OI-038 alias-indirection SPEC preflight.

Reason: after the policy 4 quality-signal decision artifact is created, the remaining clear follow-up is OI-038 alias-indirection. CSV update remains a separate candidate, and the N-beta enforce gate remains blocked.

## 16. Completion criteria

- result artifact created exactly once.
- no source/test/config changes.
- no existing CSV changes.
- no existing SPEC/result changes.
- no go test/go command.
- no pure API/hook/probe.
- no git add/commit/push/PR.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.
