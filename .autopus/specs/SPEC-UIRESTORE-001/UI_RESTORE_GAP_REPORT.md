# SPEC-UIRESTORE-001 — GAP 보고서

**기준 파일**: `content/ui/dashboard.html` (커밋 b2a81d8, 2149줄)  
**분석 일자**: 2026-05-18  

---

## GAP 분석 요약

| REQ | 기능 | 현재 상태 | GAP | 위험도 |
|-----|------|-----------|-----|--------|
| REQ-01 | view-footer 버튼 완성 | 부분 구현 | 승인·취소·재분석 버튼 없음 | 중 |
| REQ-02 | 노드 클릭 → 메시지창 우선 | 미구현 | completed 클릭 시 결과 패널 직접 열림 | 중 |
| REQ-03 | drawer 취소 버튼 위치 | 미구현 | 작업표시줄 겹침 가능 | 중 |
| REQ-04 | 자동 진행 흐름 | 미구현 | 수동 실행만 가능 | 고 |
| REQ-05 | 파일 목록 유효성 | 미구현 | stale 파일 표시 가능 | 저 |
| REQ-06 | 연결선 UX | 부분 구현 | hover 삭제 UI 없음 | 저 |
| REQ-07 | 시스템 프롬프트 저장 | 부분 구현 | localStorage 미사용, 새로고침 시 소실 | 저 |
| REQ-08 | 추가질문 view 연동 | 미구현 | approval 패널에서만 작동 | 중 |
| REQ-09 | DevOps provider 강제 | 미구현 | 자유 선택 가능 | 저 |
| REQ-10 | 완료창 | 미구현 | 없음 | 고 |
| REQ-11 | attempt 이력 표시 | 미구현 | 저장만 됨, 표시 없음 | 저 |

---

## 상세 GAP

### REQ-01 — view-footer 버튼

**현재 HTML** (L626-631):
```html
<div id="view-footer" ...>
    <button onclick="rerunCompletedNode()">✏️ 수정 재실행</button>
    <button id="btn-rerun-direct" onclick="rerunCompletedNodeDirect()">🔄 이 작업자 다시 실행</button>
    <button id="btn-resume-from" onclick="resumeFromCompletedNode()">⏩ 여기서부터 다시 진행</button>
    <button onclick="closePanel()">닫기</button>
</div>
```

**누락 버튼**: 
- `✅ 승인 / 다음으로 진행` → `approveAndProceed()` [NEW 함수]
- `❌ 취소` (현재 "닫기"가 있으나 명칭·역할 명확화 필요)
- `🔄 수정 후 재분석` → followup-footer 토글

---

### REQ-02 — 노드 클릭 동작

**현재 코드** (L1305-1312):
```javascript
if (nodeState.status === 'error') {
    openDrawer(agent, nodeState.lastPrompt || '');
} else if (nodeState.status === 'completed') {
    viewCompletedOutput(agent.id);   // ← 결과 직접 열림
} else {
    selectNode(agent);
}
```

**목표 코드**:
```javascript
} else if (nodeState.status === 'completed') {
    openDrawer(agent, nodeState.lastPrompt || '');  // ← drawer 먼저
    // drawer 내 btn-view-output 버튼이 viewCompletedOutput 호출
}
```

**drawer-output-btn-area** (L651): 이미 존재하며 `openDrawer()`에서 버튼을 동적으로 주입하는 구조(L1737-1734).
완료 노드 클릭 시 이 영역에 "📋 이전 결과 보기" 버튼을 렌더링하면 됨.

---

### REQ-03 — drawer 취소 버튼

**현재 CSS**: sidebar-right에 overflow 관련 고정 높이 없음. 취소 버튼(L669)이 컨텐츠 맨 아래 위치.

**문제**: Windows 화면 해상도 낮거나 작업표시줄 크기에 따라 스크롤 없이 버튼 접근 불가.

**수정 방향**:
```css
.sidebar-right {
    max-height: calc(100vh - 48px); /* 작업표시줄 48px 여유 */
    overflow-y: auto;
    padding-bottom: 16px;
}
```
취소 버튼을 `position: sticky; bottom: 0; background: var(--bg2);` 처리.

---

### REQ-04 — 자동 진행

**현재 상태**: 자동 진행 트리거 없음. `workflowState`에 autoFlow 관련 필드 없음.

**필요 로직**:
1. `startAutoFlow()` — in-degree 0인 노드 찾기 → `runAgent()` 호출
2. SSE 완료 이벤트 수신 시 다음 연결 노드 자동 실행
3. `pauseAutoFlow()` — `workflowState.autoFlowPaused = true` 설정
4. `rollbackToPrevNode(agentId)` — `workflowState.connections.find(c => c.to === agentId)`로 prev 찾기 → prev 노드 상태 초기화, attemptHistory 보존

**변경 파일**: dashboard.html 1개 (JS 추가)

---

### REQ-05 — 파일 유효성

**현재 코드** (L696):
```javascript
try { const _af = sessionStorage.getItem('agentFiles'); 
      if (_af) JSON.parse(_af).forEach(([k,v]) => agentFiles.set(k, new Set(v))); } catch(_) {}
```

**문제**: 세션스토리지에서 복원 시 실제 파일 존재 여부 미검증.

**수정 방향**: `openDrawer()` 또는 별도 `validateAgentFiles(agentId)` 함수에서
`/api/files/list` 응답과 교차하여 존재하지 않는 파일 제거/배지 표시.

---

### REQ-06 — 연결선 UX

**현재 CSS** (L132-133):
```css
.flow-line { stroke-dasharray:6; stroke-width:2.5; opacity:.7; }
.flow-line:hover { stroke:var(--err); stroke-width:4; opacity:1; }
```

**현재 삭제**: 클릭 시 즉시 삭제 (L1628). × 버튼 UI 없음.

**수정 방향**: `drawConnections()`에서 각 path 옆에 SVG `<text>` 또는 foreignObject로 "×" 표시.
또는 연결선 중간 좌표에 절대좌표 div를 동적 생성.

---

### REQ-07 — 시스템 프롬프트

**현재 코드**: `oninput="saveSysPromptInMap()"` — Map에만 저장.

**문제**: 페이지 새로고침 시 Map 초기화 → 프롬프트 소실.

**수정 방향**:
```javascript
function saveSysPromptInMap() {
    const agentId = selectedAgent?.id;
    if (!agentId) return;
    const val = document.getElementById('sys-prompt-input').value;
    systemPrompts.set(agentId, val);
    localStorage.setItem('sys-prompt-' + agentId, val); // [NEW]
}
```
`openDrawer()`에서 localStorage 복원 추가.

---

### REQ-08 — 추가질문 view 연동

**현재 코드** (L1794-1795):
```javascript
document.getElementById('approval-footer').style.display = hasPending ? 'flex' : 'none';
document.getElementById('followup-footer').style.display = hasPending ? 'block' : 'none';
```

`viewCompletedOutput()` (L1152)에서 followup-footer를 표시하지 않음.

**수정 방향**: `viewCompletedOutput()` 내에 `followup-footer` 표시 추가.
`sendFollowUp()` 결과 렌더링 시 구분 헤더 삽입.

---

### REQ-09 — DevOps provider 강제

**현재 코드**: provider 선택이 agentId와 무관하게 자유로움.

**수정 방향**: `renderProviderCheckboxes(agentId)` 내에:
```javascript
if (agentId === 'devops') {
    // codex만 체크, 나머지 disabled
}
```

---

### REQ-10 — 완료창

**현재 상태**: 없음. SSE completion 이벤트 수신 로직에 DevOps 분기 없음.

**수정 방향**: SSE `type === 'complete'` && `agentId === 'devops'` → `showCompletionDialog()` 호출.
완료창 HTML 구조:
```html
<div id="completion-dialog" style="display:none; position:fixed; ...">
    <h2>🎉 파이프라인 완료</h2>
    <section id="completion-summary">...</section>
    <section id="completion-files">...</section>
    <div class="completion-footer">
        <button onclick="restartPipeline()">🔄 다시 실행</button>
        <button onclick="closeCompletionDialog()">✕ 닫기</button>
    </div>
</div>
```

---

### REQ-11 — attempt 이력 표시

**현재 상태**: `nodeState.attemptHistory[]` 배열이 저장되나 패널에 표시 안 됨.

**수정 방향**: `viewCompletedOutput()` 내에서:
```javascript
const attempts = nodeState.attemptHistory || [];
if (attempts.length > 0) {
    // 패널 하단에 "📜 이전 시도 N건" 접을 수 있는 섹션 추가
}
```

---

### REQ-12 — 마지막 작업 상태 복원

**현재 상태**: `loadState()`가 `changeWorkspace()` 경유로 호출되므로 재시작 시 상태 복원은 됨. 그러나 `active` 상태 노드가 실행 중인 것처럼 남아 있고, `loadState()` 실패 시 사용자 피드백 없음.

**GAP 항목**:
1. `active` 노드 → `interrupted` 변환 없음 (`loadState()` 내부)
2. `loadState()` try/catch 미설정 — 실패 무성
3. `interrupted` CSS 클래스 미정의 (`.node.interrupted`, `.mon-node.s-interrupted`)
4. `nodeStatusClasses`에 `'interrupted'` 미포함
5. `getMonitorStatus()`에 `'interrupted'` 케이스 미처리

**수정 방향**:
- `loadState()` try/catch 래핑 + 실패 시 `appendTerminalLog` 안내 출력
- 노드 배열 순회 후 `node.status === 'active'`이면 `'interrupted'`로 교체
- CSS: `.node.interrupted { border-color:#f0a500; border-width:2px; opacity:0.75; }` 추가
- CSS: `.mon-node.s-interrupted .mon-status { color:#f0a500; }` 추가
- `nodeStatusClasses`에 `'interrupted'` 추가
- `getMonitorStatus()` switch에 `'interrupted': { cls:'s-interrupted', label:'Interrupted' }` 추가

**구현 상태**: ✅ IMPLEMENTED (2026-05-18)

---

### REQ-13 — 기본 진입 화면 Builder로 변경

**현재 상태**:
- HTML L346: `mode-builder` 버튼 `class=""`
- HTML L347: `mode-monitor` 버튼 `class="active"` → 모니터링이 기본 강조
- HTML L351: `zoom-control` `style="opacity: 0.4;"` → 모니터 모드 opacity 값으로 초기화
- HTML L375: `monitor-view` div `class="monitor-view active"` → 모니터 오버레이가 페이지 로드 시 visible
- DOMContentLoaded: savedMode='monitor'일 때만 `switchMode('monitor')` 호출, 기본이 builder임에도 HTML은 monitor 상태

**GAP 항목**:
1. HTML 기본 active 상태가 `mode-monitor`로 설정됨
2. `monitor-view` div가 `active` class를 초기부터 가져 빌더 캔버스를 가림
3. `zoom-control` 초기 opacity가 0.4 (모니터 모드값)
4. DOMContentLoaded이 builder를 명시적으로 초기화하지 않음

**수정 방향**:
- HTML: `mode-builder` → `class="active"`, `mode-monitor` → `class=""`
- HTML: `zoom-control` → `style="opacity: 1;"`
- HTML: `monitor-view` → `class="monitor-view"` (active 제거)
- DOMContentLoaded: savedMode='monitor'이면 300ms 후 `switchMode('monitor')`, 그 외 기본 builder 유지

**구현 상태**: ✅ IMPLEMENTED (2026-05-18)

---

## Self-Verify Summary

| Q-ID | 상태 | 근거 |
|------|------|------|
| Q-CORR-01 | PASS | L626-631, L1143, L1245, L1609 등 실제 확인됨 |
| Q-CORR-02 | PASS | [NEW] 함수 모두 마킹됨; REQ-12/13 수정 대상도 실제 코드 확인 완료 |
| Q-COMP-01 | PASS | 4개 문서 전체 작성 |
| Q-COMP-02 | PASS | REQ→GAP→AC→plan 추적 가능 |
| Q-FEAS-01 | PASS | dashboard.html 단일 파일 변경으로 한정 |
| Q-FEAS-02 | PASS | content/ui 모듈 경계 내 |
| Q-SEC-01 | N/A | 로컬 HTML SPA, 외부 입력 경로 없음 |
| Q-SEC-02 | N/A | 시크릿/credential 없음 |
