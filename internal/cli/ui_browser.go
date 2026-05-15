package cli

import (
	"os"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:errcheck
	case "darwin":
		exec.Command("open", url).Start() //nolint:errcheck
	case "windows":
		// Try Chrome in app mode first, then Edge, then system default.
		chromeCandidates := []string{
			os.ExpandEnv(`${LOCALAPPDATA}\Google\Chrome\Application\chrome.exe`),
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
		for _, chromePath := range chromeCandidates {
			if _, statErr := os.Stat(chromePath); statErr == nil {
				if err := exec.Command(chromePath, "--app="+url, "--disable-extensions").Start(); err == nil {
					return
				}
			}
		}
		edgeCandidates := []string{
			os.ExpandEnv(`${LOCALAPPDATA}\Microsoft\Edge\Application\msedge.exe`),
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		}
		for _, edgePath := range edgeCandidates {
			if _, statErr := os.Stat(edgePath); statErr == nil {
				if err := exec.Command(edgePath, "--app="+url, "--disable-extensions").Start(); err == nil {
					return
				}
			}
		}
		fallbackCmd := exec.Command("cmd", "/c", "start", "", url)
		hideConsoleWindow(fallbackCmd)
		fallbackCmd.Start() //nolint:errcheck
	}
}

func workflowAgentName(agentID string) string {
	if name, ok := workflowAgentDisplayName(agentID); ok {
		return name
	}
	return agentID
}

func workflowAgentDisplayName(agentID string) (string, bool) {
	names := map[string]string{
		"planner": "Planner", "spec": "Spec Writer", "arch": "Architect", "expl": "Explorer",
		"exec": "Executor", "deep": "Deep Worker", "dbug": "Debugger", "anno": "Annotator",
		"test": "Tester", "val": "Validator", "fend": "Frontend", "uxv": "UX Validator",
		"perf": "Perf Eng", "rev": "Reviewer", "sec": "Security", "devops": "DevOps",
	}
	name, ok := names[agentID]
	return name, ok
}
