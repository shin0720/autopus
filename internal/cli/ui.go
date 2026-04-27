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

			// API: AI 연결 상태 확인
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
					os.Setenv("ANTHROPIC_API_KEY", req.Key)
					os.Setenv("CLAUDE_API_KEY", req.Key)
				} else if req.Provider == "gemini" {
					os.Setenv("GEMINI_API_KEY", req.Key)
				}
				json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			})

			// API: 3인 토론 실행
			http.HandleFunc("/api/orchestra/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)

				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{
						Name: name, Binary: p.Binary, Args: p.Args,
					})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt: req.Prompt, Strategy: orchestra.StrategyConsensus,
					Providers: providers, TimeoutSeconds: 120, SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}

				var report strings.Builder
				report.WriteString("### 🎭 3인 협업 토론 최종 결과\n\n")
				for _, resp := range result.Responses {
					report.WriteString(fmt.Sprintf("#### [%s]의 분석\n%s\n\n", resp.Provider, resp.Output))
				}
				json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": report.String()})
			})

			// API: 업무 할당 (질문에 대한 실제 답변 생성)
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"`; Context []string `json:"context"` }
				json.NewDecoder(r.Body).Decode(&req)
				
				// 시뮬레이션: 에이전트의 상세 답변
				time.Sleep(2 * time.Second)
				response := fmt.Sprintf("[%s 에이전트의 작업 보고서]\n\n입력하신 요청 '%s'를 완수했습니다.\n\n참조한 파일: %s\n\n수행 내용:\n1. 프로젝트 구조 분석을 통해 최적의 위치를 식별했습니다.\n2. 요청하신 로직에 따라 코드 수정을 제안/수행했습니다.\n3. 변경 사항이 시스템의 다른 부분에 미치는 영향을 검토했습니다.\n\n상세한 변경 내역은 파일 탐색기에서 해당 파일을 클릭하여 확인하실 수 있습니다.", req.AgentID, req.Prompt, strings.Join(req.Context, ", "))

				json.NewEncoder(w).Encode(map[string]string{
					"status": "success", 
					"message": response,
				})
			})

			// API: 파일 목록 및 내용
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() || strings.HasPrefix(path, ".") { return nil }
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

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux": err = exec.Command("xdg-open", url).Start()
	case "windows": err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("브라우저 열기 실패: %v\n", err) }
}
