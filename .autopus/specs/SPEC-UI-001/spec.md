# SPEC-UI-001: 16인 가상 개발팀 시각화 대시보드

## 1. 개요
Autopus-ADK의 16인 에이전트 협업 체계를 n8n 스타일의 노드 기반 UI로 시각화하여, 사용자가 기획부터 배포까지의 전체 공정을 한눈에 파악하고 제어할 수 있도록 한다.

## 2. 핵심 요구사항 (EARS)
- **WHILE** `auto ui` 서버가 구동 중인 동안, **THE SYSTEM SHALL** 브라우저를 통해 16명의 에이전트 노드를 부서별(기획, 개발, QA, 배포)로 배치하여 보여주어야 한다.
- **WHEN** 특정 에이전트가 작업을 시작하면, **THE SYSTEM SHALL** 해당 노드의 테두리를 강조(Pulse)하고 실시간 한글 로그를 출력해야 한다.
- **WHERE** 에이전트 간의 데이터 흐름이 발생할 때, **THE SYSTEM SHALL** 노드 사이의 연결선(Connector)을 따라 애니메이션 효과를 주어야 한다.
- **THE SYSTEM SHALL** 모든 텍스트를 한국어로 표시하며, 윈도우/WSL 환경에서 모두 동일한 시각적 경험을 제공해야 한다.

## 3. 에이전트 부서 배치 (16인)
- **기획부**: Planner, Spec Writer, Architect, Explorer
- **개발부**: Executor, Deep Worker, Debugger, Annotator
- **QA부**: Tester, Validator, Frontend-Specialist, UX Validator, Perf-Engineer
- **운영부**: Reviewer, Security Auditor, DevOps

## 4. 기술 스택
- **Backend**: Go (net/http), JSON API
- **Frontend**: Vanilla JS (ES6+), CSS Grid/Flex, Canvas API (for lines)
- **Data Flow**: `/api/status` (실시간 상태 동기화)

## 5. 상태 정의
- `IDLE`: 대기 중 (회색)
- `RUNNING`: 작업 중 (파란색 애니메이션)
- `SUCCESS`: 완료/승인 (초록색)
- `FAILURE`: 오류/반려 (빨간색)
