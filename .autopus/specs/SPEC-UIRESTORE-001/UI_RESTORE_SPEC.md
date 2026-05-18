# SPEC-UIRESTORE-001 — Autopus Studio UI 복구 명세

**상태**: draft  
**대상 파일**: `content/ui/dashboard.html` (2149줄, 커밋 b2a81d8 기준)  
**연관 커밋**: fb9f15e (embed checkpoint), b2a81d8 (rerun controls)

---

## 개요

Autopus Studio SPA(`dashboard.html`)에서 누락되거나 부분 구현된 UI 동작을 체계적으로 복구한다.
10개 기능 영역에 걸쳐 현재 상태 → 목표 상태를 정의하며, 코드 수정 전 GAP 기준을 확정한다.

---

## REQ-01 — 이전 결과 보기 view-footer 버튼 완성

**EARS type**: State-driven  
**Priority**: Must

### 현재 상태
`view-footer`에 버튼 3개 존재 (수정 재실행, 이 작업자 다시 실행, 여기서부터 다시 진행).

### 목표 상태
완료 노드의 이전 결과 보기 패널 하단에 다음 버튼 4종이 표시된다:

| 버튼 | ID | 동작 |
|------|----|------|
| ✅ 승인 / 다음으로 진행 | `btn-view-approve` | 현재 노드를 승인하고 연결된 다음 노드 실행 |
| ✏️ 수정 재실행 | `btn-view-rerun` | 프롬프트 수정 후 같은 노드 재실행 (기존 rerunCompletedNode) |
| ❌ 취소 | `btn-view-cancel` | closePanel() |
| 🔄 수정 후 재분석 | `btn-view-refine` | followup-footer 표시 토글 |

- `btn-view-approve` 클릭 → `attemptHistory`에 현재 결과 보존 → 다음 노드 핸드오프 실행
- "이 작업자 다시 실행", "여기서부터 다시 진행"은 view-footer 2차 영역 또는 별도 섹션으로 분리

### 수정 대상
- `content/ui/dashboard.html` L626-631 (view-footer HTML)
- `viewCompletedOutput()` 함수 (L1143)
- 새 함수 `approveAndProceed()` [NEW]

---

## REQ-02 — 작업자 노드 클릭 동작 — 메시지창 우선

**EARS type**: Event-driven  
**Priority**: Must

### 현재 상태
완료(completed) 상태 노드 클릭 시 `viewCompletedOutput()` 직접 호출 → 이전 결과 패널 열림.

### 목표 상태
완료 상태 노드 클릭 시 우측 drawer(메시지창)가 열린다.

- drawer 내부에 "📋 이전 결과 보기" 버튼이 표시된다 (`drawer-output-btn-area`)
- "이전 결과 보기" 클릭 시 overlay-panel에서 결과가 열린다
- non-completed 상태는 현재와 동일하게 `selectNode()` 또는 `openDrawer()` 호출

### 수정 대상
- `el.onclick` 핸들러 (L1302-1313)
- `openDrawer()` 함수 (L1700+)

---

## REQ-03 — drawer 취소 버튼 위치 — Windows 작업표시줄 겹침 방지

**EARS type**: Ubiquitous  
**Priority**: Must

### 현재 상태
`sidebar-right` drawer의 취소 버튼(L669)이 화면 하단 고정이 아니라 컨텐츠 흐름상 위치.
Windows 작업표시줄(48px 기본)이 하단을 가릴 때 버튼이 숨겨짐.

### 목표 상태
- drawer 하단 버튼 영역을 `position: sticky; bottom: 0` 또는 `bottom: env(safe-area-inset-bottom)` 처리
- drawer 전체 높이에 `padding-bottom: 60px` 또는 `max-height: calc(100vh - 120px)` 적용
- 취소 버튼이 항상 가시 영역 내에 위치

### 수정 대상
- CSS `.sidebar-right` (L165+)
- 취소 버튼 영역 HTML (L669)

---

## REQ-04 — 자동 진행 흐름

**EARS type**: Event-driven  
**Priority**: Should

### 현재 상태
수동 실행만 가능. 노드 연결 완료 후 자동 시작 기능 없음. 이전 작업자 복귀 기능 없음.

### 목표 상태
- "자동 진행 시작" 버튼 클릭 → 시작 노드(in-degree 0)부터 순서대로 실행
- 각 노드 완료 후 연결된 다음 노드 자동 핸드오프
- 진행 중 "⏸ 일시 정지" 버튼으로 중단 가능
- 이전 작업자 결과 불만족 시: "↩ 이전 단계로" 버튼 → 해당 노드 상태를 `pending`으로 복원, attemptHistory 보존

### 수정 대상
- 상단 툴바 영역 (L377+)
- 새 함수 `startAutoFlow()`, `pauseAutoFlow()`, `rollbackToPrevNode()` [NEW]
- `workflowState`에 `autoFlowActive`, `autoFlowPaused` 필드 추가 [NEW]

---

## REQ-05 — 프로젝트 파일 목록 유효성 검증

**EARS type**: State-driven  
**Priority**: Should

### 현재 상태
`agentFiles`는 sessionStorage에서 복원(L696). 실제 파일 존재 여부 미검증.
프로젝트 변경 시 파일 경로가 stale 상태로 남을 수 있음.

### 목표 상태
- drawer 열릴 때(또는 파일 목록 렌더링 시) `/api/files/list` 응답과 agentFiles 교차 검증
- 서버에 없는 파일은 "⚠ missing" 배지 표시 또는 자동 제거
- 프로젝트 변경(changeWorkspace) 시 agentFiles 즉시 초기화 (L842에 이미 있음 — 확인 필요)

### 수정 대상
- `openDrawer()` 내 drawer-files 렌더링 (L1740-1758)
- 새 헬퍼 `validateAgentFiles(agentId)` [NEW]

---

## REQ-06 — 연결선 UX 개선

**EARS type**: Event-driven  
**Priority**: Should

### 현재 상태
- 연결선: `stroke-dasharray:6` 점선, hover 시 빨간색(L133)
- 클릭 시 즉시 삭제(L1628)
- 삭제 전 확인 없음, × 버튼 UI 없음

### 목표 상태
- 연결선 hover 시 중앙에 × 버튼(또는 삭제 레이블) SVG foreignObject 또는 절대좌표 div 표시
- 클릭 전 확인("이 연결을 삭제합니까?") 또는 × 클릭으로 명시적 삭제
- 점선이 명확히 보이도록 stroke-width, dasharray 조정

### 수정 대상
- `drawConnections()` (L1609-1631)
- CSS `.flow-line` (L132-133)

---

## REQ-07 — 시스템 프롬프트 독립 저장

**EARS type**: Ubiquitous  
**Priority**: Should

### 현재 상태
`sys-prompt-input` oninput="saveSysPromptInMap()" — 입력 즉시 Map에 저장됨.
단, "분석 시작" 클릭 전 drawer 닫히면 저장 여부 불명확.

### 목표 상태
- "💾 저장" 버튼 추가 — 저장 시 확인 토스트 표시
- localStorage에도 저장(agentId별 키) → 페이지 새로고침 후에도 유지
- openDrawer 시 저장된 프롬프트 자동 복원

### 수정 대상
- sys-prompt-body 영역 HTML (L657-659)
- `saveSysPromptInMap()` 함수 확인/수정
- `openDrawer()` — localStorage 복원 로직 추가

---

## REQ-08 — 결과 보기 추가질문 — view-footer 연동 및 결과 구분 표시

**EARS type**: Event-driven  
**Priority**: Should

### 현재 상태
`followup-footer`(L632-638)와 `sendFollowUp()`(L1245)이 존재하나 approval 패널에서만 활성화.
view-footer(완료 결과 보기)에서는 followup-footer가 보이지 않음.

### 목표 상태
- view-footer 열릴 때 followup-footer도 함께 표시
- sendFollowUp() 실행 결과는 기존 결과와 구분 표시: 패널 내 `<hr>` + "💬 추가질문 답변" 헤더
- 추가질문 결과는 attemptHistory에 별도 타입(`type: 'followup'`)으로 저장

### 수정 대상
- `viewCompletedOutput()` (L1143): followup-footer 표시 추가
- `sendFollowUp()` (L1245): 결과 구분 렌더링
- `showPanel()` (L1780+): followup-footer 컨트롤

---

## REQ-09 — 배포자(DevOps) provider 제한

**EARS type**: State-driven  
**Priority**: Nice

### 현재 상태
모든 작업자의 provider 선택이 자유. DevOps 노드도 예외 없음.

### 목표 상태
- DevOps 노드(agentId === 'devops') 실행 시 provider 자동으로 'codex' 강제
- drawer 열릴 때 DevOps면 provider 체크박스를 codex만 선택·잠금 처리

### 수정 대상
- `renderProviderCheckboxes()` 함수
- `runAgent()` 내 provider 파라미터 결정 로직

---

## REQ-10 — 배포자 완료 후 완료창

**EARS type**: Event-driven  
**Priority**: Should

### 현재 상태
완료 노드는 status 'completed' 로 표시될 뿐, 완료창 없음.

### 목표 상태
DevOps 노드가 completed 상태로 전환될 때 완료창(모달 또는 overlay) 표시:

- 작업 요약 (전체 노드 결과 한 줄씩)
- 생성/수정 파일 목록 (agentFiles 집합)
- 실행 방법 / 테스트 방법 (DevOps output에서 파싱 또는 별도 입력)
- 남은 주의사항
- "🔄 다시 실행" / "✕ 닫기" 버튼

### 수정 대상
- SSE 이벤트 수신부 → DevOps completed 감지 분기 [NEW]
- 새 함수 `showCompletionDialog()` [NEW]
- 완료창 HTML (overlay 또는 dialog 요소) [NEW]

---

## REQ-11 — 완료 노드 재실행 — attempt 표시

**EARS type**: State-driven  
**Priority**: Should

### 현재 상태
`rerunCompletedNodeDirect()`, `resumeFromCompletedNode()` 구현됨(b2a81d8).
attemptHistory는 저장되나, UI에서 attempt 번호/이력이 표시되지 않음.

### 목표 상태
- 완료 결과 패널 헤더에 "Attempt #N" 배지 표시
- 패널 내 이전 attempt 이력 접을 수 있는 섹션("📜 이전 시도 N건" 토글)

### 수정 대상
- `viewCompletedOutput()` 패널 헤더 렌더링
- `showPanel()` 또는 별도 렌더 헬퍼

---

---

## REQ-12 — 마지막 작업 상태 복원

**EARS type**: State-driven  
**Priority**: Must

### 현재 상태
Studio 재시작 시 `loadState()`가 `changeWorkspace()` 경유로 호출된다. `active` 상태 노드가 그대로 유지되어 실행 중인 것처럼 표시된다. `loadState()`에 에러 처리가 없어 실패 시 사용자 피드백 없음.

### 목표 상태
- Studio 재시작 시 마지막 노드 배치·연결선·결과를 자동 복원한다.
- 복원 시점에 `status === 'active'` 인 노드는 `status === 'interrupted'`로 변환하여 표시한다.
- `loadState()` 실패 시 터미널에 "이전 상태를 불러오지 못했습니다. 새로 시작합니다." 안내를 출력한다.
- `interrupted` 상태 노드는 자동 재실행하지 않는다.

### 수정 대상
- `loadState()` — try/catch 추가, `active` → `interrupted` 변환
- CSS — `.node.interrupted`, `.mon-node.s-interrupted` 시각 스타일 추가
- `nodeStatusClasses` — `'interrupted'` 추가
- `getMonitorStatus()` — `'interrupted'` 케이스 추가

---

## REQ-13 — 기본 진입 화면 Builder로 변경

**EARS type**: State-driven  
**Priority**: Must

### 현재 상태
HTML에서 `mode-monitor` 버튼에 `class="active"`, `monitor-view` div에 `class="monitor-view active"`가 설정되어 기본 진입 화면이 모니터링이다. `zoom-control` 초기 opacity가 0.4(모니터 모드값). DOMContentLoaded는 savedMode='monitor'일 때만 `switchMode('monitor')`를 호출한다.

### 목표 상태
- 기본 진입 화면은 빌더(Builder)이다.
- 사용자가 마지막으로 모니터링 탭을 선택했다면(`localStorage['autopus-mode'] === 'monitor'`) 재시작 시 모니터링으로 복원한다.
- 기존 모드 전환 기능(`switchMode()`)과 localStorage 저장은 그대로 유지한다.

### 수정 대상
- HTML L346: `mode-builder` 버튼 → `class="active"` 설정
- HTML L347: `mode-monitor` 버튼 → `class=""` 설정
- HTML L351: `zoom-control` → `style="opacity: 1;"` (빌더 기본값)
- HTML L375: `monitor-view` div → `class="monitor-view"` (active 제거)
- DOMContentLoaded: savedMode='monitor'이면 switchMode('monitor'), 그 외 기본 builder 유지

---

## 범위 외 (Out of Scope)

- 백엔드 API 변경 — 이 SPEC은 `content/ui/dashboard.html` 단일 파일만 수정
- 템플릿 파일 (`templates/`) 수정
- 네이밍서비스 프로젝트 코드
- `internal/cli`, `pkg/` 수정
- 전체 dashboard.html 재작성
