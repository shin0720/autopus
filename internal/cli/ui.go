package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/shin0720/auto-adk/pkg/config"
)

// newUICmd는 웹 UI 서버를 실행하는 ui 서브커맨드를 생성한다.
func newUICmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Autopus 시각적 대시보드 실행",
		Long:  "n8n 스타일의 노드 기반 워크플로우 대시보드를 실행합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("localhost:%d", port)
			cfg, err := config.Load(".")
			if err != nil {
				return err
			}

			fmt.Printf("🐙 Autopus Dashboard 시작 중... http://%s\n", addr)

			// API: 현재 설정 및 에이전트 상태
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"project": cfg.ProjectName,
					"agents":  cfg.Orchestra.Providers,
					"quality": cfg.Quality.Default,
				})
			})

			// API: 파일 목록 가져오기
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() { return nil }
					// 숨김 폴더 및 무시 폴더 제외
					if strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") || strings.Contains(path, "vendor") {
						return nil
					}
					files = append(files, path)
					return nil
				})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(files)
			})

			// API: 업무 할당
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)
				time.Sleep(1 * time.Second)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "업무 완료"})
			})

			// UI Main Page
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, dashboardHTML)
			})

			go openBrowser("http://" + addr)
			return http.ListenAndServe(addr, nil)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "서버 포트 번호")
	return cmd
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux": err = exec.Command("xdg-open", url).Start()
	case "windows": err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("브라우저 열기 실패: %v\n", err) }
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <title>Autopus Virtual Studio</title>
    <style>
        :root {
            --bg-color: #0d1117;
            --sidebar-bg: #161b22;
            --border-color: #30363d;
            --accent-blue: #58a6ff;
            --accent-green: #3fb950;
            --text-main: #c9d1d9;
            --text-dim: #8b949e;
        }
        body { background: var(--bg-color); color: var(--text-main); font-family: sans-serif; margin: 0; display: flex; height: 100vh; overflow: hidden; }
        
        /* Layout */
        .sidebar-left { width: 220px; background: var(--sidebar-bg); border-right: 1px solid var(--border-color); display: flex; flex-direction: column; }
        .main-area { flex: 1; display: flex; flex-direction: column; position: relative; }
        .sidebar-right { width: 350px; background: var(--sidebar-bg); border-left: 1px solid var(--border-color); padding: 20px; display: none; flex-direction: column; }
        
        .top-bar { height: 50px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border-color); display: flex; align-items: center; padding: 0 20px; justify-content: space-between; }
        
        .file-list { flex: 1; overflow-y: auto; padding: 10px; font-size: 12px; }
        .file-item { padding: 4px 8px; border-radius: 4px; cursor: pointer; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .file-item:hover { background: #1f242c; color: var(--accent-blue); }

        .canvas { flex: 1; position: relative; background-image: radial-gradient(#30363d 1px, transparent 1px); background-size: 30px 30px; overflow: hidden; }
        .lane-container { display: flex; width: 100%; height: 100%; position: absolute; pointer-events: none; }
        .lane { flex: 1; border-right: 1px dashed #30363d; display: flex; flex-direction: column; align-items: center; padding-top: 30px; }
        .lane-title { font-size: 10px; font-weight: 800; color: var(--text-dim); text-transform: uppercase; margin-bottom: 15px; }

        .node { background: #21262d; border: 1px solid var(--border-color); border-radius: 6px; width: 160px; padding: 8px; margin-bottom: 12px; pointer-events: auto; cursor: pointer; transition: 0.2s; }
        .node:hover { border-color: var(--accent-blue); }
        .node.active { border-color: var(--accent-blue); box-shadow: 0 0 10px rgba(88, 166, 255, 0.4); }
        .node-name { font-size: 12px; font-weight: bold; margin-bottom: 4px; }
        .node-role { font-size: 10px; color: var(--text-dim); }

        .terminal { height: 150px; background: #010409; border-top: 1px solid var(--border-color); padding: 10px; overflow-y: auto; font-family: monospace; font-size: 11px; }
        .log-entry { margin-bottom: 3px; }

        /* Work Panel */
        .drawer-title { font-size: 16px; font-weight: bold; color: var(--accent-blue); margin-bottom: 20px; }
        textarea { flex: 1; background: #0d1117; border: 1px solid var(--border-color); border-radius: 6px; color: white; padding: 12px; margin-bottom: 15px; resize: none; line-height: 1.6; }
        .btn-run { background: var(--accent-blue); color: white; border: none; padding: 10px; border-radius: 6px; font-weight: 700; cursor: pointer; }
    </style>
</head>
<body>
    <div class="sidebar-left">
        <div style="padding:15px; border-bottom:1px solid var(--border-color); font-weight:bold; font-size:13px;">파일 탐색기</div>
        <div class="file-list" id="file-list"></div>
    </div>

    <div class="main-area">
        <div class="top-bar">
            <div style="font-weight:900; color:var(--accent-blue); font-size:16px;">🐙 AUTOPUS STUDIO</div>
            <div id="proj-name" style="font-size:12px; color:var(--accent-green)">PROJECT: LOADING...</div>
        </div>
        
        <div class="canvas">
            <div class="lane-container">
                <div class="lane" id="lane-plan"><div class="lane-title">PLAN</div></div>
                <div class="lane" id="lane-dev"><div class="lane-title">DEV</div></div>
                <div class="lane" id="lane-qa"><div class="lane-title">QA</div></div>
                <div class="lane" id="lane-ops"><div class="lane-title">OPS</div></div>
            </div>
        </div>

        <div class="terminal" id="terminal"></div>
    </div>

    <div class="sidebar-right" id="sidebar-right">
        <div class="drawer-title" id="drawer-title">업무 할당</div>
        <div style="font-size:12px; color:var(--text-dim); margin-bottom:10px;" id="drawer-desc"></div>
        <textarea id="prompt-input" placeholder="이 에이전트에게 내릴 구체적인 지시사항을 넓게 작성하세요..."></textarea>
        <button class="btn-run" onclick="assignTask()">업무 시작</button>
        <button class="btn-run" style="background:transparent; color:var(--text-dim); margin-top:10px;" onclick="closeDrawer()">닫기</button>
    </div>

    <script>
        let selectedAgent = null;
        const agents = [
            { id: "planner", name: "Planner", role: "기획자", dept: "plan", desc: "요구사항 분석 및 태스크 분해" },
            { id: "spec", name: "Spec Writer", role: "명세 작성자", dept: "plan", desc: "EARS 명세서 작성" },
            { id: "arch", name: "Architect", role: "아키텍트", dept: "plan", desc: "시스템 구조 설계" },
            { id: "expl", name: "Explorer", role: "탐험가", dept: "plan", desc: "코드베이스 구조 매핑" },
            { id: "exec", name: "Executor", role: "실행자", dept: "dev", desc: "TDD 기반 기능 구현" },
            { id: "deep", name: "Deep Worker", role: "심층 작업자", dept: "dev", desc: "복잡한 장기 태스크 전담" },
            { id: "dbug", name: "Debugger", role: "해결사", dept: "dev", desc: "버그 수정 및 예외 처리" },
            { id: "anno", name: "Annotator", role: "태그 관리자", dept: "dev", desc: "@AX 태그 관리" },
            { id: "test", name: "Tester", role: "테스터", dept: "qa", desc: "테스트 코드 작성" },
            { id: "val", name: "Validator", role: "검증자", dept: "qa", desc: "품질 게이트 체크" },
            { id: "fend", name: "Frontend Spec", role: "UI 전문가", dept: "qa", desc: "E2E 테스트" },
            { id: "uxv", name: "UX Validator", role: "UX 검증자", dept: "qa", desc: "비주얼 회귀 검사" },
            { id: "perf", name: "Perf Engineer", role: "성능 전문가", dept: "qa", desc: "병목 분석" },
            { id: "rev", name: "Reviewer", role: "리뷰어", dept: "ops", desc: "코드 리뷰 및 반려" },
            { id: "sec", name: "Security Audit", role: "보안 감사", dept: "ops", desc: "보안 취약점 스캔" },
            { id: "devops", name: "DevOps", role: "운영자", dept: "ops", desc: "CI/CD 자동화" }
        ];

        function init() {
            agents.forEach(a => {
                const lane = document.getElementById('lane-' + a.dept);
                const node = document.createElement('div');
                node.className = 'node';
                node.id = 'node-' + a.id;
                node.onclick = () => openDrawer(a);
                node.innerHTML = '<div class="node-name">' + a.name + '</div><div class="node-role">' + a.role + '</div>';
                lane.appendChild(node);
            });
            loadFiles();
        }

        async function loadFiles() {
            const res = await fetch('/api/files/list');
            const files = await res.json();
            const list = document.getElementById('file-list');
            files.forEach(f => {
                const item = document.createElement('div');
                item.className = 'file-item';
                item.innerText = '📄 ' + f;
                list.appendChild(item);
            });
        }

        function openDrawer(agent) {
            selectedAgent = agent;
            document.getElementById('drawer-title').innerText = agent.name + ' 업무 할당';
            document.getElementById('drawer-desc').innerText = agent.desc;
            document.getElementById('sidebar-right').style.display = 'flex';
        }

        function closeDrawer() { document.getElementById('sidebar-right').style.display = 'none'; }

        function addLog(msg) {
            const term = document.getElementById('terminal');
            const entry = document.createElement('div');
            entry.className = 'log-entry';
            entry.innerHTML = '<span style="color:var(--text-dim)">[' + new Date().toLocaleTimeString() + ']</span> ' + msg;
            term.appendChild(entry);
            term.scrollTop = term.scrollHeight;
        }

        async function assignTask() {
            const prompt = document.getElementById('prompt-input').value;
            if (!prompt) return;
            const agent = selectedAgent;
            addLog('<b>' + agent.name + '</b>에게 업무를 전달했습니다.');
            const node = document.getElementById('node-' + agent.id);
            node.classList.add('active');
            
            await fetch('/api/agent/assign', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ agentId: agent.id, prompt })
            });

            setTimeout(() => {
                addLog('<b>' + agent.name + '</b>: 작업이 성공적으로 완료되었습니다.');
                node.classList.remove('active');
            }, 2000);
        }

        init();
    </script>
</body>
</html>
`
