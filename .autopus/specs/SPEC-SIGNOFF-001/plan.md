# SPEC-SIGNOFF-001: 구현 계획

## 선행 조건

- `pkg WIP 수정 금지` 및 `internal/cli 수정 금지` 제약이 해제된 상태여야 한다. ✅ 완료

## 태스크

### T1 — pkg/lore/writer.go ✅

- 파일: `pkg/lore/writer.go`
- 작업: `noreply@autopus.co` → `sinmihyeon@gmail.com` 문자열 치환 (replace_all)
- 예상 변경 라인: 1–3줄

### T2 — pkg/lore/writer_test.go ✅

- 파일: `pkg/lore/writer_test.go`
- 작업: 동일 치환
- 주의: 테스트 기대값(expected string)에도 이메일이 있으면 함께 변경

### T3 — pkg/adapter/codex/codex_plugin_manifest.go ✅

- 파일: `pkg/adapter/codex/codex_plugin_manifest.go`
- 작업: 동일 치환

### T4 — internal/cli/check_rules.go ✅

- 파일: `internal/cli/check_rules.go`
- 작업: 동일 치환

### T5 — internal/cli/check_rules_lore_test.go ✅

- 파일: `internal/cli/check_rules_lore_test.go`
- 작업: 동일 치환
- 주의: 테스트에서 sign-off 문자열을 직접 비교하는 경우 기대값도 변경

## 검증 결과 ✅

1. `grep -r "noreply@autopus.co" pkg/ internal/cli/` → 0건 확인
2. `go build ./...` → 성공
3. `go test ./pkg/lore/... ./internal/cli/...` → PASS
4. `TestRenderPluginManifestJSON_DefaultPromptsFitCodexLimits` → PASS
   - Note: `pkg/adapter/codex` 내 Windows 경로 구분자 관련 기존 실패 6개는 pre-existing 이슈 (이번 변경과 무관)
