        function viewCompletedOutput(agentId) {
            const nodeState = getNodeState(agentId);
            if (!nodeState.output) { openDrawer(agentMap[agentId]); return; }
            const agent = agentMap[agentId];
            const hasNext = workflowState.connections.some(c => c.from === agentId);
            showPanel((agent ? agent.name : agentId) + ' 완료 결과', nodeState.output, null,
                { forwardFrom: hasNext ? agentId : null });
        }

        async function forwardCompletedToNext() {
            const forwardFooter = document.getElementById('forward-footer');
            const fromId = forwardFooter.dataset.from;
            if (!fromId) return;
            const next = workflowState.connections.find(c => c.from === fromId);
            if (!next) { appendTerminalLog('System', '연결된 다음 에이전트가 없습니다.'); return; }
            const nodeState = getNodeState(fromId);
            lastResult = nodeState.output;
            const currentAgent = agentMap[fromId];
            const nextAgent = agentMap[next.to];
            const originalRequest = getRootRequest(nodeState.originalRequest || nodeState.lastPrompt || '');
            const prevOutput = lastResult ? (lastResult.output || '') : '';
            const handoffPrompt = [
                `🚨 파이프라인 자동 실행. 아래 문장은 절대 출력 금지: "어떤 부분부터", "어느 것을 먼저", "확인해 드릴까요", "지원 가능한 작업 범위", 역할 소개 텍스트, 인프라 현황 설명 테이블. 텍스트 출력 이전에 반드시 Write 도구로 파일을 생성하세요. 파일 생성이 첫 번째 행동입니다.`,
                originalRequest ? `[사용자 원래 요청]\n${originalRequest}` : '',
                `[이전 단계 완료: ${currentAgent ? currentAgent.name : fromId}]`,
                prevOutput ? `이전 에이전트 작업 결과:\n${prevOutput}` : '',
                `[현재 역할: ${nextAgent ? nextAgent.name : next.to}]`,
                getAgentTaskInstruction(next.to, nextAgent),
                buildWiredUpstreamHint(next.to),
            ].filter(Boolean).join('\n\n');
            closePanel();
            appendTerminalLog('System', `➡️ ${currentAgent ? currentAgent.name : fromId} 결과를 ${nextAgent ? nextAgent.name : next.to}로 전달합니다.`);
            await runAgent(next.to, handoffPrompt, null, originalRequest);
        }

        async function saveFileEdit() {
            if (!currentPreviewFile) return;
            const ta = document.getElementById('file-edit-area');
            if (!ta) return;
            const res = await fetch('/api/files/write', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ filename: currentPreviewFile, content: ta.value })
            });
            if (res.ok) {
                appendTerminalLog('System', `💾 ${currentPreviewFile} 저장 완료`);
            } else {
                const msg = await res.text();
                alert('파일 저장 실패: ' + msg);
            }
        }


        async function saveApprovedDoc() {
            const resultToSave = panelResult || lastResult;
            if (!resultToSave) return;
            const agentId = panelAgentId || workflowState.approval.pendingNodeId || (resultToSave.fromAgent || 'result');
            const date = new Date().toISOString().slice(0, 10);
            const defaultName = `${agentId}-${date}.md`;
            const filename = prompt('저장할 파일명을 입력하세요:', defaultName);
            if (!filename) return;
            const content = resultToSave.output || '';
            try {
                const res = await fetch('/api/files/write', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ filename, content })
                });
                const data = await res.json();
                if (data.status === 'success') {
                    appendTerminalLog('System', `💾 저장 완료: ${data.path}`);
                } else {
                    appendTerminalLog('System', `저장 실패: ${data.message || '알 수 없는 오류'}`);
                }
            } catch(e) {
                appendTerminalLog('System', `저장 실패: ${e.message}`);
            }
        }

        function initNodes() {
            const container = document.getElementById('nodes-container');
            container.innerHTML = '';
            agents.forEach((agent) => {
                const node = getNodeState(agent.id);
                const el = document.createElement('div');
                el.className = 'node';
                el.id = 'node-' + agent.id;
                el.style.left = node.x || agent.x + 'px';
                el.style.top = node.y || agent.y + 'px';
                el.innerHTML = `<div class="node-name">${agent.name}</div><div class="node-role">${agent.role}</div><div class="node-desc">${agent.desc}</div><div class="node-elapsed" id="elapsed-${agent.id}">⏳ 0:00 실행 중</div><button class="node-reset-btn" onclick="event.stopPropagation();resetNodeError('${agent.id}')">× 에러 리셋</button><button class="node-view-btn" onclick="event.stopPropagation();reopenApprovalPanel('${agent.id}')">📋 결과 보기</button><button class="node-cancel-btn" onclick="event.stopPropagation();cancelAgent('${agent.id}')">⛔ 작업 취소</button><div class="port port-in" onmouseup="endConnection(event, '${agent.id}')"></div><div class="port port-out" onmousedown="startConnection(event, '${agent.id}')"></div>`;
                el.onmousedown = (event) => {
                    const t = event.target;
                    if (!t.classList.contains('port') && !t.classList.contains('node-reset-btn') && !t.classList.contains('node-view-btn') && !t.classList.contains('node-cancel-btn')) {
                        startDragging(event, el);
                    }
                };
                el.onclick = (event) => {
                    const t = event.target;
                    if (t.classList.contains('port') || t.classList.contains('node-reset-btn') || t.classList.contains('node-view-btn') || t.classList.contains('node-cancel-btn')) return;
                    const nodeState = getNodeState(agent.id);
                    if (nodeState.status === 'error') {
                        openDrawer(agent, nodeState.lastPrompt || '');
                    } else {
                        selectNode(agent);
                    }
                };
                el.ondblclick = (event) => {
                    if (event.target.classList.contains('port')) return;
                    const nodeState = getNodeState(agent.id);
                    if (nodeState.status === 'awaiting-approval' && nodeState.output) {
                        reopenApprovalPanel(agent.id);
                    } else {
                        openDrawer(agent, getNodeState(agent.id).lastPrompt || '');
                    }
                };
                container.appendChild(el);
                if (node.status) setNodeStatus(agent.id, node.status);
            });
        }

        function applyZoom() {
            const inner = document.getElementById('canvas-inner');
            if (inner) inner.style.transform = `translate(${panX}px,${panY}px) scale(${zoomLevel})`;
            const label = document.getElementById('zoom-label');
            if (label) label.textContent = Math.round(zoomLevel * 100) + '%';
            const area = document.querySelector('.canvas-area');
            if (area) {
                area.style.backgroundSize = (40 * zoomLevel) + 'px ' + (40 * zoomLevel) + 'px';
                area.style.backgroundPosition = panX + 'px ' + panY + 'px';
            }
        }
        function zoomIn() { zoomLevel = Math.min(Math.round((zoomLevel + 0.1) * 10) / 10, 3.0); applyZoom(); }
        function zoomOut() { zoomLevel = Math.max(Math.round((zoomLevel - 0.1) * 10) / 10, 0.1); applyZoom(); }
        function zoomReset() { zoomLevel = 1.0; panX = 0; panY = 0; applyZoom(); }

        // ═══════════════════════════════════════════════
        //  테마 토글 (다크 / 라이트)
        // ═══════════════════════════════════════════════
        function getInitialTheme() {
            try {
                const saved = localStorage.getItem('autopus-theme');
                if (saved === 'light' || saved === 'dark') return saved;
            } catch (_) {}
            if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) return 'light';
            return 'dark';
        }
        function applyTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            try { localStorage.setItem('autopus-theme', theme); } catch (_) {}
            setTimeout(() => {
                if (typeof drawConnections === 'function') drawConnections();
                if (currentMode === 'monitor' && typeof drawMonitorConnections === 'function') drawMonitorConnections();
            }, 280);
        }
        function toggleTheme() {
            const cur = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
            applyTheme(cur);
        }
        applyTheme(getInitialTheme());
        if (window.matchMedia) {
            const mq = window.matchMedia('(prefers-color-scheme: light)');
            const onMqChange = (e) => {
                try { if (localStorage.getItem('autopus-theme')) return; } catch (_) {}
                applyTheme(e.matches ? 'light' : 'dark');
            };
            if (mq.addEventListener) mq.addEventListener('change', onMqChange);
            else if (mq.addListener) mq.addListener(onMqChange);
        }

        // ═══════════════════════════════════════════════
        //  뷰 모드 토글 (빌더 / 모니터링)
        // ═══════════════════════════════════════════════
        let currentMode = 'builder';

