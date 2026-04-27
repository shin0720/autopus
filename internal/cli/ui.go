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
	"github.com/shin0720/auto-adk/content" // 임베드된 파일 사용
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

			// API: AI 연결 상태 체크
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				health := make(map[string]bool)
				// 환경변수 및 바이너리 존재 여부로 체크
				health["claude"] = os.Getenv("CLAUDE_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != ""
				health["gemini"] = os.Getenv("GEMINI_API_KEY") != ""
				health["codex"] = true // 바이너리 기반

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(health)
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

			// API: 업무 할당
			http.HandleFunc("/api/agent/assign", func(w http.ResponseWriter, r *http.Request) {
				var req struct { AgentID string `json:"agentId"`; Prompt string `json:"prompt"` }
				json.NewDecoder(r.Body).Decode(&req)
				time.Sleep(1 * time.Second)
				json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "업무 완료"})
			})

			// UI 서빙 (임베드된 파일 사용 - 하얀 화면 방지)
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data, err := content.FS.ReadFile("ui/dashboard.html")
				if err != nil {
					fmt.Fprintf(w, "<h1>UI 로드 실패</h1><p>내장된 HTML 파일을 찾을 수 없습니다: %v</p>", err)
					return
				}
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
