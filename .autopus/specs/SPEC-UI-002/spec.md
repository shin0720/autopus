# SPEC-UI-002: 시각적 auto plan 및 실시간 협업 인터페이스

## 1. 개요
사용자가 UI에서 특정 에이전트에게 업무를 할당하거나 전체 파이프라인(`auto plan`)을 실행했을 때, 에이전트 간의 데이터 흐름과 작업 결과물을 시각적으로 관리한다.

## 2. 추가 요구사항 (EARS)
- **WHEN** 에이전트가 응답을 생성하면, **THE SYSTEM SHALL** 하단 '메시지 패널'에 해당 내용을 실시간으로 스트리밍해야 한다.
- **WHEN** 작업 결과로 파일이 생성/수정되면, **THE SYSTEM SHALL** 노드 옆에 '문서 아이콘'을 표시하고 클릭 시 미리보기를 제공해야 한다.
- **WHILE** `auto plan`이 진행되는 동안, **THE SYSTEM SHALL** 사용자가 상단바에서 현재 소모 중인 토큰과 예상 비용을 확인할 수 있게 해야 한다.
- **THE SYSTEM SHALL** 에이전트가 다음 에이전트에게 바통을 넘길 때(Handoff), 시각적인 연결선 애니메이션을 활성화해야 한다.

## 3. UI 구성 요소 추가
- **하단 터미널 패널**: 에이전트들의 "생각(Thinking Process)" 출력.
- **우측 파일 미리보기**: 마크다운(.md) 및 소스 코드 뷰어.
- **상단 리소스 바**: 토큰 사용량, 소요 시간, 현재 활성 에이전트 수.

## 4. 데이터 흐름
1. UI (Prompt) -> 2. Backend (Invoke Agent) -> 3. Real-time Log (WebSocket/SSE) -> 4. UI (Message/File Update)
