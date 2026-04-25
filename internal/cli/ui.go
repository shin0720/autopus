package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

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
			
			// 현재 디렉토리에서 설정 로드
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

			// UI Main Page (n8n Style Prototype)
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, dashboardHTML)
			})

			// 브라우저 자동 열기
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
<html>
<head>
    <title>Autopus Dashboard</title>
    <style>
        body { background: #0f172a; color: #f8fafc; font-family: sans-serif; margin: 0; overflow: hidden; }
        .canvas { width: 100vw; height: 100vh; position: relative; background-image: radial-gradient(#334155 1px, transparent 1px); background-size: 20px 20px; }
        .node { position: absolute; background: #1e293b; border: 2px solid #3b82f6; border-radius: 8px; width: 200px; padding: 10px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); cursor: move; }
        .node-header { font-weight: bold; border-bottom: 1px solid #334155; padding-bottom: 5px; margin-bottom: 10px; display: flex; justify-content: space-between; }
        .status-dot { width: 10px; height: 10px; border-radius: 50%; background: #22c55e; align-self: center; }
        .node-body { font-size: 12px; color: #94a3b8; }
        .toolbar { position: fixed; top: 20px; left: 20px; background: rgba(30, 41, 59, 0.8); padding: 10px; border-radius: 8px; backdrop-filter: blur(4px); z-index: 10; }
        h1 { margin: 0; font-size: 18px; color: #3b82f6; }
    </style>
</head>
<body>
    <div class="toolbar">
        <h1>🐙 Autopus Dashboard</h1>
        <div id="project-name">Loading project...</div>
    </div>
    <div class="canvas" id="canvas"></div>

    <script>
        async function loadStatus() {
            const res = await fetch('/api/status');
            const data = await res.json();
            document.getElementById('project-name').innerText = "Project: " + data.project;
            
            const canvas = document.getElementById('canvas');
            let x = 100;
            data.agents.forEach(p => {
                const node = document.createElement('div');
                node.className = 'node';
                node.style.left = x + 'px';
                node.style.top = '200px';
                node.innerHTML = '<div class="node-header">' + p.name + '<div class="status-dot"></div></div>' +
                               '<div class="node-body">Model: ' + (p.args ? p.args.join(" ") : "default") + '<br>Binary: ' + p.binary + '</div>';
                canvas.appendChild(node);
                x += 300;
            });
        }
        loadStatus();
    </script>
</body>
</html>
`
