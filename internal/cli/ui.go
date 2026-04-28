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
	"github.com/shin0720/auto-adk/pkg/orchestra"
)

// newUICmd는 웹 UI 서버를 실행하는 ui 서브커맨드를 생성한다.
func newUICmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Autopus 시각적 대시보드 실행",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("localhost:%d", port)
			
			// 전역 변수로 현재 작업 디렉토리 관리
			currentDir, _ := os.Getwd()

			fmt.Printf("🐙 Autopus Studio v3.8 시작 중... http://%s\n", addr)

			// API: 작업 디렉토리 변경 (드라이브 이동)
			http.HandleFunc("/api/workspace/change", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Path string `json:"path"` }
				json.NewDecoder(r.Body).Decode(&req)
				
				targetPath := req.Path
				// WSL 특화 경로 변환 (C:, E: 등)
				if strings.Contains(targetPath, ":") {
					drive := strings.ToLower(targetPath[:1])
					targetPath = "/mnt/" + drive + strings.ReplaceAll(targetPath[2:], "\\", "/")
				}

				if _, err := os.Stat(targetPath); err == nil {
					currentDir = targetPath
					os.Chdir(currentDir)
					json.NewEncoder(w).Encode(map[string]string{"status": "success", "currentDir": currentDir})
				} else {
					http.Error(w, "경로를 찾을 수 없습니다: "+targetPath, 404)
				}
			})

			// API: 드라이브 및 상위 폴더 목록
			http.HandleFunc("/api/workspace/list", func(w http.ResponseWriter, r *http.Request) {
				entries, _ := os.ReadDir(currentDir)
				var folders []string
				for _, e := range entries {
					if e.IsDir() && !strings.HasPrefix(e.Name(), ".") { folders = append(folders, e.Name()) }
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"current": currentDir,
					"folders": folders,
					"parent": filepath.Dir(currentDir),
				})
			})

			// API: 연쇄 워크플로우 실행
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"`; Context []string `json:"context"` }
				json.NewDecoder(r.Body).Decode(&req)
				
				cfg, _ := config.Load(currentDir)
				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(filepath.Join(currentDir, p))
					ctxFiles.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", p, string(data)))
				}

				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt: fmt.Sprintf("역할: %s\n요청: %s\n분석내용: %s", req.AgentID, req.Prompt, ctxFiles.String()),
					Strategy: orchestra.StrategyFastest,
					Providers: providers, TimeoutSeconds: 180, SubprocessMode: true,
				}
				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "output": result.Merged})
			})

			// API: 파일 목록
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && !strings.HasPrefix(path, ".") && !strings.Contains(path, "node_modules") { files = append(files, path) }
					return nil
				})
				json.NewEncoder(w).Encode(files)
			})
			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path"); content, _ := os.ReadFile(path); w.Write(content)
			})
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				h := map[string]bool{"claude": os.Getenv("CLAUDE_API_KEY")!="", "gemini": os.Getenv("GEMINI_API_KEY")!="", "codex": true}
				json.NewEncoder(w).Encode(h)
			})

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data, _ := content.FS.ReadFile("ui/dashboard.html")
				w.Write(data)
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
