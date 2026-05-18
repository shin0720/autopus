# SPEC-UIRESTORE-001 — 구현 계획

**수정 대상**: `content/ui/dashboard.html` (단일 파일)  
**현재 줄 수**: 2149줄 (커밋 b2a81d8 기준)  
**금지**: 전체 재작성 / git add -A / push / 다른 파일 수정

---

## 구현 우선순위

| 순서 | REQ | 기능 | 줄 수 영향 | 위험도 | 선행 조건 |
|------|-----|------|-----------|--------|-----------|
| P1 | REQ-02 | 노드 클릭 → 메시지창 우선 | ~5줄 수정 | 중 | 없음 |
| P2 | REQ-01 | view-footer 버튼 완성 | +20줄 | 중 | REQ-02 |
| P3 | REQ-03 | drawer 취소 버튼 위치 | ~10줄 수정 | 저 | 없음 |
| P4 | REQ-08 | 추가질문 view 연동 | ~15줄 수정 | 중 | REQ-01 |
| P5 | REQ-07 | 시스템 프롬프트 localStorage | ~20줄 추가 | 저 | 없음 |
| P6 | REQ-11 | attempt 이력 표시 | ~30줄 추가 | 저 | 없음 |
| P7 | REQ-05 | 파일 유효성 검증 | ~25줄 추가 | 저 | 없음 |
| P8 | REQ-06 | 연결선 × 삭제 UI | ~20줄 수정 | 저 | 없음 |
| P9 | REQ-09 | DevOps provider 강제 | ~15줄 추가 | 저 | 없음 |
| P10 | REQ-04 | 자동 진행 흐름 | ~80줄 추가 | 고 | REQ-01, REQ-02 |
| P11 | REQ-10 | 완료창 | ~60줄 추가 | 고 | REQ-04 |

---

## Phase 1 — 노드 클릭 + view-footer (REQ-01, REQ-02)

### Task 1.1 — 완료 노드 클릭 → drawer 열기

**수정 위치**: `el.onclick` 핸들러 (L1302-1313)

```javascript
// Before
} else if (nodeState.status === 'completed') {
    viewCompletedOutput(agent.id);
}

// After
} else if (nodeState.status === 'completed') {
    openDrawer(agent, nodeState.lastPrompt || '');
}
```

**함수 `openDrawer()` 수정** (L1700+):
완료 상태 노드면 `drawer-output-btn-area`에 "📋 이전 결과 보기" 버튼 추가.

```javascript
if (nodeState.status === 'completed' && nodeState.output) {
    const btn = document.createElement('button');
    btn.className = 'btn-run';
    btn.style.cssText = 'width:100%; margin-bottom:8px; background:var(--bg3); color:var(--fg); border:1px solid var(--border);';
    btn.textContent = '📋 이전 결과 보기';
    btn.onclick = () => viewCompletedOutput(agent.id);
    outputBtnArea.style.display = 'block';
    outputBtnArea.appendChild(btn);
}
```

### Task 1.2 — view-footer 버튼 추가

**수정 위치**: L626-631

승인/취소/재분석 버튼 추가. 새 함수 `approveAndProceed()` 추가.

```javascript
async function approveAndProceed() {
    const agentId = viewingCompletedAgentId;
    if (!agentId) return;
    const nodeState = getNodeState(agentId);
    if (nodeState.output) {
        if (!nodeState.attemptHistory) nodeState.attemptHistory = [];
        nodeState.attemptHistory.push({ attempt: nodeState.attemptHistory.length + 1, output: nodeState.output, timestamp: new Date().toISOString(), type: 'approved' });
    }
    const next = workflowState.connections.find(c => c.from === agentId);
    if (!next) { alert('연결된 다음 에이전트가 없습니다.'); return; }
    closePanel();
    // 다음 노드 핸드오프 실행 (resumeFromCompletedNode와 유사하나 승인 의미)
    await resumeFromCompletedNode();
}
```

---

## Phase 2 — drawer 위치 + 추가질문 (REQ-03, REQ-08)

### Task 2.1 — drawer CSS 수정

```css
.sidebar-right {
    max-height: calc(100vh - 56px);
    overflow-y: auto;
}
/* 취소 버튼 고정 */
.sidebar-right .drawer-action-footer {
    position: sticky;
    bottom: 0;
    background: var(--bg2);
    padding: 8px 0 0;
    margin-top: 8px;
}
```

취소 버튼 div를 `.drawer-action-footer` 클래스 div로 감싸기.

### Task 2.2 — followup-footer view 연동

**수정 위치**: `viewCompletedOutput()` (L1152 이후)

```javascript
// After: document.getElementById('view-footer').style.display = 'flex';
document.getElementById('followup-footer').style.display = 'block';
```

**수정 위치**: `closePanel()` (L1800)

```javascript
document.getElementById('followup-footer').style.display = 'none';
```

**수정 위치**: `sendFollowUp()` (L1245) — 결과 구분 렌더링

```javascript
// 기존 panel-content에 구분선 + 헤더 추가 후 답변 append
const divider = document.createElement('hr');
const header = document.createElement('div');
header.style.cssText = 'color:var(--accent); font-weight:bold; margin: 12px 0 6px;';
header.textContent = '💬 추가질문 답변';
panelContent.appendChild(divider);
panelContent.appendChild(header);
```

---

## Phase 3 — 시스템 프롬프트 + attempt 이력 (REQ-07, REQ-11)

### Task 3.1 — 시스템 프롬프트 localStorage

**수정 위치**: `saveSysPromptInMap()` 함수 확인 후 localStorage 추가  
**수정 위치**: `openDrawer()` 내 sys-prompt 복원 로직

### Task 3.2 — attempt 이력 UI

**수정 위치**: `viewCompletedOutput()` — 패널 헤더 + 이력 섹션

```javascript
const attemptNum = (nodeState.attemptHistory || []).length + 1;
// panel title에 "Attempt #N" 추가
const title = (agent ? agent.name : agentId) + ` 이전 결과 보기 — Attempt #${attemptNum}`;
// 하단에 이력 토글 섹션 추가
```

---

## Phase 4 — 파일 유효성 + 연결선 UX (REQ-05, REQ-06)

### Task 4.1 — 파일 유효성 검증

```javascript
async function validateAgentFiles(agentId) {
    const res = await fetch('/api/files/list');
    const data = await res.json();
    const existingFiles = new Set(data.files || []);
    const fileSet = agentFiles.get(agentId) || new Set();
    const missing = [];
    fileSet.forEach(f => { if (!existingFiles.has(f)) missing.push(f); });
    return { missing, valid: [...fileSet].filter(f => existingFiles.has(f)) };
}
```

### Task 4.2 — 연결선 × 삭제 버튼

**수정 위치**: `drawConnections()` (L1609-1631)

path의 중간 좌표에 delete-hint div를 SVG 위에 절대좌표로 렌더링.
또는 path의 `onclick`에 confirm 추가:

```javascript
path.onclick = () => {
    if (confirm('이 연결을 삭제하시겠습니까?')) {
        workflowState.connections.splice(index, 1);
        drawConnections();
        saveState();
    }
};
```

---

## Phase 5 — DevOps provider + 완료창 + 자동 진행 (REQ-09, REQ-10, REQ-04)

### Task 5.1 — DevOps provider 강제

**수정 위치**: `renderProviderCheckboxes(agentId)` 함수

### Task 5.2 — 완료창

**추가 HTML**: `<div id="completion-dialog">` (body 또는 overlay 위)  
**추가 함수**: `showCompletionDialog()`, `closeCompletionDialog()`, `restartPipeline()`  
**트리거**: SSE 완료 이벤트 + agentId === 'devops'

### Task 5.3 — 자동 진행

**추가 함수**: `startAutoFlow()`, `pauseAutoFlow()`, `rollbackToPrevNode(agentId)`  
**UI**: 상단 툴바에 "▶ 자동 진행", "⏸ 일시 정지" 버튼 추가

---

## 파일 크기 관리

현재 2149줄. 구현 완료 후 예상 줄 수:

| Phase | 추가/수정 줄 | 누적 예상 |
|-------|-------------|-----------|
| Phase 1 | +40 | ~2189 |
| Phase 2 | +30 | ~2219 |
| Phase 3 | +35 | ~2254 |
| Phase 4 | +40 | ~2294 |
| Phase 5 | +160 | ~2454 |

**주의**: `*.html` 파일은 file-size-limit 제외 대상. 단, JS 함수별 가독성 유지.

---

## 커밋 전략

각 Phase 완료 후 별도 커밋:

| 커밋 | 내용 |
|------|------|
| `fix(ui): restore node-click-to-drawer + view-footer approve/cancel buttons` | Phase 1 |
| `fix(ui): fix drawer scroll, followup-footer view mode` | Phase 2 |
| `feat(ui): persist sys-prompt to localStorage, show attempt history` | Phase 3 |
| `fix(ui): validate agent files, add connection delete confirm` | Phase 4 |
| `feat(ui): devops provider lock, completion dialog, auto-flow` | Phase 5 |

각 커밋은 Lore 형식 준수 (`🐙 Autopus <sinmihyeon@gmail.com>`).

---

## Phase 6 — REQ-12: 마지막 작업 상태 복원 (✅ IMPLEMENTED 2026-05-18)

### Task 6-1 — `loadState()` try/catch + active→interrupted 변환

**수정 파일**: `content/ui/dashboard.html`

```javascript
async function loadState() {
    try {
        const res = await fetch('/api/workflow/state');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        // ... 기존 상태 복원 ...
        workflowState.nodes.forEach((node) => {
            if (node.status === 'active') node.status = 'interrupted';
        });
        // ... initNodes, drawConnections, renderTerminalLogs ...
    } catch (err) {
        appendTerminalLog('System', `이전 상태를 불러오지 못했습니다. 새로 시작합니다. (${err.message})`);
        workflowState.nodes = [];
        // ...
    }
}
```

### Task 6-2 — `interrupted` 상태 CSS 및 JS 지원

- CSS `.node.interrupted { border-color:#f0a500; border-width:2px; opacity:0.75; }` 추가
- CSS `.mon-node.s-interrupted .mon-status { color:#f0a500; }` 추가
- `nodeStatusClasses`에 `'interrupted'` 추가
- `getMonitorStatus()` switch에 `'interrupted'` 케이스 추가

---

## Phase 7 — REQ-13: 기본 진입 화면 Builder (✅ IMPLEMENTED 2026-05-18)

### Task 7-1 — HTML 기본 active 상태 수정

**수정 파일**: `content/ui/dashboard.html`

| 위치 | 변경 전 | 변경 후 |
|------|---------|---------|
| `mode-builder` 버튼 | `class=""` | `class="active"` |
| `mode-monitor` 버튼 | `class="active"` | `class=""` |
| `zoom-control` div | `style="opacity: 0.4;"` | `style="opacity: 1;"` |
| `monitor-view` div | `class="monitor-view active"` | `class="monitor-view"` |

DOMContentLoaded는 savedMode='monitor'이면 `switchMode('monitor')` 호출, 그 외 변경 없음(builder 기본).

---

## 테스트 방법

1. `./AutopusStudio.exe` 또는 `go run ./cmd/studio` 실행
2. 브라우저에서 `http://localhost:PORT` 접속
3. AC-01 ~ AC-13 수락 기준 순서대로 수동 검증
4. DevTools Console에서 JavaScript 오류 없는지 확인
5. 1366×768 해상도에서 drawer 취소 버튼 가시성 확인 (AC-03)
6. REQ-13: localStorage에서 `autopus-mode` 키 삭제 후 새로고침 → Builder가 기본임을 확인
7. REQ-12: 노드 배치·실행 중 Studio 재시작 → 위치 복원 + running 노드 interrupted 표시 확인

---

## 완료 판정 기준

- [x] AC-01 ~ AC-11 기준 구현 완료 (Part 1 — Node Click UX 오버홀)
- [x] AC-12 — 마지막 작업 상태 복원 구현 완료 (REQ-12)
- [x] AC-13 — 기본 진입 화면 Builder 구현 완료 (REQ-13)
- [x] `go build ./...` 성공
- [ ] Dashboard HTML에서 JavaScript 콘솔 오류 없음 (브라우저에서 직접 확인 필요)
- [ ] 기존 기능 회귀 없음 (노드 드래그, 연결, 승인 패널, 터미널 로그)
