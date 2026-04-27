package cli

import (
	"context"
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

			fmt.Printf("🐙 Autopus Studio (Final) 시작 중... http://%s\n", addr)

			// API: AI 상태 체크 및 상세 에러 보고
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				health := make(map[string]interface{})
				health["claude"] = os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_API_KEY") != ""
				health["gemini"] = os.Getenv("GEMINI_API_KEY") != ""
				health["codex"] = true
				json.NewEncoder(w).Encode(health)
			})

			// API: 연쇄 워크플로우 실행
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { 
					AgentID string `json:"agentId"`; Prompt string `json:"prompt"`; Context []string `json:"context"` 
				}
				json.NewDecoder(r.Body).Decode(&req)

				// 1. 컨텍스트 구성
				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxFiles.WriteString(fmt.Sprintf("\n[File: %s]\n%s\n", p, string(data)))
				}

				// 2. 에이전트 전문 프롬프트
				finalPrompt := fmt.Sprintf("당신은 %s 전문가입니다. 다음 요청을 프로젝트 상황에 맞춰 해결하세요. 반드시 한국어로 답변하세요.\n\n[요청]\n%s\n\n[참조코드]\n%s", 
					req.AgentID, req.Prompt, ctxFiles.String())

				// 3. AI 엔진 실행
				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt: finalPrompt, Strategy: orchestra.StrategyFastest,
					Providers: providers, TimeoutSeconds: 180, SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "모든 AI 엔진이 응답하지 않습니다. API 키가 정확히 등록되었는지 확인하세요. (Error: " + err.Error() + ")"})
					return
				}

				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"output": result.Merged,
					"nextAgent": getNextAgentFull(req.AgentID),
				})
			})

			// API: 나머지 (기존과 동일)
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]interface{}{"project": cfg.ProjectName, "agents": cfg.Orchestra.Providers})
			})
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && !strings.HasPrefix(path, ".") { files = append(files, path) }
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
				json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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

// 16인 에이전트 전체 협업 지도
func getNextAgentFull(id string) string {
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
