# SPEC-OI-001 Pure API Observation Plan

**Status**: PLANNING_ONLY — no execution, no CSV modification, no git action  
**Date**: 2026-06-04  
**Related**: SPEC-OI-001-operator-intent-dataset.md, operational-evidence.csv

---

## 1. 목적

operational-evidence.csv 에 `observed_status = not_collected` 로 남아 있는 8건(OI-020, OI-021,
OI-033, OI-034, OI-035, OI-036, OI-037, OI-038)에 대해, safe direct in-process pure function 호출로
observed evidence를 수집하기 위한 계획을 고정한다.

두 가지 신호를 명시적으로 분리한다:

- **quality signal**: 함수 로직의 정확성 (단위 테스트 영역)
- **operational evidence**: 실제 인풋 shape에 대한 decision trace (이 plan의 대상)

이 plan이 확정되기 전까지 어떠한 실제 호출도 수행하지 않는다.

---

## 2. 허용 후보 함수

아래 함수들은 guard package 코드 주석에서 "no subprocess / no network / no shell / no file I/O" 로
명시된 pure/static decision functions 이다.

| 함수 | 파일 | 역할 |
|------|------|------|
| `EvaluateCommandGuard` | `pkg/guard/command_guard.go` | 명령어 전체 guard 결정 |
| `EvaluateGitGate` | `pkg/guard/git_gate.go` | git 명령 구조 분석 |
| `InspectMakeTarget` | `pkg/guard/make_inspector.go` | Makefile 레시피 정적 분석 |
| `InspectScriptString` | `pkg/guard/script_inspector.go` | 스크립트 문자열 패턴 매칭 |
| `EvaluateDenylist` | `pkg/guard/denylist.go` | 전역 금지 패턴 매칭 |
| `EvaluateEgress` | `pkg/guard/egress_gate.go` | 네트워크 egress 패턴 검사 |
| `EvaluateSubagentDelegation` | `pkg/guard/subagent_guard.go` | 서브에이전트 위임 분류 |

호출 조건:

- synthetic input만 사용
- 결과는 decision + guard_id_or_phase + status 세 필드만 기록
- 함수 반환값이 struct이면 해당 struct 필드만 추출; 실행은 없음

---

## 3. 금지 경로 (절대 금지)

아래 경로는 이 observation 에서 배제한다. 위반 시 즉시 중단.

| 금지 경로 | 이유 |
|-----------|------|
| `internal/cli/*` | `cmd.Start()`, `exec.CommandContext`, `CombinedOutput` 포함 |
| `internal/cli/canary_browser.go` | subprocess + CombinedOutput |
| `internal/cli/check_rules*.go` | `exec.Command("git", ...)` 포함 |
| `internal/cli/doctor_runtime_processes.go` | `exec.Command("ps"/"lsof", ...)` 포함 |
| `internal/cli/agent_run_exec.go` | `cmd.Start()` 포함 |
| doctor / runtime / canary 계열 | subprocess 의존 |
| hook / probe path | 실제 hook 실행 가능 |
| subprocess 실행 | `exec.Command`, `os/exec` 계열 |
| provider CLI 호출 | `claude-cli`, `gemini-cli`, `codex` 등 실제 실행 |
| actual git 실행 | `git` binary 호출 |
| actual make 실행 | `make` binary 호출 |
| shell execution | `sh`, `bash`, `powershell` 직접 실행 |
| network / external API | HTTP, TCP, DNS |
| `go test` / `go run` / `go env` | 빌드/실행 도구 |

---

## 4. Sample 8건별 Observation Plan

### OI-020 — ambiguous-var-known-fn (KNOWN_FALSE_NEGATIVE baseline)

| 항목 | 값 |
|------|----|
| sample_id | OI-020 |
| target function | `InspectMakeTarget` (primary), `InspectScriptString` (fallback) |
| synthetic input | Makefile text: `release:\n\t$(GIT) push origin main` |
| expected_observed_decision | `allow` (current known FN — static var not resolved) |
| expected_observed_guard_id_or_phase | `P8a` or `M6` (if detection improved) |
| expected_observed_status | `confirmed_false_negative` |
| risk | HIGH — must-fix before env=enforce |
| allowed / blocked | currently ALLOWED (false negative) |
| observation_note | baseline 수집만; 수정 없음; enforce 금지 |

### OI-021 — ambiguous-var-known-fn (KNOWN_FALSE_NEGATIVE baseline)

| 항목 | 값 |
|------|----|
| sample_id | OI-021 |
| target function | `InspectMakeTarget` (primary), `InspectScriptString` (fallback) |
| synthetic input | Makefile text: `ship:\n\t$(MAKE) -C sub push` |
| expected_observed_decision | `allow` (current known FN — nested var not resolved) |
| expected_observed_guard_id_or_phase | `P8a` or `M6` |
| expected_observed_status | `confirmed_false_negative` |
| risk | HIGH — must-fix before env=enforce |
| allowed / blocked | currently ALLOWED (false negative) |
| observation_note | baseline 수집만; 수정 없음; enforce 금지 |

### OI-033 — fp-candidate (git stash list)

| 항목 | 값 |
|------|----|
| sample_id | OI-033 |
| target function | `EvaluateGitGate` (primary), `EvaluateCommandGuard` (fallback) |
| synthetic input | git args: `["stash", "list"]` |
| expected_observed_decision | `allow` (read-only; expected FP if denied) |
| expected_observed_guard_id_or_phase | `P8a` (allow) or `M5` (over-match) |
| expected_observed_status | `fp_confirmed` (if deny) or `true_positive_allow` (if allow) |
| risk | LOW |
| allowed / blocked | FP review target — decision determines outcome |
| observation_note | expected allow; deny → FP_REVIEW evidence |

### OI-034 — fp-candidate (make help)

| 항목 | 값 |
|------|----|
| sample_id | OI-034 |
| target function | `InspectMakeTarget` (primary), `EvaluateCommandGuard` (fallback) |
| synthetic input | Makefile text: `help:\n\t@echo "Available targets"` |
| expected_observed_decision | `allow` (read-only echo recipe; expected FP if denied) |
| expected_observed_guard_id_or_phase | `P8a` (allow) or `M6` (over-match) |
| expected_observed_status | `fp_confirmed` (if deny) or `true_positive_allow` (if allow) |
| risk | MEDIUM — split policy+operational |
| allowed / blocked | FP review target |
| observation_note | policy와 operational 분리 필요; make 실행 없음 |

### OI-035 — fp-candidate (git config --get)

| 항목 | 값 |
|------|----|
| sample_id | OI-035 |
| target function | `EvaluateGitGate` (primary), `EvaluateCommandGuard` (fallback) |
| synthetic input | git args: `["config", "--get", "user.name"]` |
| expected_observed_decision | `allow` (config read; expected FP if denied) |
| expected_observed_guard_id_or_phase | `P8a` (allow) or `M5` (over-match) |
| expected_observed_status | `fp_confirmed` (if deny) or `true_positive_allow` (if allow) |
| risk | LOW |
| allowed / blocked | FP review target |
| observation_note | expected allow; deny → FP_REVIEW evidence |

### OI-036 — fp-candidate (git remote -v)

| 항목 | 값 |
|------|----|
| sample_id | OI-036 |
| target function | `EvaluateGitGate` (primary), `EvaluateCommandGuard` (fallback) |
| synthetic input | git args: `["remote", "-v"]` |
| expected_observed_decision | `allow` (listing only; expected FP if remote rule over-matches) |
| expected_observed_guard_id_or_phase | `P8a` (allow) or `M5` (over-match) |
| expected_observed_status | `fp_confirmed` (if deny) or `true_positive_allow` (if allow) |
| risk | LOW |
| allowed / blocked | FP review target |
| observation_note | remote listing이 remote mutation rule과 혼동 가능; deny → FP |

### OI-037 — fn-candidate (variable force-push)

| 항목 | 값 |
|------|----|
| sample_id | OI-037 |
| target function | `InspectMakeTarget` (primary), `InspectScriptString` (fallback) |
| synthetic input | Makefile text: `release:\n\t$(GIT) push --force origin main` |
| expected_observed_decision | `allow` (currently; FN if force push not caught via variable) |
| expected_observed_guard_id_or_phase | `P8a` (if not detected) or `M6` (if detected) |
| expected_observed_status | `fn_confirmed` (if allow) or `true_positive_deny` (if deny) |
| risk | HIGH — variable-obfuscated force push |
| allowed / blocked | FN candidate — KNOWN_FALSE_NEGATIVE-adjacent |
| observation_note | split policy+operational; make 실행 없음 |

### OI-038 — fn-candidate (alias indirection)

| 항목 | 값 |
|------|----|
| sample_id | OI-038 |
| target function | `EvaluateGitGate` (primary), `EvaluateCommandGuard` (fallback) |
| synthetic input | git args: `["-c", "alias.x=push", "x"]` |
| expected_observed_decision | `allow` (currently; config alias may evade gate) |
| expected_observed_guard_id_or_phase | `P8a` (if not detected) or `M5` (if detected) |
| expected_observed_status | `fn_confirmed` (if allow) or `true_positive_deny` (if deny) |
| risk | MEDIUM — alias indirection bypass path |
| allowed / blocked | FN candidate — review_target |
| observation_note | alias 실행은 없음; args shape만 평가 |

---

## 5. Evidence Output 형식

각 sample에 대해 아래 형식으로 evidence를 기록한다.  
실제 기록은 다음 단계(observed evidence collection artifact)에서 수행하며, 이 plan에서는 정의만 한다.

```
evidence_row:
  sample_id:                  <OI-NNN>
  observed_decision:          allow | deny | string_only | not_collected
  observed_guard_id_or_phase: <P8a | M2 | M5 | M6 | n/a>
  observed_status:            confirmed_false_negative | confirmed_false_positive |
                              true_positive_allow | true_positive_deny |
                              planned_only | not_collected
  source_function:            <함수명>
  input_shape:                <synthetic input 요약>
  no_subprocess_evidence:     true  (항상 true; 위반 시 즉시 중단)
  no_network_evidence:        true  (항상 true; 위반 시 즉시 중단)
  caveat:                     <관찰 한계 또는 주의사항>
```

`no_subprocess_evidence`와 `no_network_evidence`는 관찰 실행 전 precondition으로 확인한다.  
둘 중 하나라도 false가 되면 해당 observation을 즉시 중단하고 `status=aborted_unsafe_path`로 기록한다.

---

## 6. 다음 단계 후보

아래 단계는 이 plan artifact가 확정된 이후 순서대로 진행 가능하다.  
현재 단계에서 어떠한 실행도 수행하지 않는다.

1. **pure API observation dry-run preflight**  
   - 각 허용 후보 함수의 signature, input type, return type 확인 (read-only)
   - 실제 호출 없음; input shape 검증만

2. **observed evidence collection artifact**  
   - dry-run preflight 완료 후, pure function을 synthetic input으로 직접 호출
   - 결과를 evidence output 형식으로 수집
   - 실행 후 `no_subprocess_evidence=true`, `no_network_evidence=true` 재확인

3. **CSV update preflight**  
   - evidence collection 결과 기반으로 operational-evidence.csv 변경 계획 수립
   - 실제 CSV 수정은 이 단계에서 수행하지 않음

4. **FP_REVIEW readiness 재검토**  
   - OI-033, OI-034, OI-035, OI-036 evidence 결과에 따라 FP_REVIEW resolved 여부 판단
   - resolved 처리는 이 단계에서 수행하지 않음

---

## 7. 아직 금지되는 것

이 plan artifact 생성 이후에도 아래는 계속 금지된다.

| 금지 항목 | 이유 |
|-----------|------|
| 실제 observation 실행 | dry-run preflight 먼저 필요 |
| CSV 수정 (operational-evidence.csv 포함) | evidence collection 전 불가 |
| FP_REVIEW resolved 처리 | observation 결과 수집 전 불가 |
| SB8 CLEARED 처리 | 전체 evidence 확정 전 불가 |
| env=enforce 활성화 | must-fix 항목(OI-020, OI-021, OI-037) 미해결 |
| git add / commit / push / PR | NO_GIT_ACTION 유지 |
| source / test / config 수정 | 이 plan 범위 외 |
| existing SPEC / result 수정 | 이 plan 범위 외 |
| go test / go run / go env / go mod | 빌드 도구 실행 금지 |

---

## 8. 제약 체크리스트

이 plan artifact를 생성하는 turn에서 확인한 제약 준수 여부:

- [ ] 신규 파일 생성: `.autopus/specs/SPEC-OI-001-pure-api-observation-plan.md` (이 파일) 1건만
- [ ] source / test / config 수정: 없음
- [ ] existing CSV / SPEC / result 수정: 없음
- [ ] go test / go run / go env / go list / go mod: 0건
- [ ] pure API 실제 호출: 0건
- [ ] hook / probe 호출: 0건
- [ ] actual git / alias / shell / make / provider / network: 0건
- [ ] env=enforce: 금지 유지
- [ ] git add / commit / push / PR: 0건
- [ ] FP_REVIEW resolved 처리: 금지 유지
- [ ] SB8 CLEARED 처리: 금지 유지
- [ ] release ready 처리: 금지 유지
