        function showPipelineCompleteModal(lastAgentId, lastResult) {
            const existing = document.getElementById('pipelineCompleteModal');
            if (existing) existing.remove();
            const lastAgent = agentMap[lastAgentId];
            const lastName = lastAgent ? lastAgent.name : lastAgentId;
            const summary = (lastResult && (lastResult.summary || lastResult.output)) || '';
            const summaryShort = summary.length > 600 ? summary.slice(0, 600) + '\n...(이하 생략 — 마지막 작업자 결과 카드 참고)' : summary;
            const completedAgents = workflowState.nodes
                .filter(n => n.status === 'completed' || n.status === 'awaiting-approval')
                .map(n => agentMap[n.id] ? agentMap[n.id].name : n.id);
            const modal = document.createElement('div');
            modal.id = 'pipelineCompleteModal';
            modal.style.cssText = 'position:fixed;inset:0;z-index:9999;background:rgba(0,0,0,0.55);display:flex;align-items:center;justify-content:center;padding:24px;';
            modal.innerHTML = `
                <div style="background:#1f2937;color:#f3f4f6;border-radius:14px;max-width:720px;width:100%;max-height:80vh;overflow:auto;box-shadow:0 24px 60px rgba(0,0,0,0.5);">
                    <div style="padding:24px 28px 16px;border-bottom:1px solid #374151;">
                        <div style="font-size:28px;font-weight:700;color:#22d3ee;">🎉 전체 파이프라인 완료</div>
                        <div style="margin-top:8px;color:#9ca3af;font-size:14px;">마지막 작업자: <b style="color:#f3f4f6;">${lastName}</b></div>
                    </div>
                    <div style="padding:20px 28px;">
                        <div style="font-size:13px;color:#9ca3af;margin-bottom:8px;">완료된 작업자 (${completedAgents.length}명)</div>
                        <div style="display:flex;flex-wrap:wrap;gap:6px;margin-bottom:18px;">
                            ${completedAgents.map(n => `<span style="background:#064e3b;color:#a7f3d0;padding:4px 10px;border-radius:999px;font-size:12px;">✓ ${n}</span>`).join('')}
                        </div>
                        <div style="font-size:13px;color:#9ca3af;margin-bottom:8px;">마지막 작업자 결과 요약</div>
                        <pre style="background:#111827;border:1px solid #374151;border-radius:8px;padding:14px;white-space:pre-wrap;font-size:13px;line-height:1.6;max-height:300px;overflow:auto;">${summaryShort.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}</pre>
                        <div style="margin-top:16px;font-size:12px;color:#9ca3af;line-height:1.6;">
                            각 작업자의 상세 결과는 캔버스의 해당 노드를 클릭해서 확인할 수 있습니다.<br>
                            전체 출력 파일은 <code style="background:#111827;padding:2px 6px;border-radius:4px;">.autopus/results/</code> 폴더에 저장돼 있습니다.
                        </div>
                    </div>
                    <div style="padding:16px 28px 24px;display:flex;justify-content:flex-end;gap:8px;border-top:1px solid #374151;">
                        <button id="pipelineCompleteCloseBtn" style="background:#22d3ee;color:#0f172a;border:none;padding:10px 22px;border-radius:8px;font-weight:600;cursor:pointer;font-size:14px;">확인</button>
                    </div>
                </div>`;
            document.body.appendChild(modal);
            document.getElementById('pipelineCompleteCloseBtn').onclick = () => modal.remove();
            modal.onclick = (e) => { if (e.target === modal) modal.remove(); };
        }

        // routeToUpstream sends the named upstream agent a regression-context
        // prompt and records that the currently blocked agent should re-run
        // when the upstream finishes.
        //
        // Bound to the user's wiring: regression can ONLY follow the existing
        // connection graph in reverse. If the named upstream isn't reachable
        // backward from the blocked agent through user-drawn connections, we
        // refuse to route and mark the blocked agent as error so the user can
        // either wire it up or handle manually.
        async function routeToUpstream(blockedAgentId, blockedResult, report) {
            const upstreamId = report.upstreamId;

            // Verify the named upstream is part of the user's wired pipeline.
            if (!isReachableUpstream(blockedAgentId, upstreamId)) {
                const wiredList = listWiredUpstreams(blockedAgentId);
                const wiredNames = wiredList.length
                    ? wiredList.map((id) => (agentMap[id] ? agentMap[id].name : id)).join(', ')
                    : '(연결된 상류 없음)';
                appendTerminalLog('System',
                    `⚠️ ${agentMap[blockedAgentId] ? agentMap[blockedAgentId].name : blockedAgentId} 가 ${upstreamId} 로 상류 회귀를 요청했지만, 현재 그래프에 그 연결선이 없습니다.\n` +
                    `   현재 ${blockedAgentId} 의 연결된 상류: ${wiredNames}\n` +
                    `   필요하면 캔버스에서 ${upstreamId} → ${blockedAgentId} 연결선을 그어주세요. 회귀 라우팅을 보류합니다.`);
                setNodeStatus(blockedAgentId, 'error');
                return;
            }

            // Prevent infinite ping-pong.
            let depth = 1;
            for (const entry of pendingResumption.values()) {
                if (entry.blockedAgent === upstreamId) depth = Math.max(depth, entry.depth + 1);
            }
            if (depth > MAX_REGRESSION_DEPTH) {
                appendTerminalLog('System', `⚠️ 상류 회귀 깊이 한계(${MAX_REGRESSION_DEPTH}) 도달. ${blockedAgentId} 가 계속 ${upstreamId} 를 요청합니다. 사용자 개입 필요.`);
                setNodeStatus(blockedAgentId, 'error');
                return;
            }

            const blockedAgent = agentMap[blockedAgentId];
            const upstreamAgent = agentMap[upstreamId];
            const blockedNodeState = getNodeState(blockedAgentId);
            const originalRequest = getRootRequest(blockedNodeState.originalRequest || blockedNodeState.lastPrompt || '');
            const prevOutput = (blockedResult && blockedResult.output) ? blockedResult.output : '';

            // Mark blocked node visually so user sees what's happening.
            setNodeStatus(blockedAgentId, 'awaiting-approval');
            appendTerminalLog('System',
                `↩️ 상류 회귀: ${blockedAgent ? blockedAgent.name : blockedAgentId} 가 ${upstreamAgent ? upstreamAgent.name : upstreamId} 에게 사전 작업을 요청합니다.\n사유: ${report.missing}`);

            pendingResumption.set(upstreamId, {
                blockedAgent: blockedAgentId,
                missing: report.missing,
                blockedHandoff: prevOutput,
                originalRequest,
                depth,
            });

            const upstreamPrompt = [
                `🚨 상류 회귀 호출. 하류 작업자 ${blockedAgent ? blockedAgent.name : blockedAgentId} 가 본인 일을 진행할 수 없어서 당신에게 거꾸로 돌아왔습니다.`,
                originalRequest ? `[사용자 원래 요청]\n${originalRequest}` : '',
                `[하류 작업자가 멈춘 이유]\n${report.missing}`,
                prevOutput ? `[하류 작업자의 최근 응답]\n${prevOutput.slice(0, 4000)}` : '',
                `[현재 역할: ${upstreamAgent ? upstreamAgent.name : upstreamId}]`,
                getAgentTaskInstruction(upstreamId, upstreamAgent),
                buildWiredUpstreamHint(upstreamId),
                `⚠️ 위 부족한 항목을 보완하는 작업만 수행하세요. 끝나면 자동으로 ${blockedAgent ? blockedAgent.name : blockedAgentId} 가 재실행됩니다.`,
            ].filter(Boolean).join('\n\n');

            await runAgent(upstreamId, upstreamPrompt, null, originalRequest);
        }

        async function autoAdvancePipeline(completedId, result) {
            await publishWorkflowEvent('approved', completedId, '자동 파이프라인: 다음 단계로 진행합니다.');

            const completedOutput = (result && result.output) ? result.output : (typeof result === 'string' ? result : '');

            // (1) Did this agent escalate to an upstream worker?
            const upstreamReport = parseUpstreamReport(completedOutput);
            if (upstreamReport && upstreamReport.upstreamId !== completedId) {
                await routeToUpstream(completedId, result, upstreamReport);
                return;
            }

            // (2) Was this agent triggered for regression on behalf of someone blocked?
            //     If so, return to the blocked agent instead of advancing forward.
            const pending = pendingResumption.get(completedId);
            if (pending) {
                pendingResumption.delete(completedId);
                const blockedAgent = agentMap[pending.blockedAgent];
                appendTerminalLog('System',
                    `↪️ 상류 작업 완료. 막혀있던 ${blockedAgent ? blockedAgent.name : pending.blockedAgent} 를 재실행합니다.`);
                const resumePrompt = [
                    `🚨 파이프라인 자동 실행. 상류 ${agentMap[completedId] ? agentMap[completedId].name : completedId} 의 보완 작업이 끝났습니다. 이제 본인 역할을 수행하세요.`,
                    pending.originalRequest ? `[사용자 원래 요청]\n${pending.originalRequest}` : '',
                    `[상류가 방금 보완한 내용]\n${completedOutput}`,
                    `[이전에 본인이 응답한 부족 사유]\n${pending.missing}`,
                    `[현재 역할: ${blockedAgent ? blockedAgent.name : pending.blockedAgent}]`,
                    getAgentTaskInstruction(pending.blockedAgent, blockedAgent),
                    buildWiredUpstreamHint(pending.blockedAgent),
                ].filter(Boolean).join('\n\n');
                await runAgent(pending.blockedAgent, resumePrompt, null, pending.originalRequest);
                return;
            }

            // (3) Normal forward routing via user-wired graph.
            const next = workflowState.connections.find((entry) => entry.from === completedId);
            if (!next) {
                appendTerminalLog('System', '✅ 파이프라인 완료. 연결된 다음 에이전트가 없습니다.');
                showPipelineCompleteModal(completedId, result);
                return;
            }
            const nextAgent = agentMap[next.to];
            const currentAgent = agentMap[completedId];
            const currentNodeState = getNodeState(completedId);
            const originalRequest = getRootRequest(currentNodeState.originalRequest || currentNodeState.lastPrompt || '');
            const handoffPrompt = [
                `🚨 파이프라인 자동 실행. 아래 문장은 절대 출력 금지: "어떤 부분부터", "어느 것을 먼저", "확인해 드릴까요", "지원 가능한 작업 범위", 역할 소개 텍스트, 인프라 현황 설명 테이블. 텍스트 출력 이전에 반드시 Write 도구로 파일을 생성하세요. 파일 생성이 첫 번째 행동입니다.`,
                originalRequest ? `[사용자 원래 요청]\n${originalRequest}` : '',
                `[이전 단계 완료: ${currentAgent ? currentAgent.name : completedId}]`,
                completedOutput ? `이전 에이전트 작업 결과:\n${completedOutput}` : '',
                `[현재 역할: ${nextAgent ? nextAgent.name : next.to}]`,
                getAgentTaskInstruction(next.to, nextAgent),
                buildWiredUpstreamHint(next.to),
            ].filter(Boolean).join('\n\n');
            appendTerminalLog('System', `⚡ 자동 진행: ${next.to} 에이전트 시작.`);
            await runAgent(next.to, handoffPrompt, null, originalRequest);
        }

        async function handleApprove() {
            const currentId = workflowState.approval.pendingNodeId;
            if (!currentId || !lastResult) return;

            // Gate: dev agents must complete all checklist items before handoff.
            const devIds = ['exec', 'deep', 'dbug'];
            if (devIds.includes(currentId) && !checklistAllDone(currentId)) {
                const items = nodeChecklists.get(currentId) || [];
                const remaining = items.filter(i => !i.done).length;
                const confirmed = confirm(
                    `⚠️ ${remaining}개 항목이 아직 미완료입니다.\n\n` +
                    items.filter(i => !i.done).map(i => '• ' + i.label).join('\n') +
                    `\n\n그래도 승인하고 다음 에이전트로 넘기시겠습니까?\n` +
                    `(아니오: 에이전트를 재실행하여 나머지 항목을 완료하세요)`
                );
                if (!confirmed) return;
            }
            closePanel();
            workflowState.approval.lastDecision = 'approved';
            workflowState.approval.pendingNodeId = '';
            setNodeStatus(currentId, 'completed');
            await publishWorkflowEvent('approved', currentId, '결과를 승인했습니다.');
            saveState();

            const prevOutput = lastResult ? (lastResult.output || '') : '';

            // Upstream regression takes precedence — even when manually approved,
            // if the agent escalated, route to the named upstream worker.
            const upstreamReport = parseUpstreamReport(prevOutput);
            if (upstreamReport && upstreamReport.upstreamId !== currentId) {
                await routeToUpstream(currentId, lastResult, upstreamReport);
                return;
            }

            // If this agent was triggered to unblock someone, resume them.
            const pending = pendingResumption.get(currentId);
            if (pending) {
                pendingResumption.delete(currentId);
                const blockedAgent = agentMap[pending.blockedAgent];
                appendTerminalLog('System',
                    `↪️ 상류 작업 완료. 막혀있던 ${blockedAgent ? blockedAgent.name : pending.blockedAgent} 를 재실행합니다.`);
                const resumePrompt = [
                    `🚨 파이프라인 자동 실행. 상류 ${agentMap[currentId] ? agentMap[currentId].name : currentId} 의 보완 작업이 끝났습니다. 이제 본인 역할을 수행하세요.`,
                    pending.originalRequest ? `[사용자 원래 요청]\n${pending.originalRequest}` : '',
                    `[상류가 방금 보완한 내용]\n${prevOutput}`,
                    `[이전에 본인이 응답한 부족 사유]\n${pending.missing}`,
                    `[현재 역할: ${blockedAgent ? blockedAgent.name : pending.blockedAgent}]`,
                    getAgentTaskInstruction(pending.blockedAgent, blockedAgent),
                    buildWiredUpstreamHint(pending.blockedAgent),
                ].filter(Boolean).join('\n\n');
                await runAgent(pending.blockedAgent, resumePrompt, null, pending.originalRequest);
                return;
            }

            const next = workflowState.connections.find((entry) => entry.from === currentId);
            if (!next) {
                appendTerminalLog('System', '✅ 승인됨. 연결된 다음 에이전트가 없어 워크플로우를 종료합니다.');
                showPipelineCompleteModal(currentId, lastResult);
                return;
            }
            const nextAgent = agentMap[next.to];
            const currentAgent = agentMap[currentId];
            const currentNodeState = getNodeState(currentId);
            const originalRequest = getRootRequest(currentNodeState.originalRequest || currentNodeState.lastPrompt || '');
            const handoffPrompt = [
                `🚨 파이프라인 자동 실행. 아래 문장은 절대 출력 금지: "어떤 부분부터", "어느 것을 먼저", "확인해 드릴까요", "지원 가능한 작업 범위", 역할 소개 텍스트, 인프라 현황 설명 테이블. 텍스트 출력 이전에 반드시 Write 도구로 파일을 생성하세요. 파일 생성이 첫 번째 행동입니다.`,
                originalRequest ? `[사용자 원래 요청]\n${originalRequest}` : '',
                `[이전 단계 완료: ${currentAgent ? currentAgent.name : currentId}]`,
                prevOutput ? `이전 에이전트 작업 결과:\n${prevOutput}` : '',
                `[현재 역할: ${nextAgent ? nextAgent.name : next.to}]`,
                getAgentTaskInstruction(next.to, nextAgent),
                buildWiredUpstreamHint(next.to),
            ].filter(Boolean).join('\n\n');
            appendTerminalLog('System', `✅ 승인됨. 다음 에이전트(${next.to}) 업무 시작.`);
            await runAgent(next.to, handoffPrompt, null, originalRequest);
        }

        function handleReject() {
            const currentId = workflowState.approval.pendingNodeId;
            if (!currentId) return;
            closePanel();
            workflowState.approval.lastDecision = 'rejected';
            setNodeStatus(currentId, 'rejected');
            publishWorkflowEvent('rejected', currentId, '결과가 반려되었습니다. 수정 지시를 입력하세요.');
            saveState();
            openDrawer(agentMap[currentId], `다음 결과를 보완해서 다시 실행하세요.\n\n이전 결과:\n${lastResult ? lastResult.output : ''}`);
        }

        function handleRerunWithEdit() {
            const currentId = workflowState.approval.pendingNodeId || panelAgentId;
            if (!currentId) return;
            const nodeState = getNodeState(currentId);
            // Preserve previous output so assignTask() can pass it as handoff context.
            nodeState.rerunContext = nodeState.output || null;
            closePanel();
            openDrawer(agentMap[currentId], '');
        }

        function reopenApprovalPanel(agentId) {
            const nodeState = getNodeState(agentId);
            if (!nodeState.output) { openDrawer(agentMap[agentId]); return; }
            lastResult = nodeState.output;
            workflowState.approval.pendingNodeId = agentId;
            const agent = agentMap[agentId];
            showPanel((agent ? agent.name : agentId) + ' 분석 결과', nodeState.output, agentId);
        }

