# Automation Quality (Hard Rules)

IMPORTANT: 이 5개 규칙은 모든 `/auto` 파이프라인의 품질 마지노선입니다. 위반은 즉시 파이프라인 중단 사유입니다.

## R1 — Abstract Input Triggers Discovery, Not Implementation

WHEN the user input is abstract (e.g., "X 만들어줘", "Y 기능 추가해줘", "Z 사이트 만들어줘"),
THE SYSTEM SHALL NOT proceed directly to implementation.

THE SYSTEM SHALL FIRST:
1. Classify the project domain (web app / CLI / library / data pipeline / SaaS / e-commerce / content / other)
2. Run the universal Discovery Interview (see `content/skills/discovery.md`)
3. If a domain-specific template exists at `templates/shared/discovery-{domain}.md.tmpl`, run those questions IN ADDITION to the universal 9

**Abstract input detection signals**:
- Verbs without specific target: "만들어", "구현해", "추가해", "make", "build", "create"
- Noun phrases without metrics: "추천 시스템", "검색 기능", "결제 흐름"
- No file path / function name / module reference
- Word count > 3 AND specific implementation target undefined

**Anti-pattern**: 사용자가 "한국 이름 추천 사이트 만들어줘"라고 했을 때, AI가 바로 SPEC을 만들기 시작하는 행위. 도메인 분류 + 인터뷰 없이 진행 금지.

## R2 — Empty Required Answer Halts the Pipeline

WHEN a required Discovery answer is empty, vague, or matches `reject_if_contains`,
THE SYSTEM SHALL NOT substitute a plausible default.

**Prohibited substitutions**:
- 무료/유료 결과 수가 미정 → AI가 "3개/10개" 같은 임의값으로 채우기
- 디자인 톤 미정 → AI가 학습 분포 평균(=대중적 디자인)으로 채우기
- 점수 임계값 미정 → 0.5, 0.7 같은 더미 상수로 채우기
- 시드 데이터 출처 미정 → LLM이 한자/이름 풀을 추측 생성

**Required action**: Halt with the BLOCKED report format:

```
🐙 BLOCKED ──────────────────────────
  미답변 필수 항목:
  - 무료/유료 결과 수가 정의되지 않았습니다.
  - 이 값은 상품 구조와 API 응답 스키마에 직접 영향을 주므로
    구현을 진행할 수 없습니다.
  
  다음 행동:
    /auto plan "..." --rediscover  로 누락 항목 답변
```

**Exception**: WHEN `--auto` flag set AND `autopus.yaml → discovery.auto_skip = true`,
THE SYSTEM MAY proceed with AI-generated defaults BUT MUST flag each substituted value
as `[AI-GUESSED — NEEDS CONFIRMATION]` in the PRD.

## R3 — Master Plan / PRD Must Contain 10 Mandatory Items

WHEN generating a PRD or Master Plan, THE SYSTEM SHALL include ALL 10 items below.
Missing any item triggers the PRD Quality Gate (Step 1.6) to FAIL.

| Item | Source | Section name in PRD |
|---|---|---|
| 1. 타겟 사용자 | Discovery Q1 | Target Users |
| 2. 무료/유료 정책 | Discovery Q2 | Tier Structure / Pricing |
| 3. 핵심 기능 | Discovery Q3 | Core Mechanism |
| 4. 응답 JSON 예시 | Discovery Q5 | Response Schema |
| 5. DB 변경 항목 | Derived | Data Model Changes |
| 6. API 목록 | Derived | API Surface |
| 7. 디자인 톤 | Discovery Q6 | Design Tone |
| 8. 금기 사항 | Discovery Q6 | Out of Scope / Prohibited |
| 9. 외부 API / 시드 데이터 출처 | Discovery Q7, Q8 | Data Sources / External Dependencies |
| 10. 완료 기준 | Discovery Q9 | Definition of Done |

PRDs missing items 5 or 6 (DB, API) → the planner MUST add them via codebase analysis.
PRDs missing items 1-4, 7-10 → re-run planner with the discovery file as input.

## R4 — SPEC Must Contain Testable Acceptance Criteria

WHEN generating a SPEC, THE SYSTEM SHALL require `acceptance.md` to contain at least
one Given-When-Then scenario per requirement, with concrete input/expected output values.

**Forbidden in acceptance.md**:
- "테스트가 통과해야 한다" (vague — what test?)
- "사용자가 만족해야 한다" (unmeasurable)
- "성능이 좋아야 한다" (no number)

**Required in acceptance.md**:
- Concrete input value: "POST /api/recommendations with {birthDate: '2025-01-01', ...}"
- Concrete expected output: "response.items.length === 3, response.items[0].score >= 0.7"
- Concrete test command: "pytest tests/test_recommend.py::test_quick_mode -v"

**Post-implementation requirement**: After `/auto go` completes, THE SYSTEM SHALL run:
- Execution validation: actually start the backend/frontend and call the API
- Output quality validation: verify response contains all required fields with non-empty values
- See R5 for completion declaration.

## R5 — Completion Declaration Requires Evidence

WHEN declaring a task/pipeline complete, THE SYSTEM SHALL NOT use phrases like
"완료했습니다" or "Done" without a structured Completion Report.

**Required Completion Report sections**:

```
🐙 Completion Report ─────────────────
  구현 완료 항목:
    - [list of files created/modified with line counts]
  
  미구현 항목:
    - [list of features in SPEC not yet implemented, with reasons]
  
  스텁으로 남은 항목:
    - [list of functions/endpoints that are stubs, with TODO references]
  
  외부 API 키 필요 항목:
    - [list of integrations awaiting credentials, e.g., PROKERAIA_API_KEY]
  
  시드 데이터 필요 항목:
    - [list of tables/files needing human-curated data, e.g., hanja_mappings (~8000 rows)]
  
  테스트 결과:
    - Unit:      N pass / M fail
    - Integration: N pass / M fail
    - E2E:       N pass / M fail (or "not run, reason: ...")
  
  사용자가 직접 확인해야 할 화면:
    - [list of URLs/routes the user should open and verify visually]
  
  다음 수정 우선순위:
    1. [highest priority follow-up]
    2. [next]
    3. [next]
```

**Empty sections** MUST appear as `- 없음` (not omitted). The user needs to see
that the AI considered each axis.

**No Completion Report = No Completion Declaration**. The pipeline status stays
`in_progress` until the report is rendered.

## Ref

- `content/skills/discovery.md` — universal interview + quality gate
- `templates/shared/discovery-universal.md.tmpl` — question bank
- `.claude/skills/auto/SKILL.md` — pipeline integration (Steps 1.4, 1.6)
