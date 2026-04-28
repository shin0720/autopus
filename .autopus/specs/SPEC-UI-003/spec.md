# SPEC-UI-003: 워크플로우 영속성 저장 시스템

## 1. 개요
브라우저 새로고침이나 세션 종료 시에도 작업 상태를 유지하기 위해, 현재 워크플로우 상태를 로컬 JSON 파일로 저장하고 불러온다.

## 2. 요구사항 (EARS)
- **WHEN** 사용자가 UI에서 '저장' 버튼을 누르거나 작업이 완료되면, **THE SYSTEM SHALL** 현재의 노드 상태와 에이전트 답변을 `.autopus/workflows/state.json`에 기록해야 한다.
- **WHEN** `auto ui` 서버가 시작될 때, **THE SYSTEM SHALL** 해당 파일을 읽어 마지막 작업 상태를 UI로 전송해야 한다.
- **WHILE** 연쇄 작업이 진행되는 동안, **THE SYSTEM SHALL** 각 단계의 결과(Output)를 자동으로 누적 저장해야 한다.

## 3. 데이터 구조 (Example)
```json
{
  "projectName": "kakao",
  "lastUpdated": "2026-04-25T15:00:00Z",
  "nodes": [
    { "id": "planner", "status": "completed", "output": "계획 완료..." },
    { "id": "spec", "status": "active", "output": "" }
  ],
  "logs": [
    { "agent": "Planner", "msg": "분석을 시작합니다", "time": "15:00" }
  ]
}
```
