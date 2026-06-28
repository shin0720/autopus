        async function changeWorkspace(path, skipSave = false) {
            if (!skipSave && (workflowState.nodes.length > 0 || workflowState.connections.length > 0)) {
                await saveState();
            }
            const res = await fetch('/api/workspace/change', { method: 'POST', body: JSON.stringify({ path }) });
            const data = await res.json();
            document.getElementById('dir-name').innerText = data.currentDir;
            agentFiles.clear();
            fileCache.clear();
            try { sessionStorage.removeItem('agentFiles'); } catch(_) {}
            await Promise.all([loadWorkspaceList(), loadFiles()]);
            const norm = p => p.replace(/\\/g, '/').replace(/\/$/, '');
            const normNew = norm(data.currentDir);
            const normLast = norm(lastStatePath);
            const isDriveRoot = p => /^[a-zA-Z]:$/.test(p);
            // Only skip loadState when we're navigating deeper inside the same project.
            // Drive roots are never treated as project roots.
            const isInCurrentProject = normLast !== '' && !isDriveRoot(normLast) && normNew.startsWith(normLast + '/');
            if (!isInCurrentProject) {
                await loadState();
                lastStatePath = data.currentDir;
            }
            if (!skipSave) {
                pushRecentProject(data.currentDir);
                renderRecentProjects();
            }
        }

        function pushRecentProject(fullPath) {
            let projects = [];
            try { projects = JSON.parse(localStorage.getItem('recentProjects') || '[]'); } catch(_) {}
            projects = projects.filter(p => p.path !== fullPath);
            const label = fullPath.replace(/\\/g, '/').split('/').filter(Boolean).pop() || fullPath;
            projects.unshift({ path: fullPath, label, time: new Date().toISOString() });
            if (projects.length > 10) projects = projects.slice(0, 10);
            localStorage.setItem('recentProjects', JSON.stringify(projects));
        }

        function toggleRecentProjects(btn) {
            const el = document.getElementById('recent-projects-list');
            if (el.style.display !== 'none') { el.style.display = 'none'; return; }
            const rect = btn.getBoundingClientRect();
            el.style.top = (rect.bottom + 4) + 'px';
            el.style.right = (window.innerWidth - rect.right) + 'px';
            el.style.display = 'block';
        }

        document.addEventListener('click', e => {
            const list = document.getElementById('recent-projects-list');
            const btn = document.getElementById('btn-recent');
            if (list && !list.contains(e.target) && e.target !== btn) {
                list.style.display = 'none';
            }
        });

        function removeRecentProject(path) {
            let projects = [];
            try { projects = JSON.parse(localStorage.getItem('recentProjects') || '[]'); } catch(_) {}
            projects = projects.filter(p => p.path !== path);
            localStorage.setItem('recentProjects', JSON.stringify(projects));
            renderRecentProjects();
        }

        function renderRecentProjects() {
            let projects = [];
            try { projects = JSON.parse(localStorage.getItem('recentProjects') || '[]'); } catch(_) {}
            const list = document.getElementById('recent-projects-list');
            if (!list) return;
            list.innerHTML = '';
            if (!projects.length) {
                list.innerHTML = '<div style="padding:10px;font-size:11px;color:var(--text-dim);">최근 방문한 프로젝트 없음</div>';
                return;
            }
            projects.forEach(p => {
                const item = document.createElement('div');
                item.style.cssText = 'padding:8px 12px;cursor:pointer;border-bottom:1px solid var(--border-color);display:flex;justify-content:space-between;align-items:center;gap:8px;';
                const info = document.createElement('div');
                info.style.cssText = 'flex:1;overflow:hidden;min-width:0;';
                const lbl = document.createElement('div');
                lbl.style.cssText = 'font-size:12px;font-weight:bold;color:var(--fg);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;';
                lbl.textContent = '📁 ' + p.label;
                const sub = document.createElement('div');
                sub.style.cssText = 'font-size:10px;color:var(--text-dim);margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
                sub.title = p.path;
                sub.textContent = p.path;
                info.appendChild(lbl);
                info.appendChild(sub);
                const xBtn = document.createElement('button');
                xBtn.textContent = '×';
                xBtn.title = '목록에서 제거';
                xBtn.style.cssText = 'background:transparent;border:none;color:var(--text-dim);cursor:pointer;font-size:18px;padding:0 4px;flex-shrink:0;line-height:1;';
                xBtn.onmouseenter = () => { xBtn.style.color = 'var(--err)'; };
                xBtn.onmouseleave = () => { xBtn.style.color = 'var(--text-dim)'; };
                xBtn.onclick = (e) => { e.stopPropagation(); removeRecentProject(p.path); };
                item.appendChild(info);
                item.appendChild(xBtn);
                item.onmouseenter = () => item.style.background = '#1f242c';
                item.onmouseleave = () => item.style.background = '';
                item.onclick = () => {
                    document.getElementById('recent-projects-list').style.display = 'none';
                    changeWorkspace(p.path);
                };
                list.appendChild(item);
            });
        }

        async function loadWorkspaceList() {
            const res = await fetch('/api/workspace/list');
            const data = await res.json();
            document.getElementById('dir-name').innerText = data.current;
            const list = document.getElementById('folder-list');
            list.innerHTML = '';
            if (data.parent && data.parent !== data.current) list.innerHTML += '<div class="item" onclick="changeWorkspace(\'..\')">⬆️ 상위 폴더로</div>';
            data.folders.forEach((folder) => {
                const item = document.createElement('div');
                item.className = 'item';
                item.innerHTML = '📁 ' + folder;
                item.onclick = () => changeWorkspace(folder);
                list.appendChild(item);
            });
        }

        async function loadFiles() {
            const res = await fetch('/api/files/list');
            const data = await res.json(); // [{path, mod}, ...]
            const list = document.getElementById('file-list');
            list.innerHTML = '';
            const aid = selectedAgent?.id;
            if (!aid) {
                const hint = document.createElement('div');
                hint.className = 'agent-files-hint';
                hint.innerText = '작업자를 선택하면 파일을 배정할 수 있습니다';
                list.appendChild(hint);
            }
            const fileSet = agentFiles.get(aid) || new Set();
            const hadCache = fileCache.size > 0;
            const badgeBase = 'display:inline-flex;align-items:center;justify-content:center;min-width:14px;height:14px;border-radius:3px;font-size:9px;font-weight:700;flex-shrink:0;padding:0 2px;';
            data.forEach((entry) => {
                const file = entry.path;
                const mod = entry.mod || 0;
                const item = document.createElement('div');
                item.className = 'item';
                item.style.cssText = 'display:flex;align-items:center;gap:6px;';
                const cb = document.createElement('input');
                cb.type = 'checkbox';
                cb.checked = fileSet.has(file);
                cb.disabled = !aid;
                cb.onchange = (ev) => {
                    if (!aid) return;
                    if (!agentFiles.has(aid)) agentFiles.set(aid, new Set());
                    ev.target.checked ? agentFiles.get(aid).add(file) : agentFiles.get(aid).delete(file);
                    persistAgentFiles();
                };
                item.appendChild(cb);
                if (hadCache) {
                    if (!fileCache.has(file)) {
                        const b = document.createElement('span');
                        b.textContent = 'N'; b.title = '새로 생성된 파일';
                        b.style.cssText = badgeBase + 'background:#FF9F70;color:#161A24;';
                        item.appendChild(b);
                    } else if (mod > fileCache.get(file)) {
                        const b = document.createElement('span');
                        b.textContent = 'M'; b.title = '수정된 파일';
                        b.style.cssText = badgeBase + 'background:#7DD8B0;color:#161A24;';
                        item.appendChild(b);
                    }
                }
                const label = document.createElement('span');
                label.className = 'file-preview-link';
                label.innerText = '📄 ' + file;
                label.onclick = () => previewFile(file);
                item.appendChild(label);
                list.appendChild(item);
            });
            fileCache.clear();
            data.forEach(e => fileCache.set(e.path, e.mod || 0));
        }

        async function previewFile(path) {
            currentPreviewFile = path;
            const res = await fetch('/api/files/read?path=' + encodeURIComponent(path));
            const text = await res.text();
            document.getElementById('panel-title').innerText = '✏️ ' + path;
            const content = document.getElementById('panel-content');
            content.style.padding = '0';
            const ta = document.createElement('textarea');
            ta.id = 'file-edit-area';
            ta.value = text || '';
            ta.style.cssText = 'width:100%;height:calc(100% - 44px);background:var(--bg);color:var(--fg);border:none;border-bottom:1px solid var(--border);padding:16px;font-family:monospace;font-size:13px;box-sizing:border-box;resize:none;outline:none;';
            const saveBar = document.createElement('div');
            saveBar.style.cssText = 'height:44px;padding:8px 16px;display:flex;justify-content:flex-end;background:var(--bg2);border-bottom-left-radius:0;border-bottom-right-radius:0;';
            const saveBtn = document.createElement('button');
            saveBtn.className = 'btn-run';
            saveBtn.style.background = 'var(--accent-green)';
            saveBtn.innerText = '💾 저장';
            saveBtn.onclick = saveFileEdit;
            saveBar.appendChild(saveBtn);
            content.innerHTML = '';
            content.appendChild(ta);
            content.appendChild(saveBar);
            document.getElementById('approval-footer').style.display = 'none';

            document.getElementById('overlay-panel').style.display = 'flex';
        }

