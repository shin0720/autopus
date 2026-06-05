# SPEC-OI-001 Known False Negative Must-Fix — Make Variable Push Detection Before Enforce

## 1. Title

SPEC-OI-001 Known False Negative Must-Fix — Make Variable Push Detection Before Enforce.

This SPEC defines how the make-recipe static inspector must be reinforced so that variable-indirected dangerous push commands are not silently allowed before any enforce mode is enabled.

## 2. Status

- Stage: **Draft / documentation artifact only**.
- No implementation in this turn.
- No tests in this turn.
- No source changes in this turn.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
- release ready: NO.

This document is a design artifact. It does not change behaviour by itself and is not acceptance evidence.

## 3. Scope

- OI-020 (variable-expanded git push) — must cover.
- OI-021 (nested make variable push) — must cover.
- OI-037 (variable force push) — should cover, otherwise tracked as follow-up.
- OI-034 (make help read-only target) — false-positive control (must stay allowed when clearly safe).
- make recipe **static inspection only**.
- No actual make execution.

## 4. Non-goals

- No actual make execution.
- No shell execution.
- No full Makefile interpreter.
- No network.
- No provider CLI.
- No hook path.
- No env=enforce.
- No dataset relabeling.
- No operational evidence collection.

## 5. Background

- The external-temp-harness observation retry was abandoned because offline module-graph build is structurally blocked: `gopkg.in/yaml.v3` (a direct, pre-1.17/unpruned dependency) forces `gopkg.in/check.v1` into the module graph, and `check.v1`'s `.mod` is absent from the local module cache while network lookup is disabled. The cost of working around this outweighs the value of the observation.
- `operational-evidence.csv` `observed_*` columns remain `not_collected`.
- The 8-sample verdicts are **source-based static prediction**, not observed evidence, and are **not acceptance evidence**.
- `policy.csv` marks OI-020 and OI-021 as `must_fix_before_enforce`, with `accept_as_documented_limitation_allowed = no`.

## 6. Must-fix targets

| sample_id | category | current static prediction | risk | must-fix reason | desired future decision |
|---|---|---|---|---|---|
| OI-020 | ambiguous-var-known-fn | allow@M6 (variable not resolved) | high | `$(GIT) push` evades deny logic; under enforce it would pass a real remote mutation | deny or fail-closed |
| OI-021 | ambiguous-var-known-fn | allow@M6 (nested variable not resolved) | high | `$(MAKE) -C sub push` nested indirection evades deny logic | deny or fail-closed |

## 7. Observed evidence caveat

- `operational-evidence.csv` `observed_decision` remains `not_collected` for all 8 samples.
- All predictions in this SPEC are **source-based static prediction only**.
- Static prediction is **not acceptance evidence**.
- FP_REVIEW is **not resolved**.
- The enforce gate is **not allowed**.

## 8. Problem statement

- `InspectMakeTarget` does not safely resolve make variable indirection.
- A variable-expanded git push (`$(GIT) push`) can evade the deny logic because the inspector passes the literal `$(GIT)` token to the git gate, which classifies it as a non-git executable.
- A nested make variable push (`$(MAKE) -C sub push`) can likewise evade deny logic.
- Exact reconstruction is impossible for complex Makefiles (recursive variables, `$(shell ...)`, includes, conditionals).
- Enforce therefore requires **fail-closed** handling: when a dangerous mutation cannot be statically ruled out, the recipe must be denied rather than allowed.

## 9. Recommended strategy

- **Primary strategy**: limited variable expansion + deny-on-uncertain fallback.
  - Simple, statically-resolvable single-step variable assignments are expanded and re-inspected.
  - Anything that cannot be safely resolved and cannot be proven non-dangerous is denied / fail-closed.
- **Scope-reduction fallback**: deny-on-uncertain only (drop limited expansion) if the expansion logic proves too large or risky.
- **Documented limitation alone**: **prohibited** for OI-020 and OI-021. These must not be accepted as a documented limitation; they must be fixed before enforce.

## 10. Variable expansion policy

- Allow only **simple one-step** variable assignment expansion.
- Examples:
  - `GIT=git` then `$(GIT) push` resolves to `git push`.
  - `GIT := git` then `$(GIT) push` resolves to `git push`.
- Do not execute shell.
- Do not follow include files.
- Do not recursively expand unbounded or self-referential variables.
- Undefined variables are treated as **uncertain** (route to deny-on-uncertain).

## 11. Deny-on-uncertain policy

- If a recipe contains variable indirection and a dangerous mutation cannot be ruled out, **deny or fail-closed**.
- Dangerous tokens (non-exhaustive):
  - `push`
  - `force`
  - `--force`
  - `remote`
  - `install`
  - `curl`
  - `wget`
  - `Invoke-WebRequest`
  - `Invoke-RestMethod`
  - pipe-to-shell patterns (e.g. `| sh`, `| bash`, `| iex`)
- A `make help` / echo-only read-only target **should not be overblocked** when no dangerous token is present after expansion.

## 12. Expected behavior table

| sample_id | current static prediction | desired future decision | reason | risk | SPEC coverage | regression test needed |
|---|---|---|---|---|---|---|
| OI-020 | allow@M6 | deny or fail-closed | variable-expanded git push must not pass | high | must cover | yes |
| OI-021 | allow@M6 | deny or fail-closed | nested make variable push must not pass | high | must cover | yes |
| OI-037 | allow@M6 | deny or fail-closed | variable force push | high | should cover / follow-up | yes |
| OI-034 | allow@M6 | allow | read-only help/echo, no dangerous token | med | false-positive control | yes (FP regression) |
| OI-033 | allow@M5 | allow | read-only git stash list, unaffected | low | unaffected | optional |
| OI-035 | allow@M5 | allow | read-only git config --get, unaffected | low | unaffected | optional |
| OI-036 | allow@M5 | allow | read-only git remote -v, unaffected | low | unaffected | optional |
| OI-038 | allow@M5 | deny in separate follow-up | git alias indirection, not covered by make-variable SPEC | med | not covered (separate) | separate |

## 13. Acceptance criteria

- **Given** a Makefile with `GIT=git` and a recipe `$(GIT) push`, **When** the target is inspected, **Then** the decision is deny or fail-closed.
- **Given** a Makefile with a nested make variable push (`$(MAKE) -C sub push`), **When** the target is inspected, **Then** the decision is deny or fail-closed.
- **Given** a Makefile help target with an echo-only read-only recipe, **When** the target is inspected, **Then** the decision is allow (not marked dangerous).
- **Given** the read-only git commands OI-033 / OI-035 / OI-036, **When** the git gate evaluates them, **Then** allow remains unchanged.
- **Given** alias indirection OI-038, **Then** it is tracked as a separate follow-up if not covered by the make-variable fix.

## 14. Regression test plan

- Tests are **planned, not written in this turn**.
- OI-020 and OI-021 deny/fail-closed tests are required.
- OI-034 false-positive (read-only help stays allowed) regression is required.
- OI-037 follow-up regression is recommended.
- Existing dataset CSV files **must not be relabeled in this turn** (or in the implementation turn without separate approval).

## 15. Implementation boundary

- Target file candidates (proposal only, not modified here):
  - `pkg/guard/make_inspector.go`
  - possible new tests under `pkg/guard/`
- Non-target:
  - `pkg/worker/*`
  - `pkg/orchestra/*`
  - `internal/cli/*`
  - dataset CSV files
  - pipeline / YAML
- **No source modification in this turn.**
- The inspector must remain a pure static function: no shell, no make execution, no network, no file I/O beyond the in-memory Makefile text already passed in.

## 16. Risk and false-positive controls

- False-positive risk concentrates on `make help` and other read-only targets.
- Deny-on-uncertain may overblock if its uncertainty threshold is too broad.
- Limited expansion may miss complex Makefiles (recursive / shell / include / conditional).
- Fail-closed is preferred before enforce when a dangerous mutation cannot be ruled out.
- Read-only targets should stay allowed when they are clearly safe (no dangerous token after expansion).

## 17. Rollback criteria

- If read-only help targets are overblocked, narrow the uncertain rules (tighten the dangerous-token gate, widen the safe-allow set).
- If a dangerous push still passes, tighten deny-on-uncertain.
- If the false-positive rate rises beyond an acceptable threshold, split the policy (separate read-only allowlist from the deny-on-uncertain path).
- Revert only the implementation changes in a future implementation turn — never this SPEC document.

## 18. Out-of-scope follow-ups

- OI-038 git alias indirection (`git -c alias.x=push x`).
- Hook path non-executing proof.
- Operational evidence retry.
- Dataset relabeling.
- N-β enforce gate.
- UI verification (UI01–UI11).

## 19. Enforcement gate blockers

- review_target 8 pending.
- observed evidence not collected.
- policy_decision_required = 4.
- OI-020 / OI-021 must-fix not implemented.
- OI-034 / OI-037 split not resolved.
- M3 / M4 inert.
- production YAML source / decoder absent.
- UI01–UI11 verified:false.
- env=enforce gate incomplete (double flag not satisfied).
- SB8 CLEARED: NO.
- release gate BLOCKED.

## 20. Completion criteria

- SPEC file created exactly once.
- No source / test / config changes.
- No existing CSV changes.
- No go command.
- No pure API call.
- No git add / commit / push / PR.
- FP_REVIEW resolved: NO.
- SB8 CLEARED: NO.
