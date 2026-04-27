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
	"github.com/shin0720/auto-adk/content"
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
					if strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") || strings.Contains(path, "vendor") || strings.Contains(path, "__pycache__") {
						return nil
					}
					files = append(files, path)
					return nil
				})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(files)
			})

			// API: 파일 내용 읽기
			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path")
				if path == "" { return }
				content, err := os.ReadFile(path)
				if err != nil {
					http.Error(w, "파일을 읽을 수 없습니다", 500)
					return
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Write(content)
			})

			// API: 업무 할당
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { 
					AgentID string `json:"agentId"`
					Prompt string `json:"prompt"`
					Context []string `json:"context"`
				}
				json.NewDecoder(r.Body).Decode(&req)
				fmt.Printf("📝 [%s] 업무 할당 (참조파일 %d개): %s\n", req.AgentID, len(req.Context), req.Prompt)
				time.Sleep(1 * time.Second)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "에이전트가 작업을 수락했습니다."})
			})

			// UI Main Page
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				html, err := content.FS.ReadFile("ui/dashboard.html")
				if err != nil {
					http.Error(w, "대시보드 파일을 찾을 수 없습니다", 404)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(html)
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
