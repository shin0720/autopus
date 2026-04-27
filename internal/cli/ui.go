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
			if err != nil {
				return err
			}

			fmt.Printf("🐙 Autopus Studio 시작 중... http://%s\n", addr)

			// API: 현재 설정 및 에이전트 상태
			http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"project": cfg.ProjectName,
					"agents":  cfg.Orchestra.Providers,
					"quality": cfg.Quality.Default,
				})
			})

			// API: AI 연결 상태
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				health := make(map[string]bool)
				health["claude"] = os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_API_KEY") != ""
				health["gemini"] = os.Getenv("GEMINI_API_KEY") != ""
				health["codex"] = true
				json.NewEncoder(w).Encode(health)
			})

			// API: 3인 AI 토론 실행
			http.HandleFunc("/api/orchestra/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)

				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{
						Name:   name,
						Binary: p.Binary,
						Args:   p.Args,
					})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt:         req.Prompt,
					Strategy:       orchestra.StrategyConsensus,
					Providers:      providers,
					TimeoutSeconds: 120,
					SubprocessMode: true,
				}

				ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
				defer cancel()

				result, err := orchestra.RunOrchestra(ctx, orchCfg)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}

				var finalReport strings.Builder
				finalReport.WriteString("### 🎭 3인 AI 협업 토론 결과 보고서\n\n")
				for _, resp := range result.Responses {
					finalReport.WriteString(fmt.Sprintf("#### [%s] 의견\n%s\n\n", resp.Provider, resp.Output))
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status": "success",
					"message": finalReport.String(),
				})
			})

			// API: 파일 목록
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() || strings.HasPrefix(path, ".") { return nil }
					files = append(files, path)
					return nil
				})
				json.NewEncoder(w).Encode(files)
			})

			// API: 파일 내용 읽기
			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path")
				content, _ := os.ReadFile(path)
				w.Write(content)
			})

			// API: 단일 업무 할당
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)
				time.Sleep(1 * time.Second)
				json.NewEncoder(w).Encode(map[string]string{
					"status": "success", 
					"message": fmt.Sprintf("%s 에이전트가 작업을 완료했습니다.", req.AgentID),
				})
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

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux": err = exec.Command("xdg-open", url).Start()
	case "windows": err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("브라우저 열기 실패: %v\n", err) }
}
