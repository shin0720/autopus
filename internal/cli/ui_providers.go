package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/insajin/autopus-adk/pkg/config"
)

type uiProviderStatus struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Mode      string `json:"mode"`
	Installed bool   `json:"installed"`
	Runnable  bool   `json:"runnable"`
	Connected bool   `json:"connected"`
	Version   string `json:"version,omitempty"`
	Binary    string `json:"binary,omitempty"`
	Quota     string `json:"quota,omitempty"`
	Issue     string `json:"issue,omitempty"`
}

type cachedProviderStatus struct {
	status    uiProviderStatus
	expiresAt time.Time
}

var (
	providerCacheMu sync.Mutex
	providerCache   = make(map[string]cachedProviderStatus)
)

func handleProviderStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := collectProviderStatuses(uiProjectRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statuses)
}

func handleProviderConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	providerID := strings.TrimSpace(req.Provider)
	if providerID == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}

	resolved, err := resolveRunnableBinary(providerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := launchProviderCLI(resolved); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cache so next poll reflects fresh login state.
	providerCacheMu.Lock()
	delete(providerCache, providerID)
	providerCacheMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func collectProviderStatuses(root string) ([]uiProviderStatus, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	_, _ = config.MigrateOrchestraConfig(cfg)

	ordered := []struct {
		id    string
		label string
	}{
		{id: "claude", label: "Claude"},
		{id: "codex", label: "Codex"},
		{id: "gemini", label: "Gemini"},
	}

	type indexedResult struct {
		index  int
		status uiProviderStatus
	}
	resultCh := make(chan indexedResult, len(ordered))

	for i, item := range ordered {
		i, item := i, item
		entry, ok := cfg.Orchestra.Providers[item.id]
		if !ok {
			resultCh <- indexedResult{i, uiProviderStatus{
				ID: item.id, Label: item.label, Mode: "CLI",
				Issue: "autopus.yaml에 provider 설정이 없습니다.", Quota: "N/A",
			}}
			continue
		}

		providerCacheMu.Lock()
		cached, hit := providerCache[item.id]
		providerCacheMu.Unlock()
		if hit && time.Now().Before(cached.expiresAt) {
			resultCh <- indexedResult{i, cached.status}
			continue
		}

		// Probe in parallel so 3 providers don't block sequentially.
		go func() {
			s := probeProviderStatus(item.id, item.label, entry)
			ttl := 30 * time.Second
			if s.Connected {
				ttl = 5 * time.Minute
			}
			providerCacheMu.Lock()
			providerCache[item.id] = cachedProviderStatus{status: s, expiresAt: time.Now().Add(ttl)}
			providerCacheMu.Unlock()
			resultCh <- indexedResult{i, s}
		}()
	}

	statuses := make([]uiProviderStatus, len(ordered))
	for range ordered {
		r := <-resultCh
		statuses[r.index] = r.status
	}
	return statuses, nil
}

func probeProviderStatus(id, label string, entry config.ProviderEntry) uiProviderStatus {
	status := uiProviderStatus{
		ID:    id,
		Label: label,
		Mode:  "CLI",
		Quota: "CLI 미지원",
	}

	resolved, err := resolveRunnableBinary(entry.Binary)
	if err != nil {
		status.Issue = err.Error()
		return status
	}

	status.Installed = true
	status.Binary = resolved
	version, runErr := providerVersion(resolved)
	if runErr != nil {
		status.Issue = runErr.Error()
		return status
	}

	status.Runnable = true
	status.Connected = true
	status.Version = version
	return status
}

func resolveRunnableBinary(name string) (string, error) {
	if runtime.GOOS != "windows" {
		path, err := exec.LookPath(name)
		if err != nil {
			return "", fmt.Errorf("%s CLI를 찾지 못했습니다", name)
		}
		return path, nil
	}

	whereCmd := exec.Command("where.exe", name)
	hideConsoleWindow(whereCmd)
	out, err := whereCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s CLI를 찾지 못했습니다", name)
	}

	candidates := strings.Fields(strings.ReplaceAll(string(out), "\r", ""))
	if len(candidates) == 0 {
		return "", fmt.Errorf("%s CLI를 찾지 못했습니다", name)
	}

	for _, candidate := range candidates {
		lower := strings.ToLower(candidate)
		if strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".bat") {
			return candidate, nil
		}
	}

	return candidates[0], nil
}

// providerVersion runs the binary with --version and enforces a hard 4-second
// deadline using a goroutine + explicit Kill, because on Windows exec.CommandContext
// may not terminate child processes reliably when the context is cancelled.
func providerVersion(binary string) (string, error) {
	type outcome struct {
		out []byte
		err error
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", binary, "--version")
	} else {
		cmd = exec.Command(binary, "--version")
	}
	hideConsoleWindow(cmd)

	ch := make(chan outcome, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		ch <- outcome{out, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			msg := strings.TrimSpace(string(r.out))
			if msg == "" {
				msg = r.err.Error()
			}
			return "", fmt.Errorf("CLI 실행 실패: %s", msg)
		}
		return strings.TrimSpace(firstLine(string(r.out))), nil
	case <-time.After(4 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", fmt.Errorf("CLI 버전 확인 timeout")
	}
}

func launchProviderCLI(binary string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd.exe", "/c", "start", "", "cmd.exe", "/k", binary).Start()
	}

	return exec.Command("x-terminal-emulator", "-e", binary).Start()
}

func firstLine(text string) string {
	releases := []*regexp.Regexp{
		regexp.MustCompile(`codex-cli\s+[0-9]+\.[0-9]+\.[0-9]+`),
		regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+\s+\(Claude Code\)`),
	}
	for _, pattern := range releases {
		if match := pattern.FindString(text); match != "" {
			return match
		}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	lastNonEmpty := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lastNonEmpty = trimmed
			if !strings.HasPrefix(strings.ToUpper(trimmed), "WARNING:") {
				return trimmed
			}
		}
	}
	return lastNonEmpty
}
