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
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <title>Autopus 제어 센터</title>
    <style>
        :root {
            --bg-color: #0b0f1a;
            --panel-color: #161b2c;
            --border-color: #2d334a;
            --accent-color: #3b82f6;
            --text-main: #f8fafc;
            --text-dim: #94a3b8;
            --success: #22c55e;
        }
        body { background: var(--bg-color); color: var(--text-main); font-family: sans-serif; margin: 0; overflow: hidden; }
        .canvas { width: 100vw; height: 100vh; position: relative; background-image: radial-gradient(#2d334a 1px, transparent 1px); background-size: 30px 30px; }
        
        .toolbar { 
            position: fixed; top: 0; left: 0; right: 0; height: 60px;
            background: rgba(22, 27, 44, 0.8); backdrop-filter: blur(10px);
            border-bottom: 1px solid var(--border-color);
            display: flex; align-items: center; padding: 0 20px; z-index: 100;
            justify-content: space-between;
        }
        .logo { font-size: 20px; font-weight: 800; color: var(--accent-color); display: flex; align-items: center; gap: 10px; }
        .project-info { font-size: 14px; color: var(--text-dim); background: #1e293b; padding: 5px 12px; border-radius: 20px; border: 1px solid var(--border-color); }

        .node { 
            position: absolute; background: var(--panel-color); 
            border: 1px solid var(--border-color); border-radius: 12px; 
            width: 260px; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.5);
            transition: transform 0.2s, border-color 0.2s;
        }
        .node:hover { border-color: var(--accent-color); transform: translateY(-2px); }
        .node-header { 
            padding: 12px 15px; border-bottom: 1px solid var(--border-color);
            display: flex; justify-content: space-between; align-items: center;
            background: rgba(59, 130, 246, 0.05); border-radius: 12px 12px 0 0;
        }
        .node-title { font-weight: 700; font-size: 15px; letter-spacing: -0.02em; }
        .status-badge { font-size: 10px; padding: 2px 8px; border-radius: 10px; background: rgba(34, 197, 94, 0.2); color: var(--success); border: 1px solid rgba(34, 197, 94, 0.3); }
        
        .node-body { padding: 15px; }
        .info-row { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 12px; }
        .info-label { color: var(--text-dim); }
        .info-value { color: var(--text-main); font-family: monospace; }

        .connector-line {
            position: absolute; height: 2px; background: linear-gradient(90deg, var(--accent-color), transparent);
            transform-origin: left center; z-index: -1; opacity: 0.5;
        }

        .bottom-bar {
            position: fixed; bottom: 20px; left: 50%; transform: translateX(-50%);
            background: var(--panel-color); padding: 10px 25px; border-radius: 30px;
            border: 1px solid var(--border-color); display: flex; gap: 20px;
            box-shadow: 0 10px 15px -3px rgba(0,0,0,0.3);
        }
        .stat-item { font-size: 13px; display: flex; align-items: center; gap: 8px; }
        .stat-label { color: var(--text-dim); }
        .stat-value { font-weight: 600; color: var(--accent-color); }
    </style>
</head>
<body>
    <div class="toolbar">
        <div class="logo">🐙 Autopus 제어 센터</div>
        <div class="project-info" id="project-name">프로젝트 로딩 중...</div>
    </div>

    <div class="canvas" id="canvas"></div>

    <div class="bottom-bar">
        <div class="stat-item"><span class="stat-label">에이전트</span><span class="stat-value" id="agent-count">0</span></div>
        <div class="stat-item"><span class="stat-label">품질 모드</span><span class="stat-value" id="quality-mode">-</span></div>
        <div class="stat-item"><span class="stat-label">상태</span><span class="stat-value" style="color:var(--success)">연결됨</span></div>
    </div>

    <script>
        async function loadStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                
                document.getElementById('project-name').innerText = "현재 프로젝트: " + data.project;
                document.getElementById('agent-count').innerText = data.agents ? Object.keys(data.agents).length : 0;
                document.getElementById('quality-mode').innerText = data.quality ? data.quality.toUpperCase() : "-";
                
                const canvas = document.getElementById('canvas');
                canvas.innerHTML = '';
                
                let x = 100;
                let y = 150;
                
                const providers = data.agents || [];
                providers.forEach((p, index) => {
                    const node = document.createElement('div');
                    node.className = 'node';
                    node.style.left = x + 'px';
                    node.style.top = y + 'px';
                    
                    let modelName = "기본 모델";
                    if (p.args) {
                        const found = p.args.find(a => a.includes('gpt') || a.includes('gemini') || a.includes('opus'));
                        if (found) modelName = found;
                    }

                    node.innerHTML = '<div class="node-header">' +
                            '<div class="node-title">' + p.name.toUpperCase() + '</div>' +
                            '<div class="status-badge">대기 중</div>' +
                        '</div>' +
                        '<div class="node-body">' +
                            '<div class="info-row"><span class="info-label">모델</span><span class="info-value">' + modelName + '</span></div>' +
                            '<div class="info-row"><span class="info-label">플랫폼</span><span class="info-value">' + p.binary + '</span></div>' +
                            '<div class="info-row"><span class="info-label">모드</span><span class="info-value">Subprocess</span></div>' +
                        '</div>';
                    canvas.appendChild(node);
                    
                    if (index < providers.length - 1) {
                        const line = document.createElement('div');
                        line.className = 'connector-line';
                        line.style.left = (x + 260) + 'px';
                        line.style.top = (y + 45) + 'px';
                        line.style.width = '40px';
                        canvas.appendChild(line);
                    }
                    
                    x += 300;
                    y = (y === 150) ? 220 : 150;
                });
            } catch (e) {
                console.error("데이터 로드 실패:", e);
            }
        }
        
        loadStatus();
        setInterval(loadStatus, 5000);
    </script>
</body>
</html>
`
