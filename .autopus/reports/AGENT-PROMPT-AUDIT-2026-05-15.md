# Agent Prompt Audit Report
**감사 날짜**: 2026-05-15  
**감사 대상**: `content/agents/` (Source-of-Truth) 16개 에이전트  
**감사 목적**: 사용자 후속 프롬프트 없이 99% 완성도까지 자동 진행 가능한 구조인지 진단

---

## 1. 전체 결론

**현재 자동화 완성도: 약 71/100 — 99% 완성도 도달까지 상당한 구조적 보완 필요**

- 16개 에이전트 중 **3개(architect, debugger, devops)** 가 입력 계약·출력 계약·종료 상태 중 하나 이상이 완전히 누락된 치명 결함 상태
- **4개(planner, explorer, ux-validator, security-auditor)** 가 중대 결함으로 단독 실행 시 막힐 위험
- planner는 `content/agents/` SoT에 G1-G8 품질 게이트가 없고 runtime `.claude/agents/autopus/planner.md`에만 존재 — source-of-truth 불일치
- 네이밍서비스 프로젝트 보호 가드가 **모든 에이전트에 누락**

---

## 2. 평균 점수

| 항목 | 값 |
|------|-----|
| 총 에이전트 수 | 16 |
| 평균 점수 | **71.4 / 100** |
| 80점 이상 (양호) | 6개 |
| 60~79점 (주의) | 7개 |
| 60점 미만 (위험) | 3개 |

---

## 3. 99% 완성도까지의 현재 거리

99% 완성도의 정의: 파이프라인 어느 단계에서도 사용자 개입 없이 DONE/PARTIAL/BLOCKED 중 하나로 명확히 종료되는 구조.

현재 거리:

| 갭 유형 | 현황 |
|---------|------|
| 종료 상태 미정의 | 5개 에이전트 (architect, debugger, devops, security-auditor, ux-validator) |
| 입력 계약 없음 | 6개 에이전트 (architect, explorer, debugger, security-auditor, devops, reviewer) |
| 출력 계약 없음 | 4개 에이전트 (architect, debugger, devops, explorer) |
| 자기 복구 루프 없음 | 11개 에이전트 (spec-writer, tester 제외 대다수) |
| SoT vs runtime 불일치 | 1건 (planner G1-G8) |
| 네이밍서비스 가드 없음 | 16개 에이전트 전부 |

핵심 병목: **architect**가 없으면 Phase 1.5(구현 전 설계 검토)가 맹목적으로 진행되고, **debugger**가 없으면 RALF 루프에서 실패 수정이 구조 없이 진행된다.

---

## 4. 에이전트별 점수표

| # | 에이전트 | 점수 | 등급 | 줄 수 | 주요 결함 |
|---|---------|------|------|-------|-----------|
| 1 | spec-writer | **88** | ✅ 양호 | 258 | 핸드오프 티켓 미정의 |
| 2 | validator | **88** | ✅ 양호 | 277 | 입력 계약 없음 |
| 3 | tester | **87** | ✅ 양호 | 274 | BLOCKED 명시 없음 |
| 4 | frontend-specialist | **85** | ✅ 양호 | 234 | existing-first 스캔 미명시 |
| 5 | executor | **82** | ✅ 양호 | 202 | 재작업 정리 체크리스트 없음 |
| 6 | annotator | **79** | ✅ 양호 | 153 | 핸드오프 형식 최소화 |
| 7 | perf-engineer | **77** | ⚠️ 주의 | 166 | 자기 복구 루프 없음 |
| 8 | deep-worker | **75** | ⚠️ 주의 | 153 | 핸드오프 티켓 없음 |
| 9 | reviewer | **75** | ⚠️ 주의 | 228 | BLOCKED 없음, 입력 계약 없음 |
| 10 | planner | **70** | ⚠️ 주의 | 233 | G1-G8 SoT 누락, 핸드오프 없음 |
| 11 | explorer | **65** | ⚠️ 주의 | 100 | 출력 계약·종료 상태 없음 |
| 12 | ux-validator | **62** | ⚠️ 주의 | 111 | DONE/PARTIAL/BLOCKED 없음 |
| 13 | security-auditor | **58** | ❌ 위험 | 94 | 종료 상태·입력 계약 없음 |
| 14 | debugger | **55** | ❌ 위험 | 96 | 출력 계약·종료 상태 없음 |
| 15 | devops | **52** | ❌ 위험 | 109 | 입력·출력·종료 상태 없음 |
| 16 | architect | **45** | ❌ 위험 | 86 | 사실상 stub 수준 |

---

## 5. 에이전트별 누락 항목

범례: ✅ 있음 | ⚠️ 부분 | ❌ 없음

| 에이전트 | 입력계약 | 출력계약 | 품질게이트 | 핸드오프 | DONE/PARTIAL/BLOCKED | 반사실방지 | 기존우선스캔 | 자기복구루프 | 스택감지 | 네이밍가드 |
|---------|---------|---------|----------|---------|---------------------|----------|------------|------------|---------|---------|
| planner | ⚠️ | ⚠️ | ❌ (SoT 누락) | ❌ | ⚠️ | ⚠️ | ❌ (SoT 누락) | ❌ (SoT 누락) | ✅ | ❌ |
| spec-writer | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ❌ |
| architect | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ⚠️ | ❌ |
| explorer | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ⚠️ | ❌ |
| executor | ✅ | ✅ | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ⚠️ | ✅ | ❌ |
| deep-worker | ✅ | ✅ | ✅ | ❌ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ❌ |
| frontend-specialist | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ❌ |
| tester | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ❌ |
| reviewer | ❌ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ✅ | ❌ |
| validator | ❌ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ❌ | ✅ | ❌ |
| ux-validator | ✅ | ✅ | ✅ | ⚠️ | ❌ | ⚠️ | N/A | ❌ | N/A | ❌ |
| debugger | ❌ | ❌ | ✅ | ⚠️ | ❌ | ⚠️ | ✅ | ⚠️ | ✅ | ❌ |
| security-auditor | ❌ | ✅ | ✅ | ⚠️ | ❌ | ⚠️ | N/A | ❌ | ✅ | ❌ |
| devops | ❌ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| perf-engineer | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| annotator | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |

### 5a. 에이전트별 세부 누락 항목 설명

**planner** (70/100)
- `content/agents/planner.md` SoT에 G1-G8 품질 게이트 없음 (runtime `.claude/agents/autopus/planner.md`에만 있음)
- 핸드오프 티켓 형식 미정의 — 다음 단계(spec-writer)에 무엇을 넘길지 불명확
- Anti-speculation 절 없음 — 인터뷰 없이 추측으로 PRD 작성 금지 명시 필요
- 기존 프로젝트 우선 스캔 룰 SoT에 없음

**spec-writer** (88/100)
- 핸드오프 티켓: executor에게 넘길 때의 명시적 티켓 형식 없음
- BLOCKED 조건: 입력이 너무 부족할 때의 BLOCKED 판정 기준 불명확

**architect** (45/100) — CRITICAL
- 입력 계약 완전 부재 (어떤 정보를 받아야 하는지 정의 없음)
- 출력 계약 완전 부재 (ADR 형식, 결정 목록 등 없음)
- DONE/PARTIAL/BLOCKED 상태 없음
- 품질 게이트 없음 (자기 검증 기준 없음)
- 핸드오프 티켓 없음
- Anti-speculation 없음 — 추측 기반 설계 결정 남발 위험
- 기존 프로젝트 우선 스캔 없음 — 새 아키텍처 설계 전 기존 구조 파악 의무 없음
- 86줄짜리 stub 수준 파일

**explorer** (65/100)
- 출력 계약 없음 — 자유 형식 요약만, 구조화된 출력 없음
- DONE/PARTIAL/BLOCKED 없음
- 입력 계약 없음
- 핸드오프 티켓 없음

**executor** (82/100)
- 재작업 후 정리 체크리스트 없음 (임시 파일, 스텁 제거 확인)
- Anti-speculation: 기존 코드 먼저 읽는 명시적 스캔 의무 없음
- 핸드오프 티켓: 다음 Phase(tester/validator)에 넘길 내용 구조화 없음

**deep-worker** (75/100)
- 핸드오프 티켓 없음 — 장시간 작업 완료 후 넘길 결과 형식 없음
- Anti-speculation: 장시간 독립 실행 중 추측 결정 방지 규칙 없음

**frontend-specialist** (85/100)
- Existing-first scan: 새 컴포넌트 만들기 전 기존 UI 컴포넌트 먼저 확인 의무 미명시

**tester** (87/100)
- BLOCKED 명시: 구현 코드 자체가 없을 때 BLOCKED 판정 기준 없음
- 핸드오프: validator에게 전달할 핸드오프 티켓 형식 없음

**reviewer** (75/100)
- 입력 계약 없음 — git diff + SPEC ID 등 필수 입력 목록 불명확
- BLOCKED 없음 — git 히스토리 접근 불가 등의 상황 처리 없음
- Lore commit 포맷 검사가 TRUST 5 Trackable 항목에 명시적으로 없음

**validator** (88/100)
- 입력 계약 없음 — 어떤 SPEC ID / 변경 파일 목록을 받아야 하는지 미정의
- BLOCKED 없음 — 빌드 자체가 불가한 상황에서의 종료 기준 없음

**ux-validator** (62/100)
- DONE/PARTIAL/BLOCKED 완전 부재 — 스크린샷이 없을 때 어떻게 종료?
- 자기 복구 루프 없음
- 핸드오프 형식 최소화 ("executor에게 위임" 수준)
- 111줄 — 내용이 다소 부족

**debugger** (55/100) — CRITICAL
- 출력 계약 완전 부재 — 커밋 메시지만 있고 구조화된 결과 없음
- DONE/PARTIAL/BLOCKED 완전 부재
- 입력 계약 없음
- A3 Result Format 선언은 있으나 실제 구현 없음
- 96줄 — stub 수준

**security-auditor** (58/100)
- DONE/PARTIAL/BLOCKED 없음 — 무한 감사 루프 위험
- 입력 계약 없음 — 무엇을 감사 범위로 받는지 미정의
- 자기 복구 루프 없음
- 병렬 실행 시 reviewer와의 결과 통합 로직이 reviewer.md에만 있고 여기에 없음
- 94줄 — 내용 부족

**devops** (52/100) — CRITICAL
- 입력 계약 완전 부재
- DONE/PARTIAL/BLOCKED 완전 부재
- 핸드오프 티켓 없음
- Anti-speculation 없음
- 기존 CI 파이프라인 먼저 확인 의무 없음
- 협업 항목이 "협의", "검토" 수준 — 구체적 메시지 형식 없음

**perf-engineer** (77/100)
- 자기 복구 루프 없음 — 벤치마크 실패 시 재시도 전략 없음
- BLOCKED 시 에스컬레이션 대상 미정의

**annotator** (79/100)
- 핸드오프 티켓: "다음: next phase or validation" 수준으로 최소화

---

## 6. 플랫폼별 동기화 상태

Source-of-Truth: `content/agents/`  
배포 경로: `.claude/agents/autopus/`, `templates/codex/agents/`, `templates/gemini/agents/`

| 에이전트 | SoT | Claude runtime | Codex | Gemini |
|---------|-----|----------------|-------|--------|
| planner | G1-G8 없음 | G1-G8 있음 | 미확인 | 미확인 |
| 나머지 15개 | 위 표 기준 | 미확인 | 미확인 | 미확인 |

> 주의: Codex/Gemini 배포 파일은 이번 감사 범위에서 직접 확인되지 않았습니다. `templates/` 경로의 에이전트 파일 동기화 상태 별도 확인 필요.

---

## 7. Source-of-Truth / Runtime 불일치 목록

| 항목 | SoT 파일 | Runtime 파일 | 불일치 내용 |
|------|---------|------------|-----------|
| planner G1-G8 | `content/agents/planner.md` — 없음 | `.claude/agents/autopus/planner.md` — 있음 | 품질 게이트 7단계, 기존 프로젝트 우선 스캔, 자기 복구 루프가 SoT에 반영되지 않음 |

**영향**: `/auto sync`로 배포 시 SoT 기반으로 덮어쓸 경우 G1-G8이 사라질 위험.

---

## 8. 치명 결함 목록

### CRITICAL-01: architect — 사실상 stub

- 86줄, 입력/출력/품질게이트/핸드오프/종료상태 모두 없음
- 파이프라인에서 architect가 호출되면 설계 결정 근거 없이 임의 출력 반환
- **패치 없이 사용 금지** 수준

### CRITICAL-02: debugger — 출력/종료 계약 없음

- RALF 루프에서 debugger가 수정을 완료해도 DONE/BLOCKED 없이 종료
- 출력 형식 없어 파이프라인 다음 단계가 결과를 해석할 수 없음
- 96줄 stub 수준

### CRITICAL-03: devops — 모든 계약 누락

- 무엇을 받아 무엇을 만들지, 언제 끝나는지 정의 없음
- CI/CD 설정 작업 후 결과 검증 루프도 없음

### HIGH-01: planner SoT 불일치

- G1-G8이 runtime에만 있고 SoT에 없음
- `/auto sync`로 배포 갱신 시 품질 게이트 손실 위험

### HIGH-02: 네이밍서비스 가드 전무

- 16개 에이전트 모두 "네이밍서비스 프로젝트 코드 수정 금지" 가드 없음
- 특히 executor, deep-worker가 광범위 파일 수정 권한 보유

### MEDIUM-01: ux-validator DONE/PARTIAL/BLOCKED 없음

- 스크린샷 없이 호출될 경우 어떻게 종료할지 정의 없음

### MEDIUM-02: security-auditor 종료 상태 없음

- 대규모 코드베이스 감사 시 무한 실행 위험

---

## 9. 우선 수정 순서

| 순위 | 대상 | 이유 | 예상 규모 |
|------|------|------|-----------|
| 1 | `content/agents/architect.md` | CRITICAL — 파이프라인 설계 단계 전체 망가짐 | 150~200줄 추가 |
| 2 | `content/agents/planner.md` | SoT에 G1-G8 없음 — sync 시 품질게이트 손실 | 40~60줄 추가 |
| 3 | `content/agents/debugger.md` | CRITICAL — RALF 루프 출력 계약 없음 | 80~120줄 추가 |
| 4 | `content/agents/devops.md` | CRITICAL — 모든 계약 누락 | 80~100줄 추가 |
| 5 | `content/agents/security-auditor.md` | 종료 상태·입력 계약 없음 | 40~60줄 추가 |
| 6 | `content/agents/ux-validator.md` | DONE/PARTIAL/BLOCKED 없음 | 20~30줄 추가 |
| 7 | `content/agents/explorer.md` | 출력 계약·종료 상태 없음 | 30~40줄 추가 |
| 8 | 모든 에이전트 공통 | 네이밍서비스 가드 추가 | 에이전트당 3~5줄 |

---

## 10. 수정 대상 파일 목록

> 아래는 수정이 필요한 파일 목록입니다. 수정 전 사용자 승인 필요.

```
content/agents/architect.md          — 대폭 확장 필요 (CRITICAL)
content/agents/planner.md            — G1-G8 추가, 핸드오프 티켓 추가
content/agents/debugger.md           — 출력 계약, DONE/PARTIAL/BLOCKED 추가
content/agents/devops.md             — 입력·출력·종료 상태 추가
content/agents/security-auditor.md  — DONE/PARTIAL/BLOCKED, 입력 계약 추가
content/agents/ux-validator.md       — DONE/PARTIAL/BLOCKED 추가
content/agents/explorer.md           — 출력 계약, 종료 상태 추가
content/agents/reviewer.md           — 입력 계약, BLOCKED 상태 추가
content/agents/validator.md          — 입력 계약 추가
content/agents/tester.md             — BLOCKED 명시, 핸드오프 티켓 추가
content/agents/deep-worker.md        — 핸드오프 티켓 추가
content/agents/executor.md           — 재작업 정리 체크리스트, 네이밍서비스 가드
content/agents/perf-engineer.md      — 자기 복구 루프 추가
content/agents/frontend-specialist.md — existing-first 스캔 명시
content/agents/spec-writer.md        — 핸드오프 티켓 명시
content/agents/annotator.md          — 핸드오프 형식 보강
```

---

## 11. 수정하면 안 되는 파일 목록

```
pkg/lore/writer.go                   — pkg WIP 수정 금지
pkg/lore/writer_test.go              — pkg WIP 수정 금지
pkg/adapter/codex/codex_plugin_manifest.go — pkg WIP 수정 금지
internal/cli/check_rules.go          — internal/cli 수정 금지
internal/cli/check_rules_lore_test.go — internal/cli 수정 금지
```

> 위 5개 파일의 이메일 변경은 SPEC-SIGNOFF-001로 별도 추적 중입니다.

---

## 12. 다음 단계 패치 계획

### Phase A — 치명 결함 즉시 패치 (CRITICAL 3건)

1. **architect.md 재작성**: 입력 계약(받는 것: SPEC ID + PRD + 탐색 보고서), 출력 계약(ADR 형식), 품질 게이트(설계 일관성 3-check), DONE/PARTIAL/BLOCKED, 핸드오프 티켓 추가
2. **debugger.md 확장**: 출력 계약(DONE/PARTIAL/BLOCKED + A3 result format), 입력 계약 추가
3. **devops.md 확장**: 입력 계약, 출력 계약(Completion Report), DONE/PARTIAL/BLOCKED 추가

### Phase B — SoT 불일치 해소

4. **planner.md G1-G8 동기화**: `.claude/agents/autopus/planner.md`의 G1-G8을 `content/agents/planner.md`에 이식

### Phase C — 중대 결함 보완

5. security-auditor.md DONE/PARTIAL/BLOCKED + 입력 계약 추가
6. ux-validator.md DONE/PARTIAL/BLOCKED 추가
7. explorer.md 출력 계약 + 종료 상태 추가

### Phase D — 공통 가드 추가

8. 모든 에이전트에 네이밍서비스 보호 가드 공통 절 추가

### 배포 후 검증

각 Phase 완료 후 `templates/` 및 `.claude/agents/autopus/` 동기화 상태 확인 필요.

---

## 최종 판정

```
┌─────────────────────────────────────────────────────┐
│  판정: NEEDS_USER_DECISION                           │
│                                                       │
│  이유:                                                │
│  - architect, debugger, devops 3개가 CRITICAL 수준   │
│    (패치 내용이 새로운 설계 결정을 포함하므로 자동   │
│    패치 불가 — 사용자 의도 확인 필요)                │
│  - planner SoT 불일치는 G1-G8을 "SoT로 승격"할지    │
│    "runtime-only로 유지"할지 결정 필요               │
│  - 네이밍서비스 가드 추가는 16개 파일 일괄 수정으로  │
│    사용자 승인 후 진행해야 함                         │
│                                                       │
│  자동 안전 패치 가능:                                 │
│  - ux-validator DONE/PARTIAL/BLOCKED 추가            │
│  - explorer 출력 계약 추가                            │
│  - reviewer/validator 입력 계약 추가                  │
│  - tester BLOCKED 조건 추가                          │
└─────────────────────────────────────────────────────┘
```

---

*이 보고서는 수정 없이 진단/감사 결과만 담습니다. 모든 패치는 사용자 승인 후 별도 진행합니다.*
