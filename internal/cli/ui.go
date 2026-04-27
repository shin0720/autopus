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
			if err != nil {
				return err
			}

			fmt.Printf("🐙 Autopus Studio (v2.0) 시작 중... http://%s\n", addr)

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

			// API: 실제 AI 구동 (에이전트별 분석 답변 생성)
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { 
					AgentID string `json:"agentId"`
					Prompt  string `json:"prompt"`
					Context []string `json:"context"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				fmt.Printf("🚀 [%s] AI 분석 요청 수신: %s\n", req.AgentID, req.Prompt)

				// 1. 컨텍스트 파일 읽기
				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxFiles.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", p, string(data)))
				}

				// 2. 에이전트 역할 주입 프롬프트
				fullPrompt := fmt.Sprintf("당신은 소프트웨어 개발팀의 '%s' 전문가입니다. 현재 프로젝트 코드를 분석하고 사용자의 요청에 전문적으로 답변하세요. 반드시 한국어로 답변하세요.\n\n[사용자 요청]\n%s\n\n[참조된 프로젝트 코드]%s", 
					req.AgentID, req.Prompt, ctxFiles.String())

				// 3. 실제 오케스트라 엔진 실행
				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{
						Name: name, Binary: p.Binary, Args: p.Args,
					})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt: fullPrompt, Strategy: orchestra.StrategyFastest,
					Providers: providers, TimeoutSeconds: 120, SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "에러: " + err.Error()})
					return
				}

				// 4. AI가 생성한 실제 텍스트 반환
				json.NewEncoder(w).Encode(map[string]string{
					"status": "success",
					"message": result.Merged,
				})
			})

			// API: 3인 협업 토론 (Orchestra)
			http.HandleFunc("/api/orchestra/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct { Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)
				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
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
				json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": result.Merged})
			})

			// API: 파일 목록 및 내용
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

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux": err = exec.Command("xdg-open", url).Start()
	case "windows": err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("브라우저 열기 실패: %v\n", err) }
}
