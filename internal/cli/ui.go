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
			if err != nil {
				return err
			}

			fmt.Printf("🐙 Autopus Virtual Studio 실행 중... http://%s\n", addr)

			// API: 현재 설정 및 에이전트 상태
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"project": cfg.ProjectName,
					"agents":  cfg.Orchestra.Providers,
					"quality": cfg.Quality.Default,
				})
			})

			// API: AI 연결 상태 체크
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				health := make(map[string]bool)
				health["claude"] = os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_API_KEY") != ""
				health["gemini"] = os.Getenv("GEMINI_API_KEY") != ""
				health["codex"] = true
				json.NewEncoder(w).Encode(health)
			})

			// API: API 키 등록
			http.HandleFunc("/api/providers/keys", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Provider string `json:"provider"`; Key string `json:"key"` }
				json.NewDecoder(r.Body).Decode(&req)
				if req.Provider == "claude" {
					os.Setenv("ANTHROPIC_API_KEY", req.Key); os.Setenv("CLAUDE_API_KEY", req.Key)
				} else if req.Provider == "gemini" {
					os.Setenv("GEMINI_API_KEY", req.Key)
				}
				json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			})

			// API: 연쇄 워크플로우 실행 (Chained Execution)
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { 
					AgentID string `json:"agentId"`
					Prompt  string `json:"prompt"`
					Context []string `json:"context"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				fmt.Printf("⛓️ 워크플로우 시작 [%s]: %s\n", req.AgentID, req.Prompt)

				// 1. 컨텍스트 로드
				var ctxContent strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxContent.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", p, string(data)))
				}

				// 2. 에이전트 실행 (진짜 AI)
				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{
						Name: name, Binary: p.Binary, Args: p.Args,
					})
				}

				finalPrompt := fmt.Sprintf("당신은 %s 전문가입니다. 다음 요청을 수행하세요.\n\n요청: %s\n\n참조코드: %s", req.AgentID, req.Prompt, ctxContent.String())
				
				orchCfg := orchestra.OrchestraConfig{
					Prompt: finalPrompt, Strategy: orchestra.StrategyFastest,
					Providers: providers, TimeoutSeconds: 120, SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
					return
				}

				// 3. 결과 반환 (이 결과가 프론트엔드에서 다음 노드로 전달됨)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"output": result.Merged,
					"nextAgent": getNextAgent(req.AgentID), // 다음으로 일할 사람 자동 지정
				})
			})

			// API: 파일 목록 및 읽기
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() || strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") { return nil }
					files = append(files, path)
					return nil
				})
				json.NewEncoder(w).Encode(files)
			})
			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path")
				content, _ := os.ReadFile(path)
				w.Write(content)
			})

			// UI 서빙
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

// getNextAgent는 워크플로우 상의 다음 에이전트를 반환한다.
func getNextAgent(currentID string) string {
	workflow := map[string]string{
		"planner": "spec",
		"spec":    "arch",
		"arch":    "exec",
		"exec":    "test",
		"test":    "val",
		"val":     "rev",
	}
	return workflow[currentID]
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
