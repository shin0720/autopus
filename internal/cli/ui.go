package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
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
				fmt.Printf("설정 로드 실패: %v\n", err)
				return err
			}

			fmt.Printf("🐙 Autopus Dashboard 시작 중... http://%s\n", addr)

			// API: 현재 설정 및 에이전트 상태 반환
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"project": cfg.ProjectName,
					"agents":  cfg.Orchestra.Providers,
					"quality": cfg.Quality.Default,
				})
			})

			// API: 특정 에이전트에게 업무 할당
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				var req struct {
					AgentID string `json:"agentId"`
					Prompt  string `json:"prompt"`
				}
				json.NewDecoder(r.Body).Decode(&req)
				
				fmt.Printf("📝 [%s] 업무 할당: %s\n", req.AgentID, req.Prompt)
				
				// 시뮬레이션: 에이전트 작업 대기
				time.Sleep(1 * time.Second)
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"message": fmt.Sprintf("%s 에이전트가 업무를 성공적으로 완료했습니다.", req.AgentID),
				})
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
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		fmt.Printf("브라우저를 열 수 없습니다: %v\n", err)
	}
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <title>Autopus 16인 가상 스튜디오</title>
    <style>
        :root {
            --bg-color: #0d1117;
            --grid-color: #30363d;
            --panel-color: #161b22;
            --border-color: #30363d;
            --accent-blue: #58a6ff;
            --accent-green: #3fb950;
            --accent-orange: #d29922;
            --text-main: #c9d1d9;
            --text-dim: #8b949e;
            --terminal-bg: #010409;
        }
        body { background: var(--bg-color); color: var(--text-main); font-family: sans-serif; margin: 0; overflow: hidden; display: flex; flex-direction: column; height: 100vh; }
        
        .top-bar { height: 50px; background: var(--panel-color); border-bottom: 1px solid var(--border-color); display: flex; align-items: center; padding: 0 20px; justify-content: space-between; z-index: 1000; }
        .stats { display: flex; gap: 20px; font-size: 12px; }
        .stat-val { color: var(--accent-blue); font-weight: bold; }

        .main-content { flex: 1; position: relative; overflow: hidden; }
        .canvas { width: 100%; height: 100%; position: relative; background-image: radial-gradient(var(--grid-color) 1px, transparent 1px); background-size: 40px 40px; }
        
        .lane-container { display: flex; width: 100%; height: 100%; position: absolute; pointer-events: none; }
        .lane { flex: 1; border-right: 1px dashed var(--grid-color); display: flex; flex-direction: column; align-items: center; padding-top: 40px; }
        .lane-title { font-size: 11px; font-weight: 800; color: var(--text-dim); text-transform: uppercase; margin-bottom: 20px; background: #1f242c; padding: 3px 10px; border-radius: 10px; }

        .node { position: relative; background: var(--panel-color); border: 1px solid var(--border-color); border-radius: 8px; width: 190px; margin-bottom: 15px; pointer-events: auto; box-shadow: 0 4px 12px rgba(0,0,0,0.3); transition: all 0.2s; cursor: pointer; z-index: 10; }
        .node:hover { border-color: var(--accent-blue); transform: scale(1.02); }
        .node.active { border-color: var(--accent-blue); box-shadow: 0 0 15px rgba(88, 166, 255, 0.3); animation: pulse 1.5s infinite; }
        
        .node-header { padding: 6px 10px; border-bottom: 1px solid var(--border-color); display: flex; justify-content: space-between; align-items: center; font-size: 12px; }
        .status-dot { width: 6px; height: 6px; border-radius: 50%; background: #484f58; }
        .active .status-dot { background: var(--accent-blue); }

        .node-body { padding: 8px 10px; font-size: 10px; color: var(--text-dim); }
        .node-role { color: var(--text-main); font-weight: 600; margin-bottom: 2px; font-size: 11px; }

        .terminal { height: 180px; background: var(--terminal-bg); border-top: 1px solid var(--border-color); padding: 15px; overflow-y: auto; font-family: monospace; font-size: 12px; line-height: 1.5; }
        .log-entry { margin-bottom: 4px; }
        .log-time { color: var(--text-dim); margin-right: 10px; }
        .log-agent { color: var(--accent-orange); font-weight: bold; margin-right: 10px; }
        .log-msg { color: var(--text-main); }

        .modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); display: none; justify-content: center; align-items: center; z-index: 2000; }
        .modal { background: var(--panel-color); border: 1px solid var(--border-color); border-radius: 12px; width: 400px; padding: 20px; }
        textarea { width: 100%; height: 100px; background: #0d1117; border: 1px solid var(--border-color); border-radius: 6px; color: white; padding: 10px; margin: 15px 0; resize: none; box-sizing: border-box; }
        .btn { padding: 6px 15px; border-radius: 6px; border: none; font-weight: 700; cursor: pointer; font-size: 12px; }
        .btn-primary { background: var(--accent-blue); color: white; }

        @keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.7; } 100% { opacity: 1; } }
    </style>
</head>
<body>
    <div class="top-bar">
        <div style="font-weight:900; color:var(--accent-blue)">🐙 AUTOPUS STUDIO</div>
        <div class="stats">
            <div>현재 프로젝트: <span class="stat-val" id="proj-id">-</span></div>
            <div>활성 에이전트: <span class="stat-val">16명</span></div>
            <div>누적 토큰: <span class="stat-val" id="token-count">1,240</span></div>
            <div>예상 비용: <span class="stat-val" id="cost-val">$0.02</span></div>
        </div>
    </div>

    <div class="main-content">
        <div class="canvas">
            <div class="lane-container">
                <div class="lane" id="lane-plan"><div class="lane-title">기획 & 설계</div></div>
                <div class="lane" id="lane-dev"><div class="lane-title">개발 & 구현</div></div>
                <div class="lane" id="lane-qa"><div class="lane-title">검증 & QA</div></div>
                <div class="lane" id="lane-ops"><div class="lane-title">리뷰 & 배포</div></div>
            </div>
        </div>
    </div>

    <div class="terminal" id="terminal">
        <div class="log-entry"><span class="log-time">시스템</span><span class="log-agent">[System]</span><span class="log-msg">Autopus 16인 가상 스튜디오가 연결되었습니다. 업무를 시작할 에이전트를 클릭하세요.</span></div>
    </div>

    <div class="modal-overlay" id="modal-overlay">
        <div class="modal">
            <div id="modal-title" style="font-weight:bold; color:var(--accent-blue)">업무 할당</div>
            <textarea id="prompt-input" placeholder="무엇을 도와드릴까요?"></textarea>
            <div style="display:flex; justify-content:flex-end; gap:10px;">
                <button class="btn" style="background:transparent; color:var(--text-dim)" onclick="closeModal()">취소</button>
                <button class="btn btn-primary" onclick="assignTask()">업무 시작</button>
            </div>
        </div>
    </div>

    <script>
        let selectedAgent = null;
        const agents = [
            { id: "planner", name: "Planner", role: "기획자", dept: "plan", desc: "아이디어 분석 및 태스크 분해" },
            { id: "spec", name: "Spec Writer", role: "명세 작성자", dept: "plan", desc: "EARS 명세서 작성" },
            { id: "arch", name: "Architect", role: "아키텍트", dept: "plan", desc: "시스템 구조 및 DB 설계" },
            { id: "expl", name: "Explorer", role: "탐험가", dept: "plan", desc: "코드베이스 구조 매핑" },
            { id: "exec", name: "Executor", role: "실행자", dept: "dev", desc: "TDD 기반 핵심 기능 구현" },
            { id: "deep", name: "Deep Worker", role: "심층 작업자", dept: "dev", desc: "복잡한 장기 태스크 전담" },
            { id: "dbug", name: "Debugger", role: "해결사", dept: "dev", desc: "버그 수정 및 예외 처리" },
            { id: "anno", name: "Annotator", role: "태그 관리자", dept: "dev", desc: "@AX 태그 관리" },
            { id: "test", name: "Tester", role: "테스터", dept: "qa", desc: "테스트 코드 작성 (85%+)" },
            { id: "val", name: "Validator", role: "검증자", dept: "qa", desc: "품질 게이트 체크" },
            { id: "fend", name: "Frontend Spec", role: "UI 전문가", dept: "qa", desc: "Playwright E2E 테스트" },
            { id: "uxv", name: "UX Validator", role: "UX 검증자", dept: "qa", desc: "비주얼 회귀 검사" },
            { id: "perf", name: "Perf Engineer", role: "성능 전문가", dept: "qa", desc: "벤치마크 및 병목 분석" },
            { id: "rev", name: "Reviewer", role: "리뷰어", dept: "ops", desc: "코드 리뷰 및 반려" },
            { id: "sec", name: "Security Audit", role: "보안 감사", dept: "ops", desc: "보안 취약점 스캔" },
            { id: "devops", name: "DevOps", role: "운영자", dept: "ops", desc: "CI/CD 배포 자동화" }
        ];

        function addLog(agentName, message) {
            const term = document.getElementById('terminal');
            const now = new Date().toLocaleTimeString();
            const entry = document.createElement('div');
            entry.className = 'log-entry';
            entry.innerHTML = '<span class="log-time">' + now + '</span>' +
                            '<span class="log-agent">[' + agentName + ']</span>' +
                            '<span class="log-msg">' + message + '</span>';
            term.appendChild(entry);
            term.scrollTop = term.scrollHeight;
        }

        function init() {
            agents.forEach(a => {
                const lane = document.getElementById('lane-' + a.dept);
                const node = document.createElement('div');
                node.className = 'node';
                node.id = 'node-' + a.id;
                node.onclick = () => { selectedAgent = a; document.getElementById('modal-overlay').style.display = 'flex'; };
                node.innerHTML = '<div class="node-header"><div class="node-title">' + a.name + '</div><div class="status-dot"></div></div>' +
                               '<div class="node-body"><div class="node-role">' + a.role + '</div><div>' + a.desc + '</div></div>';
                lane.appendChild(node);
            });
        }

        async function assignTask() {
            const prompt = document.getElementById('prompt-input').value;
            const agent = selectedAgent;
            if (!prompt) return;
            closeModal();
            
            const node = document.getElementById('node-' + agent.id);
            node.classList.add('active');
            addLog(agent.name, "업무를 시작합니다: " + prompt);
            
            try {
                const res = await fetch('/api/agent/assign', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ agentId: agent.id, prompt: prompt })
                });
                const data = await res.json();
                
                setTimeout(() => {
                    addLog(agent.name, "생각 중... (코드베이스 분석 및 대안 탐색)");
                    setTimeout(() => {
                        addLog(agent.name, "작업 성공: " + data.message);
                        node.classList.remove('active');
                        // 가상 토큰/비용 업데이트
                        document.getElementById('token-count').innerText = parseInt(document.getElementById('token-count').innerText) + 450;
                    }, 2000);
                }, 1000);
            } catch(e) {
                node.classList.remove('active');
                addLog("System", "에러 발생: " + e.message);
            }
        }

        function closeModal() { document.getElementById('modal-overlay').style.display = 'none'; }

        async function update() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                document.getElementById('proj-id').innerText = data.project.toUpperCase();
            } catch(e) {}
        }

        init();
        update();
    </script>
</body>
</html>
`
