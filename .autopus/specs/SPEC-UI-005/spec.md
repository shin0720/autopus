# SPEC-UI-005: 에러 복구 루프 (Auto-Fix Loop) 시스템

## 1. 개요
테스트(`Tester`)나 검증(`Validator`) 단계에서 에러가 발생할 경우, 시스템이 자동으로 `Debugger`에게 에러 로그와 소스 코드를 전달하여 수정을 요청하고, 성공할 때까지 루프를 도는 자동 복구 시스템을 구축한다.

## 2. 요구사항 (EARS)
- **WHEN** `Validator` 또는 `Tester` 노드에서 실행 에러가 발생하면, **THE SYSTEM SHALL** 즉시 중단하지 않고 `Debugger` 노드로 루프(Loop)를 형성해야 한다.
- **WHILE** 루프가 진행되는 동안, **THE SYSTEM SHALL** UI에 "자동 수정 시도 중 (N회차)" 메시지를 표시해야 한다.
- **WHEN** `Debugger`가 수정을 완료하면, **THE SYSTEM SHALL** 다시 `Validator`를 실행하여 검증을 재시도해야 한다.
- **WHERE** 최대 시도 횟수(예: 3회)를 초과할 경우, **THE SYSTEM SHALL** 사용자에게 수동 개입 프롬프트를 띄워야 한다.

## 3. 시각적 연출
- **빨간색 피드백 선**: 검증 실패 시 `Validator`에서 `Debugger`로 거꾸로 흐르는 빨간색 연결선 표시.
- **상태 배지**: 노드 상단에 "Fixing..." 배지 추가.
