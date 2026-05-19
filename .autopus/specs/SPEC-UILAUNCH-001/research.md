# SPEC-UILAUNCH-001: 리서치 및 자체 검증

## 배경

`시작.bat`는 세 번의 반복 수정을 거쳐 현재 형태에 도달했다.

1. **최초 분실**: 파일이 git에 커밋된 적 없어 복구 불가능 → `AutopusStudio.exe ui --help` 출력 기반으로 재생성
2. **포트 정리 추가**: 창을 닫아도 프로세스가 남아 포트 점유 문제 발생 → `netstat`/`taskkill` 추가
3. **CMD 창 자동 닫힘**: `AutopusStudio.exe ui`가 블로킹 프로세스이므로 `pause` 제거만으로는 해결 불가 → `start /min ""` 패턴으로 분리 실행 후 `exit`

## 기술 선택 근거

### `start /min "" exe args` vs `start "" exe args`

- `/min`은 서버 창을 최소화 상태로 기동 — 백그라운드처럼 실행하지만 작업 표시줄에서 확인 가능
- `/min` 없이 `start "" exe`를 사용하면 서버 창이 포그라운드로 열려 사용자 화면을 가림

### `netstat -ano | findstr ":8080 " | findstr "LISTENING"` 이중 필터

- 첫 번째 `findstr`: 8080 포트 관련 행 추출 (`:8080 `에 공백 포함 — `:80800` 같은 포트 오탐 방지)
- 두 번째 `findstr "LISTENING"`: ESTABLISHED/TIME_WAIT 등 연결 상태 행 제외, LISTENING 프로세스만 종료

### `tokens=5`

- `netstat -ano` 출력의 5번째 토큰이 PID
- 예: `TCP 0.0.0.0:8080 0.0.0.0:0 LISTENING 12345` → tokens=5 → `12345`

## 구현 검증 결과 (2026-05-19)

커밋 222c4bc 기준으로 실제 실행 검증을 완료했다.

| 항목 | 결과 | 비고 |
|------|------|------|
| AutopusStudio.exe PID | 58432 | 정상 기동 확인 |
| 포트 8080 LISTENING 프로세스 | 없음 | 기동 전 netstat 확인 |
| MainWindowTitle | 빈 문자열 | `/min` 효과 — 최소화 상태 |
| HTTP 200 (localhost:8080) | PASS | `/min` 후에도 UI 접근 차단 없음 |
| 신규 UI 문구 (result-footer 등) | PASS — FOUND | 4개 모두 확인 |
| 구버전 문구 (approval-footer 등) | PASS — ABSENT | 3개 모두 미발견 |
| Builder 기본 진입 | PASS | builder/workflow/stage 요소 확인 |
| 마지막 프로젝트 복원 | PASS | localStorage 로직 확인 |

**결론**: `/min` 플래그는 CMD 창 표시 방식에만 영향을 주며, HTTP 서버 접근성에는 영향 없음.

## Self-Verify Summary

| Q-ID | Status | Attempt | Files | Reason |
|------|--------|---------|-------|--------|
| Q-CORR-01 | PASS | 1 | 시작.bat | `%~dp0AutopusStudio.exe`, `netstat`/`taskkill`/`start` 모두 실제 Windows 명령어 |
| Q-CORR-02 | PASS | 1 | spec.md, plan.md | 신규 파일 없음, 기존 `시작.bat` 수정 사항만 기술 |
| Q-CORR-03 | PASS | 1 | acceptance.md | Given/When/Then 형식 사용 |
| Q-COMP-01 | PASS | 1 | 전체 | 4개 파일 모두 역할 명확 |
| Q-COMP-02 | PASS | 1 | spec.md↔acceptance.md | REQ-1/2/3 모두 AC에서 다룸 |
| Q-COMP-03 | PASS | 1 | spec.md | EARS type(Ubiquitous), 조건, 관측 지점 명시 |
| Q-FEAS-01 | PASS | 1 | 시작.bat | 배치 파일 단순 수정, 런타임 코드 변경 없음 |
| Q-FEAS-02 | PASS | 1 | 시작.bat | 프로젝트 루트에 위치, 경로 고정 |
| Q-FEAS-03 | PASS | 1 | acceptance.md | 수동 실행으로 검증 가능 |
| Q-STYLE-01 | PASS | 1 | spec.md | 모호어 없음 |
| Q-STYLE-02 | PASS | 1 | spec.md | Priority: Must, EARS: Ubiquitous 분리 |
| Q-STYLE-03 | PASS | 1 | acceptance.md | 완결 문장, bare Given/When/Then 사용 |
| Q-SEC-01 | N/A | 1 | — | 로컬 배치 파일, 외부 입력 없음 |
| Q-SEC-02 | N/A | 1 | — | 비밀값/credential 없음 |
| Q-SEC-03 | N/A | 1 | — | 별도 로그/아티팩트 없음 |
