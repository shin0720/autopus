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

	"github.com/shin0720/auto-adk/content"
	"github.com/shin0720/auto-adk/pkg/config"
	"github.com/shin0720/auto-adk/pkg/orchestra"
	"github.com/spf13/cobra"
)

const workflowStatePath = ".autopus/workflows/state.json"

// WorkflowState holds the persistent UI workflow state.
type WorkflowState struct {
	ProjectName string       `json:"projectName"`
	LastUpdated time.Time    `json:"lastUpdated"`
	Nodes       []NodeState  `json:"nodes"`
	Connections []Connection `json:"connections"`
	Logs        []LogEntry   `json:"logs"`
}

// NodeState holds per-node position and execution state.
// X and Y use RawMessage to accept both numeric (100) and string ("100px") JSON values.
type NodeState struct {
	ID     string          `json:"id"`
	X      json.RawMessage `json:"x,omitempty"`
	Y      json.RawMessage `json:"y,omitempty"`
	Status string          `json:"status,omitempty"`
	Output string          `json:"output,omitempty"`
}

// Connection represents a directed edge between two nodes.
type Connection struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// LogEntry records a single terminal log line.
type LogEntry struct {
	Agent string `json:"agent"`
	Msg   string `json:"msg"`
	Time  string `json:"time"`
}

// workflowStateHandler handles GET and POST /api/workflow/state.
func workflowStateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(workflowStatePath)
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			empty := WorkflowState{
				Nodes:       []NodeState{},
				Connections: []Connection{},
				Logs:        []LogEntry{},
			}
			_ = json.NewEncoder(w).Encode(empty)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var state WorkflowState
		if err := json.Unmarshal(data, &state); err != nil {
			http.Error(w, "invalid state file: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	case http.MethodPost:
		var state WorkflowState
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.LastUpdated = time.Now().UTC()
		if state.ProjectName == "" {
			if cwd, err := os.Getwd(); err == nil {
				state.ProjectName = filepath.Base(cwd)
			}
		}
		if err := os.MkdirAll(filepath.Dir(workflowStatePath), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(workflowStatePath, out, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "saved",
			"path":   workflowStatePath,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// newUICmd는 웹 UI 서버를 실행하는 ui 서브커맨드를 생성한다.
func newUICmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Autopus 시각적 대시보드 실행",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("localhost:%d", port)
			fmt.Printf("🐙 Autopus Studio v4.9 시작 중... http://%s\n", addr)

			// API: 작업 디렉토리 강제 전환 (C:, E: 완벽 지원)
			http.HandleFunc("/api/workspace/change", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Path string `json:"path"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				target := req.Path
				if strings.Contains(target, ":") {
					drive := strings.ToLower(target[:1])
					target = "/mnt/" + drive + strings.ReplaceAll(target[2:], "\\", "/")
				}
				absPath, _ := filepath.Abs(target)
				if err := os.Chdir(absPath); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				dir, _ := os.Getwd()
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "currentDir": dir})
			})

			// API: 실전 AI 업무 수행
			http.HandleFunc("/api/workflow/run", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					AgentID string   `json:"agentId"`
					Prompt  string   `json:"prompt"`
					Context []string `json:"context"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				cfg, err := config.Load(".")
				if err != nil {
					_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Config load failed: " + err.Error()})
					return
				}

				var ctxFiles strings.Builder
				for _, p := range req.Context {
					data, _ := os.ReadFile(p)
					ctxFiles.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", p, string(data)))
				}

				var providers []orchestra.ProviderConfig
				for name, p := range cfg.Orchestra.Providers {
					providers = append(providers, orchestra.ProviderConfig{Name: name, Binary: p.Binary, Args: p.Args})
				}

				orchCfg := orchestra.OrchestraConfig{
					Prompt:         fmt.Sprintf("역할: %s\n지시: %s\n코드: %s", req.AgentID, req.Prompt, ctxFiles.String()),
					Strategy:       orchestra.StrategyFastest,
					Providers:      providers,
					TimeoutSeconds: 180,
					SubprocessMode: true,
				}

				result, err := orchestra.RunOrchestra(r.Context(), orchCfg)
				if err != nil {
					_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
					return
				}

				_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "output": result.Merged})
			})

			// API: 워크플로우 상태 저장 / 로드
			http.HandleFunc("/api/workflow/state", workflowStateHandler)

			// API: 폴더/파일 목록
			http.HandleFunc("/api/workspace/list", func(w http.ResponseWriter, r *http.Request) {
				dir, _ := os.Getwd()
				entries, _ := os.ReadDir(".")
				var folders []string
				for _, e := range entries {
					if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
						folders = append(folders, e.Name())
					}
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"current": dir, "folders": folders, "parent": filepath.Dir(dir)})
			})
			http.HandleFunc("/api/files/list", func(w http.ResponseWriter, r *http.Request) {
				var files []string
				_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && !strings.HasPrefix(p, ".") && !strings.Contains(p, "node_modules") {
						files = append(files, p)
					}
					return nil
				})
				_ = json.NewEncoder(w).Encode(files)
			})
			http.HandleFunc("/api/files/read", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path")
				fileContent, _ := os.ReadFile(path)
				_, _ = w.Write(fileContent)
			})

			// API: 헬스체크 및 키 등록
			http.HandleFunc("/api/providers/health", func(w http.ResponseWriter, r *http.Request) {
				h := map[string]bool{
					"claude": os.Getenv("CLAUDE_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != "",
					"gemini": os.Getenv("GEMINI_API_KEY") != "",
					"codex":  true,
				}
				_ = json.NewEncoder(w).Encode(h)
			})
			http.HandleFunc("/api/providers/keys", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Provider string `json:"provider"`
					Key      string `json:"key"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if req.Provider == "claude" {
					_ = os.Setenv("CLAUDE_API_KEY", req.Key)
					_ = os.Setenv("ANTHROPIC_API_KEY", req.Key)
				}
				if req.Provider == "gemini" {
					_ = os.Setenv("GEMINI_API_KEY", req.Key)
				}
				w.WriteHeader(http.StatusOK)
			})

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data, _ := content.FS.ReadFile("ui/dashboard.html")
				_, _ = w.Write(data)
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
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("cmd", "/c", "start", "chrome", "--app="+url).Start()
		if err != nil {
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		}
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		fmt.Printf("브라우저 열기 실패: %v\n", err)
	}
}
