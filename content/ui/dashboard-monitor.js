        function switchMode(mode) {
            currentMode = mode;
            const monitor = document.getElementById('monitor-view');
            const builderBtn = document.getElementById('mode-builder');
            const monitorBtn = document.getElementById('mode-monitor');
            const zoomCtrl = document.getElementById('zoom-control');

            if (mode === 'monitor') {
                renderMonitorView();
                monitor.classList.add('active');
                monitorBtn.classList.add('active');
                builderBtn.classList.remove('active');
                if (zoomCtrl) zoomCtrl.style.opacity = '0.4';
            } else {
                monitor.classList.remove('active');
                builderBtn.classList.add('active');
                monitorBtn.classList.remove('active');
                if (zoomCtrl) zoomCtrl.style.opacity = '1';
            }
            try { localStorage.setItem('autopus-mode', mode); } catch (_) {}
        }

        function getMonitorStatus(builderStatus) {
            switch (builderStatus) {
                case 'active': return { cls: 's-active', label: 'Running' };
                case 'completed': return { cls: 's-completed', label: 'Done' };
                case 'awaiting-approval': return { cls: 's-awaiting', label: 'Awaiting' };
                case 'error': return { cls: 's-error', label: 'Error' };
                case 'rejected': return { cls: 's-rejected', label: 'Rejected' };
                default: return { cls: '', label: 'Idle' };
            }
        }

        function renderMonitorView() {
            const phasesEl = document.getElementById('mon-phases');
            if (!phasesEl) return;
            phasesEl.innerHTML = '';

            let doneCount = 0, activeCount = 0;
            let activeAgentName = null;

            PHASES.forEach((phase) => {
                let phaseHasActive = false;
                let phaseAllDone = true;

                const nodesHtml = phase.ids.map((agentId) => {
                    const agent = agentMap[agentId];
                    if (!agent) return '';
                    const node = workflowState.nodes.find((n) => n.id === agentId) || { status: '' };
                    const s = getMonitorStatus(node.status);

                    if (s.cls === 's-completed') doneCount++;
                    else if (s.cls === 's-active') {
                        activeCount++;
                        activeAgentName = agent.name;
                    }
                    if (s.cls === 's-active' || s.cls === 's-awaiting') phaseHasActive = true;
                    if (s.cls !== 's-completed') phaseAllDone = false;

                    const isAwaiting = node.status === 'awaiting-approval';
                    const action = isAwaiting
                        ? '<span class="mon-node-action cta">결과보기</span>'
                        : (s.cls === 's-completed' ? '<span class="mon-node-action">view</span>' : '');

                    return `
                        <div class="mon-node ${s.cls}" data-agent-id="${agent.id}" onclick="onMonitorNodeClick('${agent.id}')">
                            <span class="mon-port in"></span>
                            <span class="mon-port out"></span>
                            <div class="mon-node-title">${agent.name}</div>
                            <div class="mon-node-ko">${agent.role}</div>
                            <div class="mon-node-foot">
                                <span class="mon-status"><span class="dot"></span>${s.label}</span>
                                ${action}
                            </div>
                        </div>`;
                }).join('');

                const phaseClass = phaseHasActive ? 'active' : (phaseAllDone && phase.ids.length > 0 ? 'done' : '');

                phasesEl.insertAdjacentHTML('beforeend', `
                    <div class="mon-phase ${phaseClass}">
                        <div class="mon-phase-head">
                            <div class="mon-phase-marker">
                                <span class="mon-phase-num">${phase.num}</span>
                                <span class="mon-phase-bar"></span>
                            </div>
                            <div class="mon-phase-name">${phase.name}</div>
                        </div>
                        ${nodesHtml}
                    </div>`);
            });

            const idleCount = Math.max(0, agents.length - doneCount - activeCount);
            const setText = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val; };
            setText('mon-stat-done', doneCount);
            setText('mon-stat-active', activeCount);
            setText('mon-stat-idle', idleCount);

            const summary = activeAgentName
                ? `16개 에이전트의 협업 흐름 — 현재 <strong>${activeAgentName}</strong> 작업 중`
                : (doneCount === agents.length ? '모든 단계 완료' : '16개 에이전트의 협업 흐름');
            const sumEl = document.getElementById('mon-summary');
            if (sumEl) sumEl.innerHTML = summary;

            requestAnimationFrame(drawMonitorConnections);
        }

        function drawMonitorConnections() {
            const svg = document.getElementById('mon-connections');
            const wrap = document.querySelector('.mon-phases-wrap');
            if (!svg || !wrap) return;
            svg.innerHTML = '';
            const wrapRect = wrap.getBoundingClientRect();
            const NS = 'http://www.w3.org/2000/svg';

            const connections = workflowState.connections || [];
            if (connections.length === 0) return;

            const order = { idle: 0, done: 1, active: 2 };
            const ports = new Map();

            const links = connections.map(({ from, to }) => {
                const fromNode = workflowState.nodes.find((n) => n.id === from);
                const toNode = workflowState.nodes.find((n) => n.id === to);
                const fromStatus = fromNode ? fromNode.status : '';
                const toStatus = toNode ? toNode.status : '';
                let kind = 'idle';
                if (fromStatus === 'completed' && (toStatus === 'active' || toStatus === 'awaiting-approval')) kind = 'active';
                else if (fromStatus === 'completed' && toStatus === 'completed') kind = 'done';
                else if (fromStatus === 'completed') kind = 'active';
                return { from, to, kind };
            });

            links.sort((a, b) => order[a.kind] - order[b.kind]);

            function portXY(agentId, side) {
                const el = document.querySelector(`.mon-node[data-agent-id="${agentId}"]`);
                if (!el) return null;
                const r = el.getBoundingClientRect();
                const x = (side === 'out' ? r.right : r.left) - wrapRect.left;
                const y = r.top + r.height / 2 - wrapRect.top;
                return { x, y };
            }

            links.forEach(({ from, to, kind }) => {
                const p1 = portXY(from, 'out');
                const p2 = portXY(to, 'in');
                if (!p1 || !p2) return;
                const dx = (p2.x - p1.x) * 0.5;
                const path = document.createElementNS(NS, 'path');
                path.setAttribute('d',
                    `M ${p1.x} ${p1.y} C ${p1.x + dx} ${p1.y}, ${p2.x - dx} ${p2.y}, ${p2.x} ${p2.y}`);
                path.setAttribute('class', `mon-conn ${kind}`);
                svg.appendChild(path);

                const k1 = `${p1.x},${p1.y}`, k2 = `${p2.x},${p2.y}`;
                if (!ports.has(k1) || order[kind] > order[ports.get(k1)]) ports.set(k1, kind);
                if (!ports.has(k2) || order[kind] > order[ports.get(k2)]) ports.set(k2, kind);
            });

            ports.forEach((kind, key) => {
                const [x, y] = key.split(',').map(Number);
                const c = document.createElementNS(NS, 'circle');
                c.setAttribute('cx', x);
                c.setAttribute('cy', y);
                c.setAttribute('r', kind === 'active' ? 3 : 2);
                c.setAttribute('class', `mon-conn-port ${kind === 'active' ? 'active' : (kind === 'done' ? 'done' : '')}`);
                svg.appendChild(c);
            });
        }

        function onMonitorNodeClick(agentId) {
            const node = workflowState.nodes.find((n) => n.id === agentId);
            if (!node) return;
            if (node.status === 'awaiting-approval' && typeof reopenApprovalPanel === 'function') {
                reopenApprovalPanel(agentId);
            } else if (node.status === 'completed') {
                viewCompletedOutput(agentId);
            } else {
                openDrawer(agentMap[agentId]);
            }
        }

        function refreshMonitorIfActive() {
            if (currentMode === 'monitor') renderMonitorView();
        }

        window.addEventListener('resize', () => {
            if (currentMode === 'monitor') drawMonitorConnections();
        });

        // 페이지 로드 시 마지막 모드 복원 (모든 데이터 로드 후 약간 대기)
        window.addEventListener('DOMContentLoaded', () => {
            try {
                const savedMode = localStorage.getItem('autopus-mode');
                if (savedMode === 'monitor') setTimeout(() => switchMode('monitor'), 300);
            } catch (_) {}
            updatePipelineModeBtn();
        });

