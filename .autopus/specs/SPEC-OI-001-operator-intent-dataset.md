# SPEC-OI-001 — Operator-Intent Ground-Truth Dataset

| Field | Value |
|---|---|
| SPEC ID | SPEC-OI-001 |
| Title | Operator-Intent Ground-Truth Dataset |
| Status | draft |
| Track | SB8 dry-run telemetry → FP review |
| Scope | Documentation only (no source/test/YAML/telemetry change) |
| Resolves | FP_REVIEW_INSUFFICIENT (precondition for enforce, not enforce itself) |

> This document is a synthetic/redacted ground-truth **standard**, not execution
> data. Authoring it is NOT SB8 CLEARED, NOT release ready, NOT env=enforce
> activation. See §10.

---

## 1. Purpose

The SB8 command guard runs in dry-run mode and emits telemetry, but there is no
ground-truth standard against which a recorded `deny` can be judged as a
*legitimate block* versus an *over-block (false positive)*. That gap is tracked
as `FP_REVIEW_INSUFFICIENT` and it blocks any move to `env=enforce`.

- **FP_REVIEW_INSUFFICIENT resolution**: this dataset defines, per command class,
  what the operator's legitimate intent is and what the guard decision *should*
  be, so dry-run records can be scored for false-positive / false-negative rate.
- **Why a baseline is required before enforce**: enforce turns a `deny` into an
  actual block. Without a false-positive baseline, enabling enforce can block an
  operator's legitimate work with no prior measurement of how often that happens.
- **Synthetic/redacted, not execution data**: every sample is hand-authored
  synthetic or redacted text. The dataset is a *judgment standard*, not a capture
  of real operator commands, secrets, or telemetry bodies.

## 2. Non-goals

- NOT actual command execution.
- NOT provider CLI invocation.
- NOT raw telemetry / NDJSON collection or copying.
- NOT secret collection.
- NOT a production YAML source/decoder.
- NOT an `env=enforce` gate.
- NOT SB8 CLEARED.
- NOT release readiness.

## 3. Dataset schema

Each sample is one row with the following fields. Field names align to the
telemetry record (`pkg/guard/telemetry/record.go`) where a counterpart exists.

| Field | Type | Notes |
|---|---|---|
| `sample_id` | string | `OI-NNN`, globally unique within the dataset |
| `category` | enum | one of the §5 categories |
| `source_context` | enum | `synthetic` or `redacted` only (no real source) |
| `command_preview` | string | synthetic/redacted text, ≤120 chars recommended |
| `normalized_command_intent` | string | normalized meaning (e.g. "git read-only status") |
| `expected_operator_intent` | enum | `legitimate_read` / `legitimate_mutation_authorized` / `illegitimate_mutation` / `install_exec` / `destructive` / `administrative` |
| `expected_guard_decision` | enum | `allow` / `deny` |
| `expected_guard_id_or_phase` | enum | §4 mapping value (allow → `P8a`) |
| `expected_would_block_in_enforce` | bool | equals `deny` (record `would_block_in_enforce`) |
| `expected_false_positive` | bool | safe intent but `deny` ⇒ true (improvement target) |
| `expected_false_negative` | bool | dangerous intent but `allow` ⇒ true (e.g. KNOWN_FALSE_NEGATIVE) |
| `expected_t12_fail_closed` | bool | make file-sourcing fail-closed (record `t12_fail_closed`) |
| `expected_m3_m4_inert` | bool | `true` while profiles/providers are not injected (record `m3_m4_inert`) |
| `privacy_classification` | enum | `public_synthetic` / `redacted` (no `private`) |
| `secret_risk` | enum | `none` only — any other value ⇒ sample REJECTED |
| `provider_cli_risk` | enum | `none` / `string_only` (never invoked) |
| `actual_execution_allowed` | bool | always `false` |
| `rationale` | string | why this decision matches/mismatches operator intent |
| `reviewer_notes` | string | agreement, `unresolved`, `KNOWN_FALSE_NEGATIVE`, `policy_expected` |

## 4. GuardID / phase mapping

Mapping is **implementation-consistent** with `GuardIDFromPhase`
(`pkg/guard/telemetry/record.go`). Arbitrary constants are prohibited.

| Phase | guard_id |
|---|---|
| `script_inspector` / make | `M6` |
| `git_gate` | `M5` |
| `denylist` | `M2` |
| `profile` | `M3` |
| `provider_binding` | `M4` |
| `subagent` | `M7` |
| `egress` | `M8` |
| `non_structured` | `T02` |
| `allow` | `P8a` |

`P8a` is also the documented fallback for any unknown phase, so every row carries
a non-empty `expected_guard_id_or_phase`.

## 5. Category matrix

Minimum 12 categories. `execution rule` is `no-exec` everywhere; provider-looking
strings are additionally `never-exec`.

| # | Category | Min samples | Expected decision pattern | Risk | Privacy rule | Collection rule | Execution rule |
|---|---|---|---|---|---|---|---|
| 1 | safe git read-only (`status`/`log`/`diff`) | 5 | allow (`P8a`) | low | synthetic | hand-authored | no-exec |
| 2 | dangerous git mutation (`push`/`reset --hard`) | 5 | deny (`M5`) | high | synthetic | hand-authored | no-exec |
| 3 | make safe recipe (`go build`/`go test`) | 3 | allow (`P8a`) | low | synthetic | in-memory string | no-exec |
| 4 | make dangerous recipe (`git push`/`curl\|sh`) | 3 | deny (`M6`) | high | synthetic | in-memory string | no-exec |
| 5 | missing Makefile fail-closed | 3 | deny (`M6`, `t12_fail_closed`) | medium | synthetic (temp-path label only) | concept only | no-exec |
| 6 | ambiguous variable Makefile KNOWN_FALSE_NEGATIVE | 2 | allow (`P8a`, `false_negative=true`) | tracked | synthetic | in-memory string | no-exec |
| 7 | provider-looking command strings | 3 | string_only, not executed | medium | redacted | string only | never-exec |
| 8 | network/install pipe execution (`curl\|sh`) | 3 | deny (`M6`/`M2`) | high | synthetic | hand-authored | no-exec |
| 9 | file deletion/destructive shell (`rm -rf`) | 3 | deny (`M2`) | high | synthetic | hand-authored | no-exec |
| 10 | UI/workflow administrative | 2 | allow or `administrative` | low | synthetic | hand-authored | no-exec |
| 11 | false-positive candidate (safe but deny-suspect) | 4 | review target | medium | synthetic | hand-authored | no-exec |
| 12 | false-negative candidate (dangerous but allow-suspect) | 2 | review target | high | synthetic | hand-authored | no-exec |

Category minimums sum to 38, which satisfies the §6 total of 30 with margin.

## 6. Dataset size / acceptance criteria

| Criterion | Value |
|---|---|
| Minimum total samples | 30 |
| Safe allow samples | ≥ 10 |
| Dangerous deny samples | ≥ 10 |
| T12-related samples | ≥ 6 |
| False-positive candidate samples | ≥ 4 |
| False-negative candidate samples | ≥ 2 |
| Raw secret tolerance | 0 |
| Provider CLI actual call | 0 |
| Actual command execution | 0 |
| Network/API call | 0 |

- **Unresolved handling**: a sample marked `unresolved` in `reviewer_notes` is
  EXCLUDED from acceptance statistics and MUST NOT be used as enforce-gate input.
- **Reviewer agreement**: at least 2 reviewers must agree on `expected_operator_intent`
  and `expected_guard_decision`, or an equivalent reviewer-pass process. Disagreement
  ⇒ `unresolved`.
- A sample with `secret_risk != none` fails acceptance and the whole dataset is
  REJECTED until removed/redacted (zero tolerance).

## 7. Privacy / security boundary

- No raw token / password / API key / JWT.
- No real home path / user path (synthetic paths only).
- No provider account / project name.
- No actual customer / private repo name (use `example/repo`).
- No raw NDJSON body copying (telemetry raw files are never read or copied).
- Command strings are synthetic OR redacted only.
- No actual command execution.
- No provider CLI invocation.
- No network/API call.
- No git add/commit/push/PR.

## 8. Sample authoring rules

- **Synthetic-only**: every `command_preview` is invented or redacted, never lifted
  from a real session.
- **Secret-like literals**: placeholders only (e.g. `<TOKEN>`, `<API_KEY>`); never a
  real or realistic credential.
- **Provider-looking samples**: recorded as `provider_cli_risk = string_only`; the
  string is documentation, never invoked.
- **Execution rule**: always `no-exec` (`actual_execution_allowed = false`).
- **Length**: `command_preview` ≤ 120 chars recommended.
- **reviewer_notes**: must mark `unresolved`, `KNOWN_FALSE_NEGATIVE`, or
  `policy_expected` where applicable.

## 9. Review workflow

1. Author the sample (synthetic/redacted command_preview + normalized intent).
2. Reviewer judges `expected_operator_intent`.
3. Reviewer judges `expected_guard_decision` (+ `expected_guard_id_or_phase`).
4. Classify false-positive / false-negative candidate.
5. Separate `unresolved` samples (excluded from statistics).
6. Compute acceptance against §6 thresholds.
7. A separate approval is required before the dataset is handed to the enforce
   gate as input — passing §6 alone does not authorize enforce.

## 10. Relationship to SB8 / N-β

- Authoring this document is **NOT** SB8 CLEARED.
- Authoring this document is **NOT** release ready.
- Authoring this document is **NOT** env=enforce activation.
- The N-β enforce gate requires, after this SPEC: dataset population + reviewer
  agreement + a separate enforce-gate preflight. None of those happen here.
- `M3/M4 inert`, no production YAML source/decoder, and `UI01~UI11 verified:false`
  all remain open and are not addressed by this document.

## 11. Open limitations

- **KNOWN_FALSE_NEGATIVE**: `$(GIT) push` — variable expansion is not resolved by
  the static inspector, so it is currently ALLOWED. Tracked as a false-negative
  candidate (category 6 / 12), not "fixed" here.
- **symlink / oversized Makefile**: out of T12-C scope; candidate for a follow-up
  T12-D track (platform-dependent).
- **Synthetic baseline**: this dataset is a synthetic judgment standard, not an
  operational measurement of real traffic.
- **Production YAML / ruleset not injected**: M3/M4 stay inert; `expected_m3_m4_inert`
  is `true` for current-state samples.
- **UI verification incomplete**: UI01~UI11 remain `verified:false`.
