package orchestra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func attachCodexLastMessageCapture(req ProviderRequest, args []string) ([]string, string, func(), error) {
	if !shouldCaptureCodexLastMessage(req.Config, args) {
		return args, "", func() {}, nil
	}
	f, err := os.CreateTemp("", "codex-last-message-*.txt")
	if err != nil {
		return args, "", nil, fmt.Errorf("subprocess codex: create last-message temp file: %w", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		removePromptFile(path)
		return args, "", nil, fmt.Errorf("subprocess codex: close last-message temp file: %w", err)
	}
	insertAt := len(args)
	if req.Config.PromptViaArgs {
		if promptIndex := lastPromptArgIndex(args, req.Prompt); promptIndex >= 0 {
			insertAt = promptIndex
		}
	}
	args = insertArgs(args, insertAt, "--output-last-message", path)
	return args, path, func() { removePromptFile(path) }, nil
}

func shouldCaptureCodexLastMessage(cfg ProviderConfig, args []string) bool {
	if !isCodexExecProvider(cfg, args) {
		return false
	}
	return !hasAnyArg(args, "--output-last-message", "-o")
}

func isCodexExecProvider(cfg ProviderConfig, args []string) bool {
	if len(args) == 0 || args[0] != "exec" {
		return false
	}
	name := strings.TrimSpace(cfg.Name)
	binary := filepath.Base(strings.TrimSpace(cfg.Binary))
	return strings.EqualFold(name, "codex") || strings.EqualFold(binary, "codex")
}

func hasAnyArg(args []string, names ...string) bool {
	for _, arg := range args {
		for _, name := range names {
			if arg == name {
				return true
			}
		}
	}
	return false
}

func lastPromptArgIndex(args []string, prompt string) int {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == prompt {
			return i
		}
	}
	return -1
}

func insertArgs(args []string, index int, values ...string) []string {
	if index < 0 || index > len(args) {
		index = len(args)
	}
	next := make([]string, 0, len(args)+len(values))
	next = append(next, args[:index]...)
	next = append(next, values...)
	next = append(next, args[index:]...)
	return next
}

func appendSubprocessDiagnostic(existing, diagnostic string) string {
	diagnostic = strings.TrimSpace(diagnostic)
	if diagnostic == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return diagnostic
	}
	return existing + "\n" + diagnostic
}
