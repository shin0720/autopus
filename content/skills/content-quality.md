---
name: content-quality
description: Post-implementation Content Quality Gate. Verifies that an implemented SPEC actually delivers substantive content, not just stubs/placeholders/dummy values. Produces the Completion Report.
triggers:
  - content quality
  - 콘텐츠 품질
  - completion report
  - 완료 리포트
  - phase 5
  - phase 9
category: workflow
level1_metadata: "Stub detection, AC verification, structured completion report, DONE/PARTIAL/BLOCKED status"
---

# Content Quality Skill

`/auto go` 종료 직전에 실행되는 게이트. 구현이 "돌아가긴 하지만 내용이 비어 있음"
상태로 완료 선언되는 것을 막는 게 유일한 목적입니다.

본 스킬은 두 단계로 구성됩니다:
- Stage A — Content Quality Checks (자동 검사 8개)
- Stage B — Completion Report Generation (structured report → `.autopus/reports/`)

판정 결과 3종:
- **DONE** — 모든 체크 통과. 마무리 가능.
- **PARTIAL** — 일부 AC 미구현이지만 테스트는 통과. 사용자 확인 필요.
- **BLOCKED** — 핵심 AC 누락 또는 스텁/더미가 코어 경로에 존재. 완료 선언 불가.

## Stage A — Content Quality Checks

총 8개 체크. ALL PASS면 DONE. 하나라도 FAIL이면 PARTIAL 또는 BLOCKED로 강등.

### Check 1: Acceptance Criteria Coverage

INPUTS:
- `acceptance.md` Given-When-Then scenarios
- Implementation artifacts (files modified in this SPEC)

PROCEDURE:
1. Parse each scenario in `acceptance.md`
2. For each scenario, search the codebase for an implementation reference:
   - API endpoint mentioned → search route definitions
   - Function name mentioned → search function definitions
   - DB table/column → search schema/migration files
3. Mark each scenario as: COVERED / MISSING / PARTIAL

PASS criterion: 0 MISSING, ≤ 20% PARTIAL.

### Check 2: Stub / Placeholder Detection

PROCEDURE — search files modified in this SPEC for these markers:

| Pattern | Stub indicator |
|---|---|
| `TODO`, `FIXME`, `XXX`, `HACK` (in code, not test) | Reminder of unfinished work |
| `NotImplementedError`, `not_implemented` | Python stub |
| `throw new Error("not implemented")` | TS/JS stub |
| `panic("not implemented")` | Go stub |
| `return None  #` followed by TODO/stub | Python lazy return |
| `pass  # TODO`, `pass  # placeholder` | Python empty body |
| `return [];  // stub`, `return {};  // stub` | TS/JS empty return |
| `placeholder`, `stub`, `dummy` in function/variable name | Naming-level stub |
| Literal `= 0.5` in scoring/ranking code (where Discovery `numeric_ac` defines a real threshold) | Dummy threshold |

EXCEPTIONS:
- Test files (`*_test.go`, `*.test.ts`, `test_*.py`) may contain `TODO` for future test cases — only count as stub if marked `# CRITICAL` or `# blocker`
- Comments inside `//` or `#` are read, but `TODO(@username)` style is treated as backlog (not blocker) unless severity is annotated

PASS criterion: 0 stubs in non-test code paths touched by this SPEC.

### Check 3: Numeric Threshold Verification

INPUTS:
- `.autopus/discovery/{slug}.json` `numeric_ac` answer
- Implementation code (scoring, ranking, validation logic)

PROCEDURE:
1. Extract each numeric threshold from `numeric_ac` (e.g., "traditional_fit ≥ 0.7")
2. Search code for that threshold constant
3. Confirm the constant is referenced from the relevant code path (e.g., scoring function)

PASS criterion: every `numeric_ac` threshold is reflected in code constants AND used (not just declared).

FAIL example: discovery says "traditional_fit ≥ 0.7" but code has `traditional_fit = 0.5` hardcoded.

### Check 4: Seed Data Presence

INPUTS:
- `.autopus/discovery/{slug}.json` `data_strategy` answer
- Database / data file state

PROCEDURE:
- WHEN `data_strategy.choice = "manual"` or `"mixed"`:
  - Identify tables/files marked as human-curated
  - Query row counts or line counts
  - Compare against minimum threshold (default: ≥ 10 rows for any "core data" table)
- WHEN `data_strategy.choice = "external"`:
  - Verify external data source path exists
  - Verify import/sync script exists
- WHEN `data_strategy.choice = "ai_generated"`:
  - Verify generation script ran successfully
  - Verify output file is non-empty

PASS criterion: all expected data sources have non-empty content above threshold.

FAIL example: `hanja_mappings` table has 0 rows.

### Check 5: External Integration Status

INPUTS:
- `.autopus/discovery/{slug}.json` `integration_ids` answer
- `.env.example` and actual environment

PROCEDURE:
1. Parse integration list from discovery
2. For each integration:
   - Find env var name in `.env.example`
   - Check if actual env has the key set (presence only, value not logged)
   - Find stub fallback in code (e.g., `ManseEngineStub`)
3. Classify:
   - `real_and_set`: env var set, real engine used
   - `real_but_missing`: real engine code but env var absent → BLOCKED
   - `stub_acknowledged`: stub class used, discovery answer marked "stub_then_real" → ACCEPTABLE
   - `silent_stub`: stub used but discovery answer says "real" → BLOCKED

PASS criterion: no `real_but_missing`, no `silent_stub`.

### Check 6: Test Suite Result

PROCEDURE — detect project type and run tests:

| Project Type | Test Command |
|---|---|
| Go | `go test ./...` |
| Python (pytest) | `pytest -q` |
| Node (jest) | `npm test --silent` or `pnpm test` |
| Mixed | Run all detected runners |

CAPTURE:
- Pass count, fail count, skip count per category (unit / integration / e2e)
- List of failed test names (first 10)
- Total duration

PASS criterion: 0 failures. WHEN ≥ 1 failure → not DONE (could be PARTIAL or BLOCKED).

### Check 7: User-Verifiable Surfaces

PROCEDURE — produce a list of surfaces the user should open and verify manually:

1. Frontend: scan `app/*/page.tsx` (Next.js) or equivalent for routes touched by this SPEC
2. Backend: list API endpoints touched by this SPEC with example curl commands
3. Admin: list admin routes if Discovery `admin_scope` answer was provided
4. Reports: list any generated artifact paths (PDFs, exports)

OUTPUT: a checklist of URLs / commands / paths for the report.
This check itself never FAILs — it produces user guidance only.

### Check 8: Rework Cleanup / Duplicate Prevention

PROCEDURE — WHEN this SPEC modifies or replaces existing functionality:

1. Identify change type: new addition vs. rework (modifying / replacing existing code)
2. WHEN rework detected, scan for:

| Category | Signal |
|---|---|
| Duplicate implementation | Old + new function/class/component both present in codebase |
| Superseded route / API | Old endpoint still registered alongside new one |
| Superseded component | Old UI component still imported or rendered anywhere |
| Dead code candidate | Function / class no longer called from any active entry point |
| Duplicate endpoint | Same HTTP method + path registered in 2+ places |
| Duplicate page | Same URL route handled by 2+ page components |
| Legacy delete candidate | File / module that existed only to serve replaced functionality |

3. For each found item, classify:
   - `DELETE_CANDIDATE`: no remaining callers — safe to remove, must appear in Completion Report
   - `KEEP_WITH_REASON`: still referenced — document the reason explicitly
   - `AMBIGUOUS`: unclear usage — flag for user decision

PASS criterion:
- 0 `DELETE_CANDIDATE` items left undocumented in the Completion Report
- 0 `AMBIGUOUS` items left unresolved
- 0 active duplicate endpoints / duplicate pages / duplicate components

FAIL → PARTIAL: undocumented `DELETE_CANDIDATE` or unresolved `AMBIGUOUS` items present
FAIL → BLOCKED: active duplicate endpoint, duplicate page, or duplicate component detected

Note: WHEN change type is purely additive (no existing functionality replaced), this check auto-PASSes with result "N/A — additive change".

## Stage B — Completion Report Generation

Always emit a structured report to `.autopus/reports/{SPEC-ID}-{YYYYMMDD-HHMM}.md`.

Format follows `content/rules/automation-quality.md` R5 (Completion Report).

```markdown
# Completion Report — {SPEC-ID}

**Status**: {DONE | PARTIAL | BLOCKED}
**Generated**: {ISO timestamp}
**Pipeline**: /auto go {SPEC-ID} [flags...]
**Discovery**: .autopus/discovery/{slug}.json (or "N/A")

## Status Verdict

- Stage A checks: {N}/8 PASS
- Failing checks: {list with one-line summary each}
- Verdict logic: {rule that produced this status}

## 구현 완료 항목

- {file path}: {function/class}, {lines added}
- ...

## 미구현 항목

- AC {scenario-id}: {scenario name} — {reason}
- ...

## 스텁 / 더미 / placeholder 항목

- {file}:{line} — {pattern} — {snippet}
- ...

## 외부 API 키 필요 항목

- {env var name} — {integration description}
- ...

## 시드 데이터 필요 항목

- {table or file path} — current: {N} rows, expected: ≥ {M}
- ...

## 테스트 결과

- Unit:        N pass / M fail / K skip ({duration})
- Integration: N pass / M fail / K skip ({duration})
- E2E:         N pass / M fail / K skip ({duration})
- Failed tests: {list of names, max 10}

## 사용자가 직접 확인해야 할 화면

- [ ] {URL or route} — {what to verify}
- [ ] {command} — {expected output shape}
- ...

## 레거시 정리 후보

- {file/function/route} — DELETE_CANDIDATE | KEEP({reason}) — {last reference location}
- ...

## 다음 수정 우선순위

1. {highest priority blocker, with reason}
2. {next}
3. {next}
```

Empty sections MUST appear as `- 없음` (not omitted).

## Status Verdict Logic

```
verdict = DONE
IF Check 1 has MISSING items → verdict = PARTIAL
IF Check 2 finds stubs in core paths → verdict = BLOCKED
IF Check 3 finds dummy threshold → verdict = BLOCKED
IF Check 4 finds empty core seed → verdict = PARTIAL (or BLOCKED if core)
IF Check 5 finds silent_stub or real_but_missing → verdict = BLOCKED
IF Check 6 has failures:
  - if test count > 50% pass → verdict = max(verdict, PARTIAL)
  - if test count ≤ 50% pass → verdict = BLOCKED
IF Check 8 finds undocumented DELETE_CANDIDATE or unresolved AMBIGUOUS → verdict = max(verdict, PARTIAL)
IF Check 8 finds active duplicate endpoint/page/component → verdict = BLOCKED
```

Precedence: BLOCKED > PARTIAL > DONE.

## --loop Integration

WHEN `LOOP_MODE = true` AND verdict ≠ DONE:

1. Build a fix prompt from the report's "스텁/더미", "미구현 항목", "테스트 결과", "레거시 정리 후보" sections
2. Spawn an `executor` subagent with that prompt
3. Re-run Stage A after the executor returns
4. Repeat up to 2 iterations (consumes Phase 5 retry budget)
5. After 2 failed iterations → STOP and finalize report with current verdict

Circuit breaker rule: WHEN 2 consecutive iterations produce IDENTICAL failing-check sets,
stop loop and mark BLOCKED. The same set of failures twice means the executor cannot
self-resolve them.

## --auto Mode Behavior

WHEN `AUTO_MODE = true`:
- Stage A runs unchanged
- Stage B report is generated
- IF verdict = DONE: pipeline exits normally
- IF verdict = PARTIAL: pipeline exits with status code 0 BUT the report is the user's
  primary deliverable — auto mode does NOT silently mark it as fully done
- IF verdict = BLOCKED: pipeline exits with non-zero status code

## Result Format

```
🐙 content quality ───────────────────
  Checks: {N}/8 PASS
  Verdict: {DONE | PARTIAL | BLOCKED}
  Report: .autopus/reports/{SPEC-ID}-{timestamp}.md
  {next-action guidance}
```

WHERE next-action guidance is one of:
- DONE → "다음: /auto sync {SPEC-ID}"
- PARTIAL → "다음: 리포트 확인 후 잔여 항목 처리"
- BLOCKED → "다음: 리포트의 '다음 수정 우선순위' 1번부터 해결"

## Ref

- `content/rules/automation-quality.md` R4, R5 — testable AC + Completion Report rules
- `content/skills/discovery.md` — discovery answers consumed by Checks 3, 4, 5
- `.autopus/reports/` — output directory
