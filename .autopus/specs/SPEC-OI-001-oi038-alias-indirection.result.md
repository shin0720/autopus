# SPEC-OI-001 OI-038 Alias Indirection Result

Git Command-Line Alias Detection Quality Signal

## 1. Title

SPEC-OI-001 OI-038 Alias Indirection Result.

Git Command-Line Alias Detection Quality Signal.

## 2. Status

- Documentation artifact only.
- Quality signal only.
- Operational observation evidence: NO.
- CSV mutation: NO.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.
- git action: NO.

## 3. Purpose

- Record the OI-038 alias-indirection implementation/test green result.
- Separate observed evidence from quality signal.
- Provide review_target follow-up decision material.
- Prevent direct edits to existing CSV files.
- Document blockers that prevent moving to the N-beta enforce gate.

## 4. Scope

- OI-038.
- direct git push.
- git config --get.
- git remote -v.
- benign alias.
- dangerous alias.
- bang alias.
- ambiguous alias.
- Includes the review_target 8 matrix.

## 5. Non-goals

- no FP_REVIEW resolution.
- no SB8 clear.
- no release gate clear.
- no env=enforce.
- no operational evidence collection.
- no CSV relabeling.
- no implementation changes.
- no test execution in this turn.
- no git add/commit/push/PR.

## 6. Source inputs

- `.autopus/specs/SPEC-OI-001-oi038-alias-indirection.md`.
- `pkg/guard/git_gate.go`.
- `pkg/guard/git_gate_alias_indirection_test.go`.
- Previous `go test ./pkg/guard` result: ok.
- OI-038 implementation result consolidation preflight.
- NOTE: `go test` is not re-run in this turn.

## 7. Implementation summary

- limited one-step command-line alias expansion.
- deny-on-uncertain fallback.
- static-only analysis.
- no actual git execution.
- no actual git alias execution.
- no shell execution.
- no make execution.
- no network/provider.
- no full git config interpreter.
- no global/local/system git config lookup.

## 8. Test green quality signal

| sample_or_case | expected behavior | quality signal | caveat |
|---|---|---|---|
| OI-038 `git -c alias.x=push x` | deny/fail-closed | green | quality signal only, not observed evidence |
| direct git push | deny remains | green | baseline regression signal only |
| `git config --get user.name` | allow remains | green | read-only false-positive control signal only |
| `git remote -v` | allow remains | green | read-only false-positive control signal only |
| benign alias `git -c alias.l=status l` | allow | green | benign alias false-positive control signal only |
| dangerous alias `git -c alias.ship=push ship` | deny/fail-closed | green | quality signal only, not observed evidence |
| bang alias `git -c alias.x=!sh x` | deny/fail-closed | green | quality signal only, not observed evidence |
| ambiguous alias `git -c alias.x=$(unknown) x` | deny/fail-closed | green | quality signal only, not observed evidence |
| `go test ./pkg/guard` | ok | green (previous implementation turn) | not re-run this turn |

## 9. Observed evidence caveat

- operational-evidence.csv `observed_decision` = not_collected.
- operational-evidence.csv `observed_guard_id_or_phase` = not_collected.
- operational-evidence.csv `observed_status` = planned_only.
- Package test green is not operational observation evidence.
- Static/test signal is not acceptance evidence.
- FP_REVIEW resolved: NO.
- enforce gate allowed: NO.
- SB8 CLEARED: NO.

## 10. Review target status matrix

| sample_id | bucket | quality_signal_status | observed_evidence_status | current_classification | remaining_blocker | next_action |
|---|---|---|---|---|---|---|
| OI-020 | policy-primary | quality signal complete; make-variable must-fix green | not_collected | quality signal complete, policy proposal documented | observed evidence not collected | final review readiness preflight |
| OI-021 | policy-primary | quality signal complete; make-variable must-fix green | not_collected | quality signal complete, policy proposal documented | observed evidence not collected | final review readiness preflight |
| OI-033 | operational-only | unaffected by OI-038 result | not_collected | operational-only; operational observation deferred | observation path/cost | keep deferred |
| OI-034 | policy+operational | quality signal available; split documented | not_collected | split documented, operational still pending | observed evidence not collected | final review readiness preflight |
| OI-035 | operational-only | unaffected by OI-038 result | not_collected | operational-only; operational observation deferred | observation path/cost | keep deferred |
| OI-036 | operational-only | unaffected by OI-038 result | not_collected | operational-only; operational observation deferred | observation path/cost | keep deferred |
| OI-037 | policy+operational | quality signal available; split documented | not_collected | split documented, operational still pending | observed evidence not collected | final review readiness preflight |
| OI-038 | follow-up | implementation/test green quality signal complete | not_collected | implementation quality signal complete, review still pending | observed evidence not collected | final review readiness preflight |

## 11. Policy / review status

- review_target 8 pending.
- Policy 4 decision result artifact exists.
- OI-038 result artifact is quality signal only.
- no policy.csv modification.
- no resolution.csv modification.
- no review.csv modification.
- no samples.csv modification.
- no operational-evidence.csv modification.

## 12. Remaining blockers

- observed evidence not collected.
- operational observation deferred.
- review_target 8 pending.
- M3/M4 inert.
- production YAML source/decoder absent.
- UI01~UI11 verified:false.
- Autopus AI employee-team operations checklist pending.
- env=enforce gate incomplete.
- SB8 CLEARED: NO.
- release gate BLOCKED.

## 13. Enforcement gate blockers

- N-beta enforce gate not allowed.
- no env=enforce.
- no SB8 clear.
- no release ready.
- no FP_REVIEW resolved.
- no public/GitHub action.
- no git add/commit/push/PR.

## 14. CSV mutation boundary

- no policy.csv modification.
- no resolution.csv modification.
- no review.csv modification.
- no samples.csv modification.
- no operational-evidence.csv modification.
- Any future CSV update must be a separate preflight/patch turn.
- Do not relabel observed evidence from quality signal.

## 15. Next recommended tracks

| candidate | summary | recommendation |
|---|---|---|
| A. final review / FP_REVIEW readiness preflight | Read-only check whether the accumulated quality-signal artifacts are sufficient to plan review resolution work. | Recommended |
| B. CSV update preflight | Prepare CSV patching separately, but only after preserving evidence caveats. | Defer |
| C. operational observation retry | Could target observed evidence, but the observation path remains deferred. | Defer |
| D. N-beta enforce gate | Not allowed while blockers remain. | Forbidden |

Recommended next track: A. final review / FP_REVIEW readiness preflight.

Reason: after both make-variable and OI-038 quality-signal artifacts exist, the next appropriate step is a read-only readiness preflight to determine whether FP_REVIEW can be planned for resolution. FP_REVIEW resolved handling remains prohibited here, and the N-beta enforce gate remains blocked.

## 16. Completion criteria

- result artifact created exactly once.
- no source/test/config changes.
- no existing CSV changes.
- no existing SPEC/result changes.
- no go test/go command.
- no pure API/hook/probe.
- no actual git execution.
- no git add/commit/push/PR.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.

## Autopus AI Employee-Team Operations Checklist

This checklist is for product operations quality. It does not replace the fixed 9-item cumulative checklist. It is not implemented in this turn. This turn does not change UI verified status to true.

| item | current status |
|---|---|
| 1. Agent Profile | pending / verified:false |
| 2. Memory / Wiki | pending / verified:false |
| 3. Skill Lifecycle | pending / verified:false |
| 4. Handoff Protocol | pending / verified:false |
| 5. Bug Tracking | pending / verified:false |
| 6. UI Dashboard | pending / verified:false |
| 7. Model Quota / Fallback Routing | pending / verified:false |
| 8. Human Approval / Escalation | pending / verified:false |
| 9. Team Retrospective / Self-Improvement | pending / verified:false |
