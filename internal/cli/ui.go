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
			
			fmt.Printf("🐙 Autopus Studio v4.7 가동 중... http://%s\n", addr)

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
					os.MkdirAll(filepath.Dir(path), 0755); os.WriteFile(path, data, 0644); w.WriteHeader(http.StatusOK)
				}
			})

			// API: 실전 AI 업무 수행
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"`; Context []string `json:"context"` }
				json.NewDecoder(r.Body).Decode(&req)

				fmt.Printf("⛓️ [%s] 전문가 가동 시작\n", req.AgentID)

				cfg, _ := config.Load(".")
				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxFiles.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", p, string(data)))
				}

				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
				}

				finalPrompt := fmt.Sprintf("당신은 %s 전문가입니다. 코드를 분석하여 한국어로 상세히 답변하세요.\n\n[지시]\n%s\n\n[참조코드]%s", req.AgentID, req.Prompt, ctxFiles.String())
				
				orchCfg := orchestra.OrchestraConfig{
					Prompt: finalPrompt, Strategy: orchestra.StrategyFastest,
					Providers: providers, TimeoutSeconds: 180, SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
					return
				}

				json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "output": result.Merged})
			})

			// API: 기타 로직 유지
			http.HandleFunc("/api/workspace/change", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Path string `json:"path"` }
				json.NewDecoder(r.Body).Decode(&req)
				target := req.Path
				if strings.Contains(target, ":") { target = "/mnt/" + strings.ToLower(target[:1]) + strings.ReplaceAll(target[2:], "\\", "/") }
				os.Chdir(target); dir, _ := os.Getwd()
				json.NewEncoder(w).Encode(map[string]string{"status": "success", "currentDir": dir})
			})
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
	case "windows":
		err = exec.Command("cmd", "/c", "start", "chrome", "--app="+url).Start()
		if err != nil { err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() }
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("실행 실패: %v\n", err) }
}
