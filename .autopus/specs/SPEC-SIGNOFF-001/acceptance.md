# SPEC-SIGNOFF-001: 수락 기준

## AC-01 — 이메일 문자열 완전 치환 ✅ PASS

Given `pkg/lore/writer.go`, `pkg/lore/writer_test.go`, `pkg/adapter/codex/codex_plugin_manifest.go`, `internal/cli/check_rules.go`, `internal/cli/check_rules_lore_test.go` 파일이 존재할 때
When `grep -rn "noreply@autopus.co" pkg/ internal/cli/` 를 실행하면
Then 출력이 비어 있어야 한다 (매치 0건).

## AC-02 — 신규 이메일 반영 ✅ PASS

Given 위 5개 파일에서 기존 이메일이 있던 위치
When `grep -rn "sinmihyeon@gmail.com" pkg/ internal/cli/` 를 실행하면
Then 치환 전과 동일한 개수의 매치가 출력되어야 한다.

## AC-03 — 빌드 성공 ✅ PASS

Given 이메일 치환 완료 후
When `go build ./...` 를 실행하면
Then 빌드 에러가 없어야 한다.

## AC-04 — 테스트 통과 ✅ PASS

Given 빌드 성공 상태
When `go test ./pkg/lore/... ./internal/cli/...` 를 실행하면
Then 모든 테스트가 PASS 이어야 한다 (FAIL 0건).
