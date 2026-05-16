---
description: "SPEC 작성 — 코드베이스 분석 후 EARS 요구사항, 구현 계획, 인수 기준을 생성합니다"
agent: build
---

## Step 1.4: Discovery Interview (추상적 입력 시 필수)

WHEN 사용자 입력이 추상적인 경우 (`content/rules/automation-quality.md` R1 신호 해당 — 특정 대상 없는 동사, 수치 없는 명사구, 파일 경로·함수명·모듈 참조 없음),
THE SYSTEM SHALL canonical router 실행 전에 `content/skills/discovery.md` Discovery Interview를 먼저 실행합니다.

Discovery 절차:
1. FEATURE_DESC에서 slug 생성 → `.autopus/discovery/{slug}.json` 존재 시 재사용
2. 8개 범용 질문 2라운드 전달 (Q1-Q4, Q5-Q8)
3. 각 답변 검증, 재질문 한도 2회, 2회 FAIL 시 STOP
4. 답변을 `.autopus/discovery/{slug}.json`에 저장

`--skip-prd` 설정 시 Discovery도 건너뜁니다.

---

`$ARGUMENTS`

Treat the text above as the full argument payload for `/auto-plan`.
Reinterpret it as `/auto plan ...` payload로 다시 해석하고, `skill` 도구로 `auto`를 로드한 뒤 canonical router 규칙을 따르세요.
Preserve `--model <provider/model>` and `--variant <value>` when present.
Do not restate or expand the arguments unless needed for execution.
