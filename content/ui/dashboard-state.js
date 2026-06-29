        let selectedAgent = null;
        let isConnecting = false;
        let tempLineFrom = null;
        let lastResult = null;
        let lastStatePath = '';
        let pipelineAutoMode = JSON.parse(localStorage.getItem('autopus-pipeline-mode') ?? 'true');
        let panelAgentId = null;
        let panelResult = null;
        let currentPreviewFile = null;
        let sseController = null;
        let sseRetryDelay = 2000;
        let sseRetrying = false;
        const nodeStartTimes = new Map(); // agentId → startTimestamp ms
        let nodeElapsedInterval = null;
        const chunkQueues = new Map(); // agentId → {wrapper, pre, lines, index, timer}
        const agentFiles = new Map(); // agentId → Set of file paths
        const fileCache = new Map(); // path → mod timestamp (Unix sec), for N/M badge
        const seenEventIds = new Set();
        let lastPromptTokens = 0;
        let activeLogTab = 'all';
        const logAgents = new Set();
        const errorRecoveryMap = { 'exec': 'dbug', 'val': 'dbug', 'test': 'dbug' };
        let recoveryAttempts = 0;
        const MAX_RECOVERY = 2;
        // pendingResumption: when an agent reports "## 상류 부족 보고" we route to
        // the upstream agent instead of forward. This map remembers, for the
        // upstream agentId we just dispatched to, which downstream agent should
        // be re-triggered once the upstream completes.
        //   key   = upstream agentId (the one we sent the regression call to)
        //   value = { blockedAgent, missing, blockedHandoff, originalRequest, depth }
        const pendingResumption = new Map();
        const MAX_REGRESSION_DEPTH = 5;
        const nodeStatusClasses = ['active', 'completed', 'awaiting-approval', 'rejected', 'error'];
        const workflowState = { version: 2, nodes: [], connections: [], logs: [], approval: {} };
        let providerStatuses = [];
        const selectedProviders = new Map();
        const agentDefaultProviders = { devops: 'codex' };
        const systemPrompts = new Map();
        let zoomLevel = 1.0;
        let panX = 0, panY = 0, isPanning = false;
        try { const _af = sessionStorage.getItem('agentFiles'); if (_af) JSON.parse(_af).forEach(([k,v]) => agentFiles.set(k, new Set(v))); } catch(_) {}

        const agents = [
            { id: 'planner', name: 'Planner',     role: '기획자',   desc: '요구사항 분석 · 구현 계획 수립',        x: 40,  y: 30  },
            { id: 'spec',    name: 'Spec Writer',  role: '명세서',   desc: 'SPEC 4종 문서 생성 · 요구사항 구체화',  x: 210, y: 30  },
            { id: 'arch',    name: 'Architect',    role: '설계자',   desc: '시스템 아키텍처 설계 · 기술 결정',      x: 380, y: 30  },
            { id: 'expl',    name: 'Explorer',     role: '탐험가',   desc: '코드베이스 구조 탐색 · 파일 분석',      x: 550, y: 30  },
            { id: 'exec',    name: 'Executor',     role: '개발자',   desc: 'TDD 기반 코드 구현 · 기능 개발',        x: 40,  y: 170 },
            { id: 'deep',    name: 'Deep Worker',  role: '심층작업', desc: '장시간 복잡 작업 · 체크포인트 자율 수행', x: 210, y: 170 },
            { id: 'dbug',    name: 'Debugger',     role: '해결사',   desc: '버그 수정 · 근본 원인 분석',            x: 380, y: 170 },
            { id: 'anno',    name: 'Annotator',    role: '태그관리', desc: '@AX 태그 스캔 · 코드 주석 자동 적용',   x: 550, y: 170 },
            { id: 'test',    name: 'Tester',       role: '테스터',   desc: '단위 · 통합 · E2E 테스트 작성',         x: 40,  y: 310 },
            { id: 'val',     name: 'Validator',    role: '품질검증', desc: 'LSP · 린트 · 테스트 통과 여부 검증',    x: 210, y: 310 },
            { id: 'fend',    name: 'Frontend',     role: 'UI전문가', desc: 'Playwright E2E · UI 시각 회귀 감지',    x: 380, y: 310 },
            { id: 'uxv',     name: 'UX Validator', role: '시각검증', desc: '스크린샷 분석 · 레이아웃 오류 탐지',    x: 550, y: 310 },
            { id: 'perf',    name: 'Perf Eng',     role: '성능분석', desc: '벤치마크 실행 · 성능 회귀 탐지',        x: 40,  y: 450 },
            { id: 'rev',     name: 'Reviewer',     role: '리뷰어',   desc: '코드 리뷰 · 구조·보안 문제 탐지',       x: 210, y: 450 },
            { id: 'sec',     name: 'Security',     role: '보안감사', desc: 'OWASP 기준 취약점 감사',                x: 380, y: 450 },
            { id: 'devops',  name: 'DevOps',       role: '배포자',   desc: 'CI/CD · Docker · 배포 자동화',          x: 550, y: 450 }
        ];

        // ───── 모니터링 뷰: 단계 매핑 ─────
        const PHASES = [
            { num: '01', name: '기획',   ids: ['planner', 'spec', 'arch'] },
            { num: '02', name: '구현',   ids: ['expl', 'exec', 'deep'] },
            { num: '03', name: '검증',   ids: ['dbug', 'anno', 'test', 'val'] },
            { num: '04', name: '다듬기', ids: ['fend', 'uxv', 'perf'] },
            { num: '05', name: '배포',   ids: ['rev', 'sec', 'devops'] }
        ];

        const agentMap = Object.fromEntries(agents.map((agent) => [agent.id, agent]));

        function getNodeState(id) {
            let node = workflowState.nodes.find((entry) => entry.id === id);
            if (!node) {
                node = { id, status: '', x: '', y: '', output: null, rejectReason: '' };
                workflowState.nodes.push(node);
            }
            return node;
        }

        function updateActiveNodeTimers() {
            const now = Date.now();
            nodeStartTimes.forEach((start, id) => {
                const elapsed = Math.floor((now - start) / 1000);
                const min = Math.floor(elapsed / 60);
                const sec = elapsed % 60;
                const el = document.getElementById('elapsed-' + id);
                if (el) el.textContent = `⏳ ${min}:${sec.toString().padStart(2, '0')} 실행 중`;
            });
        }

        function setNodeStatus(id, status) {
            const node = getNodeState(id);
            node.status = status;
            const el = document.getElementById('node-' + id);
            if (el) {
                nodeStatusClasses.forEach((name) => el.classList.remove(name));
                if (status) el.classList.add(status);
            }
            if (status === 'active') {
                if (!nodeStartTimes.has(id)) nodeStartTimes.set(id, Date.now());
                if (!nodeElapsedInterval) nodeElapsedInterval = setInterval(updateActiveNodeTimers, 1000);
            } else {
                nodeStartTimes.delete(id);
                const elEl = document.getElementById('elapsed-' + id);
                if (elEl) elEl.textContent = '⏳ 0:00 실행 중';
                if (nodeStartTimes.size === 0 && nodeElapsedInterval) {
                    clearInterval(nodeElapsedInterval);
                    nodeElapsedInterval = null;
                }
            }
            if (typeof refreshMonitorIfActive === 'function') refreshMonitorIfActive();
        }

        function resetNodeError(agentId) {
            setNodeStatus(agentId, '');
            if (workflowState.approval.pendingNodeId === agentId) {
                workflowState.approval = {};
                lastResult = null;
            }
            saveState();
        }

        async function cancelAgent(agentId) {
            const agent = agentMap[agentId];
            const name = agent ? agent.name : agentId;
            try {
                const res = await fetch('/api/workflow/cancel', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ agentId })
                });
                const data = await res.json();
                if (data.status === 'not_found') {
                    appendTerminalLog('System', `[${name}] 실행 중인 작업이 없습니다.`);
                } else {
                    appendTerminalLog('System', `[${name}] ⛔ 작업을 취소했습니다.`);
                    setNodeStatus(agentId, '');
                    saveState();
                }
            } catch (e) {
                appendTerminalLog('System', `[${name}] 취소 요청 실패: ${e.message}`);
            }
        }

        function syncNodePositions() {
            workflowState.nodes = agents.map((agent) => {
                const current = getNodeState(agent.id);
                const el = document.getElementById('node-' + agent.id);
                return { ...current, id: agent.id, x: el ? el.style.left : (current.x || agent.x + 'px'), y: el ? el.style.top : (current.y || agent.y + 'px') };
            });
        }

