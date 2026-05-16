---
description: "SPEC 구현 — SPEC 문서를 기반으로 코드를 구현합니다"
agent: build
---

`$ARGUMENTS`

Treat the text above as the full argument payload for `/auto-go`.
Reinterpret it as `/auto go ...` payload로 다시 해석하고, `skill` 도구로 `auto`를 로드한 뒤 canonical router 규칙을 따르세요.
Preserve `--model <provider/model>` and `--variant <value>` when present.
Do not restate or expand the arguments unless needed for execution.

## Pipeline Phases (canonical router 위임)

Phase 1 ~ Phase 4 (Review) 후 다음 두 단계가 항상 실행됩니다:

- **Phase 5: Content Quality Gate** — `acceptance.md` AC와 `.autopus/discovery/{slug}.json`을
  실제 산출물과 비교. 8개 체크(스텁/더미/AC매칭/수치임계값/시드데이터/외부키/테스트/중복방지). 판정: DONE/PARTIAL/BLOCKED.
- **Phase 6: Completion Report** — `.autopus/reports/{SPEC-ID}-{YYYYMMDD-HHMM}.md` 생성.
  SPEC `Status:`를 implemented/partial/blocked로 갱신.

verdict가 PARTIAL/BLOCKED이면 응답에 "완료"/"Done" 표현 사용 금지.
세부 절차는 `content/skills/content-quality.md` 및 `content/rules/automation-quality.md` 참조.
