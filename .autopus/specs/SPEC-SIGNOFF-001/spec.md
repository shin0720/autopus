# SPEC-SIGNOFF-001: Lore Commit Sign-off 이메일 변경 (Go 소스)

## 상태

implemented

## 배경

Lore commit sign-off 이메일을 `noreply@autopus.co`에서 `sinmihyeon@gmail.com`으로 변경하는 작업 중,
`pkg/`, `internal/cli/` WIP 수정 제약으로 인해 아래 5개 Go 소스 파일이 미처리 상태로 남았다.
나머지 23개 파일(`.md`, `.tmpl`, `.json`, `.txt`)은 이미 변경 완료.

## 범위

변경 대상 파일 (5개):

| 파일 | 종류 |
|------|------|
| `pkg/lore/writer.go` | 소스 |
| `pkg/lore/writer_test.go` | 테스트 |
| `pkg/adapter/codex/codex_plugin_manifest.go` | 소스 |
| `internal/cli/check_rules.go` | 소스 |
| `internal/cli/check_rules_lore_test.go` | 테스트 |

## 요구사항

### REQ-01
- EARS type: Ubiquitous
- Priority: Must
- THE SYSTEM SHALL replace all occurrences of the string `noreply@autopus.co` with `sinmihyeon@gmail.com` in the 5 Go source files listed above.

### REQ-02
- EARS type: Ubiquitous
- Priority: Must
- THE SYSTEM SHALL NOT change any other string, logic, or behavior in the affected files beyond the email string substitution.

### REQ-03
- EARS type: Ubiquitous
- Priority: Must
- WHEN the substitution is complete, THE SYSTEM SHALL verify that no occurrence of `noreply@autopus.co` remains in any file under `pkg/` and `internal/cli/`.

## 범위 외

- 이미 변경된 23개 비-Go 파일 재수정
- 네이밍서비스 프로젝트 코드
- 기타 비즈니스 로직 변경
