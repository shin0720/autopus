# SPEC-SIGNOFF-001: 리서치 노트

## 배경 컨텍스트

2026-05-15, `noreply@autopus.co` → `sinmihyeon@gmail.com` 전사 치환 작업에서
`pkg WIP 수정 금지` / `internal/cli 수정 금지` 제약으로 인해 5개 Go 소스 파일이 미처리.
비-Go 파일 23개는 이미 완료 (PowerShell batch 치환).

## 영향 분석

### pkg/lore/writer.go
Lore commit 메시지를 생성하는 핵심 파일. sign-off 문자열을 하드코딩하거나 상수로 보유할 가능성 높음.
치환 후 `auto check --lore` 검증 로직이 새 이메일 포맷을 인식해야 한다.

### pkg/lore/writer_test.go
writer.go의 출력값을 기대값과 비교하는 테스트. 기대값 문자열에 `noreply@autopus.co`가 있으면
반드시 함께 변경해야 테스트가 통과한다.

### pkg/adapter/codex/codex_plugin_manifest.go ✅
Codex plugin manifest 생성 코드. `pluginAuthor.Email` 필드에 이메일 하드코딩. T1–T4와 별도 커밋(fa5890b)으로 완료.

### internal/cli/check_rules.go
`auto check --lore` 명령의 규칙 검증 로직. sign-off 포맷 정규식이나 상수에 이메일 포함 여부 확인 필요.

### internal/cli/check_rules_lore_test.go
check_rules.go의 테스트. 기대값 문자열 치환 필요.

## 위험도

낮음. 단순 문자열 치환이며 비즈니스 로직 변경 없음.
테스트 기대값을 빠뜨리면 테스트 실패로 즉시 탐지 가능.

## Pre-existing 이슈 (이번 변경과 무관)

`pkg/adapter/codex` 테스트 6개가 Windows 경로 구분자(`\` vs `/`) 및 rule count 불일치로 실패한다.
이는 SPEC-SIGNOFF-001 작업 이전부터 존재하는 실패이며 별도 이슈로 분리된다.

## Self-Verify Summary

| Q-ID | 상태 | 파일 | 비고 |
|------|------|------|------|
| Q-CORR-01 | PASS | spec.md | 5개 파일 경로 실제 확인됨 (grep 결과 기반) |
| Q-CORR-02 | N/A | — | 새 파일 없음 |
| Q-COMP-01 | PASS | 4개 파일 | spec/plan/acceptance/research 완비 |
| Q-COMP-02 | PASS | spec+acceptance | REQ-01~03 모두 AC에 대응 |
| Q-FEAS-01 | PASS | plan.md | 단순 치환, 제약 해제 후 즉시 실행 가능 |
| Q-FEAS-03 | PASS | acceptance.md | grep + go build + go test 명령 명시 |
| Q-SEC-01 | N/A | — | 외부 입력 없음 |
| Q-SEC-02 | N/A | — | 민감 경로 없음 |
