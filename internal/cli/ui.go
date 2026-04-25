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
				
				// 실전에서는 여기서 실제 에이전트를 구동합니다.
				// 현재는 UI 연동 확인을 위해 3초 대기 후 성공 응답을 보냅니다.
				time.Sleep(2 * time.Second)
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"message": fmt.Sprintf("%s 에이전트가 업무를 성공적으로 접수했습니다.", req.AgentID),
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
            --accent-red: #f85149;
            --text-main: #c9d1d9;
            --text-dim: #8b949e;
        }
        body { background: var(--bg-color); color: var(--text-main); font-family: sans-serif; margin: 0; overflow: hidden; }
        .canvas { width: 100vw; height: 100vh; position: relative; background-image: radial-gradient(var(--grid-color) 1px, transparent 1px); background-size: 40px 40px; }
        
        .lane-container { display: flex; width: 100%; height: 100%; position: absolute; top: 0; left: 0; pointer-events: none; }
        .lane { flex: 1; border-right: 1px dashed var(--grid-color); display: flex; flex-direction: column; align-items: center; padding-top: 80px; }
        .lane-title { font-size: 12px; font-weight: 800; color: var(--text-dim); text-transform: uppercase; margin-bottom: 20px; background: #1f242c; padding: 4px 12px; border-radius: 12px; }

        .toolbar { position: fixed; top: 0; left: 0; right: 0; height: 60px; background: rgba(22, 27, 34, 0.8); backdrop-filter: blur(10px); border-bottom: 1px solid var(--border-color); display: flex; align-items: center; padding: 0 20px; z-index: 1000; justify-content: space-between; }
        .logo { font-size: 20px; font-weight: 900; color: var(--accent-blue); }

        .node { position: relative; background: var(--panel-color); border: 1px solid var(--border-color); border-radius: 8px; width: 220px; margin-bottom: 20px; pointer-events: auto; box-shadow: 0 4px 12px rgba(0,0,0,0.3); transition: all 0.2s; cursor: pointer; }
        .node:hover { border-color: var(--accent-blue); transform: translateY(-2px); background: #1c2128; }
        .node.active { border-color: var(--accent-blue); box-shadow: 0 0 15px rgba(88, 166, 255, 0.4); animation: pulse 1.5s infinite; }
        
        .node-header { padding: 8px 12px; border-bottom: 1px solid var(--border-color); display: flex; justify-content: space-between; align-items: center; }
        .node-title { font-weight: 700; font-size: 13px; }
        .status-dot { width: 8px; height: 8px; border-radius: 50%; background: #484f58; }
        .active .status-dot { background: var(--accent-blue); }

        .node-body { padding: 10px 12px; font-size: 11px; color: var(--text-dim); }
        .node-role { color: var(--text-main); font-weight: 600; margin-bottom: 2px; }

        /* Modal Style */
        .modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); display: none; justify-content: center; align-items: center; z-index: 2000; }
        .modal { background: var(--panel-color); border: 1px solid var(--border-color); border-radius: 12px; width: 450px; padding: 25px; box-shadow: 0 20px 50px rgba(0,0,0,0.5); }
        .modal-h2 { margin: 0 0 15px 0; color: var(--accent-blue); font-size: 18px; }
        textarea { width: 100%; height: 120px; background: #0d1117; border: 1px solid var(--border-color); border-radius: 6px; color: white; padding: 10px; margin-bottom: 15px; resize: none; box-sizing: border-box; }
        .modal-btns { display: flex; justify-content: flex-end; gap: 10px; }
        .btn { padding: 8px 20px; border-radius: 6px; border: none; font-weight: 700; cursor: pointer; }
        .btn-primary { background: var(--accent-blue); color: white; }
        .btn-secondary { background: transparent; color: var(--text-dim); border: 1px solid var(--border-color); }

        @keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.6; } 100% { opacity: 1; } }
    </style>
</head>
<body>
    <div class="toolbar">
        <div class="logo">🐙 Autopus Virtual Studio</div>
        <div id="project-name" style="font-size:14px; color:var(--accent-green)">프로젝트: 로딩 중...</div>
    </div>

    <div class="canvas">
        <div class="lane-container">
            <div class="lane" id="lane-plan"><div class="lane-title">기획 & 설계</div></div>
            <div class="lane" id="lane-dev"><div class="lane-title">개발 & 구현</div></div>
            <div class="lane" id="lane-qa"><div class="lane-title">검증 & QA</div></div>
            <div class="lane" id="lane-ops"><div class="lane-title">리뷰 & 배포</div></div>
        </div>
    </div>

    <div class="modal-overlay" id="modal-overlay">
        <div class="modal">
            <h2 class="modal-h2" id="modal-title">업무 할당</h2>
            <p id="modal-desc" style="font-size:12px; color:var(--text-dim); margin-bottom:15px;"></p>
            <textarea id="prompt-input" placeholder="이 직원에게 시킬 일을 입력하세요..."></textarea>
            <div class="modal-btns">
                <button class="btn btn-secondary" onclick="closeModal()">취소</button>
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
            { id: "anno", name: "Annotator", role: "태그 관리자", dept: "dev", desc: "@AX 태그 및 주석 관리" },
            { id: "test", name: "Tester", role: "테스터", dept: "qa", desc: "테스트 코드 작성 (85%+)" },
            { id: "val", name: "Validator", role: "검증자", dept: "qa", desc: "빌드/린트/품질 게이트 체크" },
            { id: "fend", name: "Frontend Spec", role: "UI 전문가", dept: "qa", desc: "Playwright E2E 테스트" },
            { id: "uxv", name: "UX Validator", role: "UX 검증자", dept: "qa", desc: "비주얼 회귀 검사" },
            { id: "perf", name: "Perf Engineer", role: "성능 전문가", dept: "qa", desc: "벤치마크 및 병목 분석" },
            { id: "rev", name: "Reviewer", role: "리뷰어", dept: "ops", desc: "TRUST-5 코드 리뷰 및 반려" },
            { id: "sec", name: "Security Audit", role: "보안 감사", dept: "ops", desc: "취약점 스캔 및 보안 강화" },
            { id: "devops", name: "DevOps", role: "운영자", dept: "ops", desc: "CI/CD 및 배포 자동화" }
        ];

        function init() {
            agents.forEach(a => {
                const lane = document.getElementById('lane-' + a.dept);
                const node = document.createElement('div');
                node.className = 'node';
                node.id = 'node-' + a.id;
                node.onclick = () => openModal(a);
                node.innerHTML = '<div class="node-header"><div class="node-title">' + a.name + '</div><div class="status-dot"></div></div>' +
                               '<div class="node-body"><div class="node-role">' + a.role + '</div><div>' + a.desc + '</div></div>';
                lane.appendChild(node);
            });
        }

        function openModal(agent) {
            selectedAgent = agent;
            document.getElementById('modal-title').innerText = agent.name + ' (' + agent.role + ')에게 업무 할당';
            document.getElementById('modal-desc').innerText = agent.desc;
            document.getElementById('modal-overlay').style.display = 'flex';
            document.getElementById('prompt-input').focus();
        }

        function closeModal() {
            document.getElementById('modal-overlay').style.display = 'none';
        }

        async function assignTask() {
            const prompt = document.getElementById('prompt-input').value;
            if (!prompt) return alert('프롬프트를 입력하세요!');
            
            const agentId = selectedAgent.id;
            closeModal();
            
            // UI 상태 변경 (작업 중)
            const node = document.getElementById('node-' + agentId);
            node.classList.add('active');
            
            try {
                const res = await fetch('/api/agent/assign', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ agentId, prompt })
                });
                const data = await res.json();
                console.log('결과:', data);
                // 3초 후 작업 완료 시뮬레이션
                setTimeout(() => {
                    node.classList.remove('active');
                    alert(data.message);
                }, 1000);
            } catch(e) {
                node.classList.remove('active');
                alert('업무 할당 실패!');
            }
        }

        async function update() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                document.getElementById('project-name').innerText = "프로젝트: " + data.project.toUpperCase();
            } catch(e) {}
        }

        init();
        update();
    </script>
</body>
</html>
`
