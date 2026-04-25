package cli

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// newUICmd는 웹 UI 서버를 실행하는 ui 서브커맨드를 생성한다.
func newUICmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Autopus 시각적 대시보드 실행",
		Long:  "n8n 스타일의 노드 기반 워크플로우 대시보드를 실행합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("localhost:%d", port)
			fmt.Printf("🐙 Autopus Dashboard 시작 중... http://%s\n", addr)

			// 브라우저 자동 열기
			openBrowser("http://" + addr)

			// 임시 메시지 서버
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "<h1>🐙 Autopus Dashboard (n8n Style)</h1><p>현재 대시보드 기능을 구축 중입니다. 에이전트 상태와 워크플로우 시각화가 여기에 표시됩니다.</p>")
			})

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
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		fmt.Printf("브라우저를 열 수 없습니다: %v\n", err)
	}
}
