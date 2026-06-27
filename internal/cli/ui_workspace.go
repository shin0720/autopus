package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveWorkspacePath(target string) (string, error) {
	currentDir := getWorkspaceDir()
	if currentDir == "" {
		var err error
		currentDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return resolveWorkspacePathForOS(runtime.GOOS, currentDir, target)
}

func resolveWorkspacePathForOS(goos, currentDir, target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "." {
		return filepath.Abs(currentDir)
	}

	if goos == "windows" {
		windowsPath := normalizeWindowsPath(trimmed)
		if windowsPath != "" {
			return filepath.Abs(windowsPath)
		}
	}

	if goos != "windows" && strings.Contains(trimmed, ":") {
		drive := strings.ToLower(trimmed[:1])
		rest := ""
		if len(trimmed) > 2 {
			rest = strings.ReplaceAll(trimmed[2:], "\\", "/")
		}
		return filepath.Abs("/mnt/" + drive + rest)
	}

	return filepath.Abs(filepath.Join(currentDir, trimmed))
}

func normalizeWindowsPath(target string) string {
	if strings.HasPrefix(target, "/mnt/") && len(target) >= 6 {
		drive := strings.ToUpper(target[5:6])
		rest := strings.ReplaceAll(strings.TrimPrefix(target[6:], "/"), "/", `\`)
		if rest == "" {
			return drive + `:\`
		}
		return drive + `:\` + rest
	}

	if len(target) == 2 && target[1] == ':' {
		return target + `\`
	}

	if len(target) >= 3 && target[1] == ':' && (target[2] == '\\' || target[2] == '/') {
		return strings.ReplaceAll(target, "/", `\`)
	}

	return ""
}

func visibleFolders(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	folders := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		}
	}
	return folders, nil
}

func driveRoots() []string {
	if runtime.GOOS != "windows" {
		return []string{"/"}
	}

	roots := []string{}
	for _, drive := range []string{"C:\\", "D:\\", "E:\\", "F:\\"} {
		if _, err := os.Stat(drive); err == nil {
			roots = append(roots, drive)
		}
	}
	return roots
}

func workspaceListPayload(root string) (map[string]interface{}, error) {
	folders, err := visibleFolders(root)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"current": root,
		"folders": folders,
		"parent":  filepath.Dir(root),
		"roots":   driveRoots(),
	}
	if runtime.GOOS == "windows" {
		payload["parent"] = parentOrSelf(root)
	}
	return payload, nil
}

func parentOrSelf(path string) string {
	parent := filepath.Dir(path)
	if sameFold(parent, path) {
		return path
	}
	return parent
}

func sameFold(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func readWorkspaceFile(root, requestedPath string) ([]byte, error) {
	if strings.TrimSpace(requestedPath) == "" {
		return nil, fmt.Errorf("path is required")
	}

	resolved := requestedPath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, requestedPath)
	}
	resolved = filepath.Clean(resolved)
	return os.ReadFile(resolved)
}
