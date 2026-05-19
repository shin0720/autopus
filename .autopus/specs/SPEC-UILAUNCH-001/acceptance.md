# SPEC-UILAUNCH-001: 수락 기준

- 검증 완료: 2026-05-19
- 검증 커밋: 222c4bc

## AC-1 포트 선점 해제 (REQ-1)

Given 포트 8080을 점유 중인 프로세스가 존재한다.
When 시작.bat를 실행한다.
Then 해당 프로세스가 강제 종료된 뒤 AutopusStudio.exe가 정상 시작된다.

Given 포트 8080을 점유 중인 프로세스가 없다.
When 시작.bat를 실행한다.
Then taskkill 오류 없이 AutopusStudio.exe가 정상 시작된다.

Given 포트 8080에 ESTABLISHED 상태 연결만 존재하고 LISTENING 프로세스가 없다.
When 시작.bat를 실행한다.
Then ESTABLISHED 연결은 종료되지 않고 AutopusStudio.exe가 정상 시작된다.

**결과: PASS** — 2026-05-19 검증: 포트 8080 LISTENING 프로세스 없음 확인 후 AutopusStudio.exe(PID=58432) 정상 기동.

## AC-2 CMD 창 자동 닫힘 (REQ-2)

Given 시작.bat가 실행되었다.
When AutopusStudio.exe 프로세스가 별도 창으로 기동된다.
Then 런처 CMD 창이 사용자 입력 없이 즉시 닫힌다.

Given 시작.bat가 실행되었다.
When AutopusStudio.exe 창이 열린다.
Then AutopusStudio.exe 프로세스는 최소화된 별도 창으로 계속 실행된다.

**결과: PASS** — 2026-05-19 검증: `start /min ""` 적용, MainWindowTitle 비어 있음(최소화 상태 확인). HTTP 200 응답 — UI 접근 차단 없음.

## AC-3 실행 위치 고정 (REQ-3)

Given 시작.bat의 바탕화면 단축키를 통해 실행한다.
When 배치 파일이 시작된다.
Then AutopusStudio.exe는 배치 파일과 동일한 디렉토리에서 실행된다.

Given Windows 탐색기에서 다른 드라이브 위치로 이동한 상태에서 시작.bat를 실행한다.
When 배치 파일이 시작된다.
Then AutopusStudio.exe는 배치 파일과 동일한 디렉토리에서 실행된다.

**결과: PASS** — `%~dp0` 패턴 확인 완료.

## UI 검증 결과 (2026-05-19)

| 검증 항목 | 결과 |
|-----------|------|
| HTTP 200 (http://localhost:8080) | PASS |
| 신규 UI 문구: 이전 단계 결과 보기 | PASS — FOUND |
| 신규 UI 문구: 현재 작업 결과 보기 | PASS — FOUND |
| 신규 UI 문구: result-footer | PASS — FOUND |
| 신규 UI 문구: viewPrevStageOutput | PASS — FOUND |
| 구버전 문구: 수정 재실행 | PASS — ABSENT |
| 구버전 문구: approval-footer | PASS — ABSENT |
| 구버전 문구: view-footer | PASS — ABSENT |
| Builder 기본 진입 | PASS — builder/workflow/stage 요소 확인 |
| 마지막 프로젝트/워크스페이스 복원 | PASS — localStorage/workspace/project 로직 확인 |
| /min 적용 후 UI 접근 가능 | PASS — HTTP 200 재확인, 접근 차단 없음 |
