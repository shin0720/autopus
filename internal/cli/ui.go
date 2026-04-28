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
			cfg, err := config.Load(".")
			if err != nil { return err }

			fmt.Printf("🐙 Autopus Studio v3.4 시작 중... http://%s\n", addr)

			// API: AI 헬스체크
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				health := make(map[string]bool)
				health["claude"] = os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_API_KEY") != ""
				health["gemini"] = os.Getenv("GEMINI_API_KEY") != ""
				health["codex"] = true
				json.NewEncoder(w).Encode(health)
			})

			// API: 연쇄 워크플로우
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"`; Context []string `json:"context"` }
				json.NewDecoder(r.Body).Decode(&req)

				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxFiles.WriteString(fmt.Sprintf("\n[File: %s]\n%s\n", p, string(data)))
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt: fmt.Sprintf("역할: %s\n요청: %s\n내용: %s", req.AgentID, req.Prompt, ctxFiles.String()),
					Strategy: orchestra.StrategyFastest,
					Providers: []orchestra.ProviderConfig{},
					TimeoutSeconds: 180, SubprocessMode: true,
				}
				for name, p := range cfg.Orchestra.Providers {
					orchCfg.Providers = append(orchCfg.Providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success", "output": result.Merged, "nextAgent": getNextAgentMapFinal(req.AgentID),
				})
			})

			// API: 기타 보조 기능
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{"project": cfg.ProjectName})
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
			http.HandleFunc("/api/providers/keys", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Provider string `json:"provider"`; Key string `json:"key"` }
				json.NewDecoder(r.Body).Decode(&req)
				if req.Provider == "claude" { os.Setenv("ANTHROPIC_API_KEY", req.Key); os.Setenv("CLAUDE_API_KEY", req.Key) }
				if req.Provider == "gemini" { os.Setenv("GEMINI_API_KEY", req.Key) }
				w.WriteHeader(http.StatusOK)
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

func getNextAgentMapFinal(id string) string {
	m := map[string]string{
		"planner": "spec", "spec": "arch", "arch": "expl", "expl": "exec",
		"exec": "deep", "deep": "dbug", "dbug": "anno", "anno": "test",
		"test": "val", "val": "fend", "fend": "uxv", "uxv": "perf",
		"perf": "rev", "rev": "sec", "sec": "devops",
	}
	return m[id]
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
