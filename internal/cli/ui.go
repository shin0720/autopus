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
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("localhost:%d", port)
			
			fmt.Printf("🐙 Autopus Studio v4.1 PRO 가동 중... http://%s\n", addr)

			// API: 워크플로우 상태 관리
			http.HandleFunc("/api/workflow/state", func(w http.ResponseWriter, r *http.Request) {
				path := ".autopus/workflows/state.json"
				if r.Method == http.MethodGet {
					data, _ := os.ReadFile(path)
					if len(data) == 0 { data = []byte("{}") }
					w.Header().Set("Content-Type", "application/json"); w.Write(data)
				} else {
					var state interface{}
					json.NewDecoder(r.Body).Decode(&state)
					data, _ := json.MarshalIndent(state, "", "  ")
					os.MkdirAll(filepath.Dir(path), 0755)
					os.WriteFile(path, data, 0644)
					w.WriteHeader(http.StatusOK)
				}
			})

			// API: 작업 디렉토리 강제 전환
			http.HandleFunc("/api/workspace/change", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Path string `json:"path"` }
				json.NewDecoder(r.Body).Decode(&req)
				target := req.Path
				if strings.Contains(target, ":") { 
					drive := strings.ToLower(target[:1])
					target = "/mnt/" + drive + strings.ReplaceAll(target[2:], "\\", "/")
				}
				os.Chdir(target)
				dir, _ := os.Getwd()
				json.NewEncoder(w).Encode(map[string]string{"status": "success", "currentDir": dir})
			})

			// API: 파일 목록
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				dir, _ := os.Getwd()
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && !strings.HasPrefix(path, ".") && !strings.Contains(path, "node_modules") {
						files = append(files, path)
					}
					return nil
				})
				json.NewEncoder(w).Encode(map[string]interface{}{"dir": dir, "files": files})
			})

			// API: AI 헬스체크
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				h := map[string]bool{
					"claude": os.Getenv("CLAUDE_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != "",
					"gemini": os.Getenv("GEMINI_API_KEY") != "",
					"codex": true,
				}
				json.NewEncoder(w).Encode(h)
			})

			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path"); content, _ := os.ReadFile(path); w.Write(content)
			})

			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				cfg, _ := config.Load(".")
				json.NewEncoder(w).Encode(map[string]string{"project": cfg.ProjectName})
			})

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data, _ := content.FS.ReadFile("ui/dashboard.html"); w.Write(data)
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
