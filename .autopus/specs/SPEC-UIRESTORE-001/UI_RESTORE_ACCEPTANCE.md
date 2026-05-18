# SPEC-UIRESTORE-001 — 수락 기준

**검증 환경**: Autopus Studio 로컬 실행 (`./auto studio` 또는 `./AutopusStudio.exe`)  
**기준 파일**: `content/ui/dashboard.html`

---

## AC-01 — view-footer 버튼 (REQ-01)

**Given** Autopus Studio가 실행 중이고 하나의 에이전트 노드가 `completed` 상태이다.  
**When** 완료 노드를 클릭하여 이전 결과 패널이 열린다.  
**Then** 패널 하단 view-footer에 다음 4개 버튼이 모두 표시된다:
- "✅ 승인 / 다음으로 진행"
- "✏️ 수정 재실행" 
- "❌ 취소"
- "🔄 수정 후 재분석"

**And** "승인 / 다음으로 진행" 클릭 시 다음 연결 노드가 자동 실행된다.  
**And** "수정 후 재분석" 클릭 시 followup-footer 입력 영역이 표시된다.  
**And** "취소" 클릭 시 패널이 닫힌다.

---

## AC-02 — 노드 클릭 → 메시지창 우선 (REQ-02)

**Given** 하나의 에이전트 노드가 `completed` 상태이다.  
**When** 사용자가 해당 노드를 단일 클릭한다.  
**Then** 우측 drawer(메시지창)가 열린다.  
**And** drawer 내부에 "📋 이전 결과 보기" 버튼이 표시된다.  
**And** 해당 버튼 클릭 시 overlay-panel에 이전 결과가 표시된다.

**Given** 에이전트 노드가 `idle` 또는 `awaiting-approval` 상태이다.  
**When** 사용자가 해당 노드를 단일 클릭한다.  
**Then** 기존 동작(selectNode 또는 openDrawer)이 유지된다.

---

## AC-03 — drawer 취소 버튼 가시성 (REQ-03)

**Given** 화면 해상도가 1366×768이고 Windows 작업표시줄이 하단에 있다.  
**When** 에이전트 노드를 클릭하여 drawer가 열린다.  
**Then** 취소 버튼이 스크롤 없이 화면에서 보인다.  
**And** 취소 버튼이 작업표시줄 영역과 겹치지 않는다.

---

## AC-04 — 자동 진행 흐름 (REQ-04)

**Given** 노드 A → 노드 B → 노드 C 순서로 연결된 워크플로우가 있다.  
**When** 사용자가 "자동 진행 시작" 버튼을 클릭한다.  
**Then** 노드 A가 먼저 실행된다.  
**And** 노드 A 완료 후 노드 B가 자동으로 실행된다.  
**And** 노드 B 완료 후 노드 C가 자동으로 실행된다.

**When** 노드 B 실행 중 "일시 정지" 버튼을 클릭한다.  
**Then** 노드 B 완료 후 노드 C는 자동 실행되지 않는다.

**Given** 노드 B의 결과가 마음에 들지 않는다.  
**When** "↩ 이전 단계로" 버튼을 클릭한다.  
**Then** 노드 B의 현재 결과가 `attemptHistory`에 보존된다.  
**And** 노드 A의 상태가 `pending`으로 변경되어 재실행 가능한 상태가 된다.

---

## AC-05 — 파일 목록 유효성 (REQ-05)

**Given** agentFiles에 `old-file.py`가 저장되어 있으나 서버에는 해당 파일이 없다.  
**When** 에이전트 drawer가 열린다.  
**Then** `old-file.py`는 "⚠ missing" 배지와 함께 표시되거나 목록에서 제외된다.  
**And** 존재하는 파일은 정상 표시된다.

---

## AC-06 — 연결선 UX (REQ-06)

**Given** 두 노드 간 연결선이 그려져 있다.  
**When** 마우스를 연결선 위로 이동한다.  
**Then** 연결선에 "×" 삭제 UI 또는 강조 표시가 나타난다.  
**And** "×" 클릭 시 해당 연결선만 삭제된다.  
**And** 다른 연결선은 영향을 받지 않는다.

---

## AC-07 — 시스템 프롬프트 저장 (REQ-07)

**Given** 에이전트 A의 drawer에서 시스템 프롬프트를 입력했다.  
**When** 페이지를 새로고침한다.  
**Then** 에이전트 A의 drawer를 다시 열면 이전에 입력한 시스템 프롬프트가 복원된다.

---

## AC-08 — 추가질문 view 연동 (REQ-08)

**Given** 완료 노드의 이전 결과 패널이 열려 있다.  
**When** "🔄 수정 후 재분석" 버튼을 클릭한다.  
**Then** 패널 하단에 추가질문 입력 영역(followup-footer)이 표시된다.  
**When** 추가질문을 입력하고 "보내기"를 클릭한다.  
**Then** 결과 패널에 기존 결과와 구분된 "💬 추가질문 답변" 섹션이 표시된다.  
**And** 기존 결과는 변경되지 않고 유지된다.

---

## AC-09 — DevOps provider 강제 (REQ-09)

**Given** DevOps(배포자) 노드를 선택한다.  
**When** drawer의 provider 선택 목록이 표시된다.  
**Then** codex provider만 선택되어 있고 다른 provider는 비활성화(disabled) 상태이다.  
**And** "실전 분석 시작" 클릭 시 provider 파라미터가 `['codex']`로 전송된다.

---

## AC-10 — 완료창 (REQ-10)

**Given** DevOps(배포자) 노드가 마지막 노드이고 결과를 반환했다.  
**When** DevOps 노드가 `completed` 상태로 전환된다.  
**Then** 완료창(모달)이 자동으로 표시된다.  
**And** 완료창에 다음 항목이 포함된다:
  - 작업 요약 (각 노드별 1줄 요약)
  - 생성/수정 파일 목록
  - "🔄 다시 실행" 버튼
  - "✕ 닫기" 버튼  
**And** "닫기" 클릭 시 완료창이 닫히고 워크플로우 뷰로 돌아온다.

---

## AC-11 — attempt 이력 표시 (REQ-11)

**Given** 완료 노드가 2회 이상 재실행되어 `attemptHistory`에 이력이 있다.  
**When** 이전 결과 패널이 열린다.  
**Then** 패널 헤더에 "Attempt #N" 배지가 표시된다.  
**And** 패널 내에 "📜 이전 시도 N건" 토글 섹션이 있다.  
**And** 토글 클릭 시 각 이전 attempt의 출력 요약이 표시된다.

---

## AC-12 — 마지막 작업 상태 복원 (REQ-12)

**Given** 이전 세션에서 노드를 배치·연결하고 일부 에이전트를 실행한 뒤 Studio를 종료했다.  
**When** Studio를 재시작한다.  
**Then** 이전 노드 배치와 연결선이 캔버스에 복원된다.  
**And** 이전에 `completed` 상태였던 노드는 그대로 `completed`로 표시된다.  
**And** 이전에 `active`(실행 중) 상태였던 노드는 `interrupted`로 표시된다(주황색 테두리, opacity 0.75).  
**And** 어떤 노드도 자동으로 재실행되지 않는다.  
**And** `loadState()` 실패 시 터미널에 "이전 상태를 불러오지 못했습니다. 새로 시작합니다." 메시지가 표시된다.

**Test command**: `go build ./... && ./AutopusStudio-new.exe ui --port 8082`, 브라우저에서 http://localhost:8082 열기 후 노드 배치·연결 → Studio 재시작 → 노드 위치·상태 확인.

---

## AC-13 — 기본 진입 화면 Builder (REQ-13)

**Given** Studio가 최초 실행되거나 `localStorage['autopus-mode']`가 없는 상태이다.  
**When** http://localhost:8082 를 브라우저에서 연다.  
**Then** 빌더(Builder) 화면이 기본으로 표시된다.  
**And** "빌더" 버튼이 active 상태(강조)로 표시된다.  
**And** 모니터링 오버레이는 보이지 않는다.

**Given** 이전 세션에서 사용자가 모니터링 탭을 마지막으로 선택했다(`localStorage['autopus-mode'] === 'monitor'`).  
**When** Studio를 재시작한다.  
**Then** 모니터링 화면이 복원된다.  
**And** 빌더로 전환 시 모든 기능이 정상 동작한다.

**Test command**: 브라우저 DevTools → Application → localStorage에서 `autopus-mode` 값 확인; 키 삭제 후 새로고침 시 빌더가 기본 진입 화면인지 확인.
