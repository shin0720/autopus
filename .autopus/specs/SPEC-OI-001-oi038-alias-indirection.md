# SPEC-OI-001 OI-038 Alias Indirection

Git Command-Line Alias Detection Before Enforce

## 1. Title

SPEC-OI-001 OI-038 Alias Indirection.

Git Command-Line Alias Detection Before Enforce.

## 2. Status

- Draft / documentation artifact only.
- no implementation in this turn.
- no tests in this turn.
- no source changes in this turn.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.
- git action: NO.

## 3. Purpose

- Fix the OI-038 alias-indirection follow-up as a separate SPEC.
- Document the `git -c alias.x=push x` bypass risk.
- Define problem scope and acceptance criteria before implementation/test work.
- Prevent direct CSV modification.

## 4. Scope

- OI-038.
- git alias / command-line indirection.
- `git -c alias.<name>=<command> <name>`.
- Keep direct git push deny behavior.
- Keep read-only git command allow behavior.
- Dangerous alias must fail closed.
- Benign alias false-positive control must be preserved where static expansion is unambiguous.

## 5. Non-goals

- no actual git execution.
- no shell execution.
- no full git config interpreter.
- no global/local/system git config lookup.
- no include handling.
- no network.
- no provider CLI.
- no hook path.
- no env=enforce.
- no operational evidence collection.
- no CSV relabeling.

## 6. Background

- Policy 4 decision result artifact completed.
- OI-020 / OI-021 / OI-034 / OI-037 make-variable/policy track is handled as a quality signal proposal.
- OI-038 remains outside the make-variable fix.
- operational-evidence.csv observed_* remains `not_collected` / `planned_only`.
- OI-038 has no direct observed evidence.
- Preflight found no confirmed command-line alias expansion logic.

## 7. OI-038 target definition

| sample_id | bucket | target_type | category | pattern | risk | make-variable coverage | git_gate coverage | desired future decision | caveat |
|---|---|---|---|---|---|---|---|---|---|
| OI-038 | follow-up | git command | alias indirection | `git -c alias.x=push x` | mutation command hidden behind alias | no | direct git deny exists; command-line alias expansion not confirmed | deny/fail-closed | observed evidence not collected |

## 8. Problem statement

- A mutation command can be hidden behind a command-line alias.
- Direct deny regex may not see the expanded mutation.
- `git -c alias.x=push x` can appear non-mutating at surface level.
- Actual git execution is forbidden.
- Full git config interpretation is out-of-scope.
- Static fail-closed handling is required before enforce.

## 9. Recommended strategy

- Primary strategy: limited alias expansion + deny-on-uncertain fallback.
- Scope-reduction fallback: deny mutation-capable alias definitions.
- Documented limitation alone is not sufficient before enforce.
- Actual git execution is prohibited.

## 10. Limited alias expansion policy

- Support only command-line `-c alias.<name>=<value>`.
- Support simple alias invocation in the same command args.
- Expand only one level.
- Recognize mutation command values:
  - `push`
  - `reset`
  - `clean`
  - `rebase`
  - `merge`
  - `add`
  - `commit`
- Recognize read-only command values:
  - `status`
  - `config --get`
  - `remote -v`
  - `log`
  - `diff`
- Do not read global/local/system git config.
- Do not execute git.
- Do not resolve shell bang alias as safe.
- Undefined or ambiguous alias becomes uncertain.

## 11. Deny-on-uncertain policy

- Dangerous alias value => deny/fail-closed.
- Alias value contains shell execution or bang alias => deny/fail-closed.
- Alias value contains mutation verb => deny/fail-closed.
- Unresolved alias invocation with mutation-capable context => deny/fail-closed.
- Clearly read-only alias may allow only if static expansion is unambiguous.
- Ambiguous parsing before enforce should fail closed.

## 12. Expected behavior table

| case_id | input pattern | desired future decision | reason | risk | SPEC coverage | regression test needed |
|---|---|---|---|---|---|---|
| OI-038 | `git -c alias.x=push x` | deny/fail-closed | mutation command hidden behind alias | bypass direct deny | yes | yes |
| direct git push | `git push` | deny remains | direct mutation command | destructive remote mutation | yes | yes |
| read-only git config --get | `git config --get user.name` | allow remains | read-only config query | overblocking risk | yes | yes |
| git remote -v | `git remote -v` | allow remains | read-only remote listing | overblocking risk | yes | yes |
| benign alias candidate | `git -c alias.l=status l` | allow if unambiguous | alias expands to read-only status | false-positive risk | yes | yes |
| dangerous alias candidate | `git -c alias.ship=push ship` | deny/fail-closed | alias expands to mutation | bypass risk | yes | yes |
| bang alias candidate | `git -c alias.x=!sh x` | deny/fail-closed | shell execution via alias | shell execution risk | yes | yes |
| ambiguous alias candidate | `git -c alias.x=<ambiguous> x` | deny/fail-closed before enforce | ambiguous parsing | unsafe allow risk | yes | yes |

## 13. Acceptance criteria

- Given git args define `alias.x=push` and invoke `x`
  - When `EvaluateGitGate` evaluates the command
  - Then decision is deny or fail-closed.
- Given git args are direct `git push`
  - When `EvaluateGitGate` evaluates the command
  - Then deny remains unchanged.
- Given git args are `git config --get user.name`
  - When `EvaluateGitGate` evaluates the command
  - Then allow remains unchanged.
- Given git args define `alias.l=status` and invoke `l`
  - When `EvaluateGitGate` evaluates the command
  - Then allow may remain if unambiguously read-only.
- Given git args define bang alias
  - When `EvaluateGitGate` evaluates the command
  - Then decision is deny/fail-closed.
- Given alias parsing is ambiguous
  - When `EvaluateGitGate` evaluates the command
  - Then decision is deny/fail-closed before enforce.

## 14. Regression test plan

- Tests are planned, not written in this turn.
- OI-038 test required.
- Direct git push regression required.
- Read-only config/remote false-positive controls required.
- Benign alias FP-control recommended.
- Dangerous alias deny regression required.
- Bang alias deny regression required.
- Existing dataset CSV must not be relabeled in this turn.

## 15. Implementation boundary

- Target file candidates:
  - `pkg/guard/git_gate.go`
  - possible new tests under `pkg/guard/`
- Non-target:
  - `make_inspector.go`
  - worker/orchestra/internal CLI
  - dataset CSV files
  - pipeline/YAML
- No source modification in this turn.

## 16. Risk and false-positive controls

- Safe alias overblocking risk.
- Alias syntax complexity.
- Shell bang alias risk.
- Quoting/tokenization ambiguity.
- False-positive control for read-only alias/status/config/remote.
- Fail-closed preferred before enforce when ambiguous.

## 17. Rollback criteria

- If read-only git commands are overblocked, narrow alias deny rules.
- If dangerous alias still passes, tighten alias expansion or deny-on-uncertain.
- If false positive rate rises, split policy.
- Revert only future implementation changes, not this SPEC.

## 18. Out-of-scope follow-ups

- Operational evidence retry.
- CSV policy update.
- Full git config interpreter.
- Hook path proof.
- N-beta enforce gate.
- UI verification.
- Autopus AI employee-team operations checklist implementation.

## 19. Enforcement gate blockers

- review_target 8 pending.
- OI-038 implementation not done.
- OI-038 tests not written.
- observed evidence not collected.
- operational-evidence.csv observed_* not_collected / planned_only.
- M3/M4 inert.
- production YAML source/decoder absent.
- UI01~UI11 verified:false.
- Autopus AI employee-team operations checklist pending.
- env=enforce gate incomplete.
- SB8 CLEARED: NO.
- release gate BLOCKED.

## 20. Completion criteria

- SPEC file created exactly once.
- no source/test/config changes.
- no existing CSV changes.
- no existing SPEC/result changes.
- no go command.
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
