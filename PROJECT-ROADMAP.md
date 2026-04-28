# 🐙 Autopus 가상 스튜디오 마스터 로드맵

> 이 파일은 프로젝트의 전체 진행 상황을 추적하는 '마스터 체크리스트'입니다. 모든 작업 시작 전 이 파일을 먼저 확인합니다.

## 📋 현재 진행 상황 (v3.5 진행 중)

### 1. 핵심 구조 (Workflow Engine)
- [x] Execution Manager (실행 상태 시각화)
- [x] Node System (16인 에이전트 정의)
- [>] Workflow Engine (노드 간 데이터 전달 및 연쇄 실행) - **진행 중**
- [ ] Trigger System (파일 감지/Cron 트리거)

### 2. UI/UX (Visual Editor)
- [x] 실시간 실행 상태 표시 (Pulse 애니메이션)
- [x] 노드별 설정 패널 (Right Drawer)
- [x] 실행 결과 미리보기 (Overlay Report)
- [x] 파일 탐색기 & 코드 뷰어 (Left Sidebar)
- [>] 노드 연결 시각화 (SVG Dynamic Edges) - **진행 중**
- [ ] 드래그 앤 드롭 노드 편집

### 3. 노드 타입 & AI 연동
- [x] AI 노드 (Claude, Gemini, Codex v2.0 연결)
- [x] 파일 처리 노드 (체크박스 컨텍스트 연동)
- [x] 실시간 API 키 등록 시스템
- [ ] 로직 노드 (IF/Loop 조건 분기)

### 4. 저장 및 보안
- [x] 워크플로우 상태 저장 및 로드 (Persistence v3.1)
- [x] API Key 메모리 반영 시스템

## 🛠️ 다음 작업 예약
1. **[v3.5] SVG 동적 연결선 구현** (현재 작업)
2. **[v3.6] 파일 변경 감지 트리거 (Watch Mode)**
3. **[v3.7] 에러 복구 루프 (Auto-Fix Flow)**
