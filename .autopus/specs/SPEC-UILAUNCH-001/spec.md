# SPEC-UILAUNCH-001: 시작.bat — UI 런처 동작 규격

- 상태: implemented
- 완료: 2026-05-19
- 구현 커밋: 222c4bc (fix(launch): start Studio minimized from launcher)

## Overview

`시작.bat`는 Autopus Studio UI 서버를 Windows에서 실행하기 위한 런처 배치 파일이다.

## Requirements

### REQ-1 포트 선점 해제 (Ubiquitous)
- Priority: Must
- 서버 시작 전에 포트 8080을 점유 중인 프로세스를 강제 종료한다.
- 이전에 창을 닫았어도 프로세스가 남아 있을 수 있으므로 항상 실행한다.

### REQ-2 CMD 창 자동 닫힘 (Ubiquitous)
- Priority: Must
- `AutopusStudio.exe ui` 실행 후 런처 CMD 창이 즉시 닫혀야 한다.
- 서버 프로세스는 최소화된 별도 창으로 계속 실행된다.

### REQ-3 실행 위치 고정 (Ubiquitous)
- Priority: Must
- 배치 파일이 어디서 실행되든 `%~dp0`를 사용해 파일 위치 기준으로 동작한다.

## Invariants

- `시작.bat`는 항상 `%~dp0AutopusStudio.exe`를 참조해야 한다.
- `start /min ""` 패턴으로 서버를 별도 프로세스로 기동 후 원래 CMD는 `exit`로 종료한다.
- 포트 정리는 LISTENING 상태만 대상으로 한다 (ESTABLISHED 상태 연결 제외).
