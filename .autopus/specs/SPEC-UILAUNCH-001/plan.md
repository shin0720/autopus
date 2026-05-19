# SPEC-UILAUNCH-001: 구현 계획

- 상태: 전체 완료 (2026-05-19, 커밋 222c4bc)

## 대상 파일

- `시작.bat` (프로젝트 루트) ✅

## 구현 내용 (전체 완료 ✅)

### 포트 정리 (REQ-1) ✅

```bat
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080 " ^| findstr "LISTENING"') do (
    taskkill /PID %%a /F >nul 2>&1
)
```

- `netstat -ano`에서 `:8080 LISTENING` 상태의 PID만 추출
- `taskkill /F`로 강제 종료, 오류 출력 억제

### CMD 창 자동 닫힘 (REQ-2) ✅

```bat
start /min "" "%~dp0AutopusStudio.exe" ui
exit
```

- `start /min ""`: 서버를 최소화된 별도 프로세스로 실행
- `exit`: 런처 CMD 창 즉시 종료
- `pause` 제거 — 사용자 입력 대기 없음

### 실행 위치 고정 (REQ-3) ✅

- `cd /d "%~dp0"` 및 `"%~dp0AutopusStudio.exe"` 사용
- 바탕화면 단축키, 탐색기 등 어디서 실행해도 경로 안전
