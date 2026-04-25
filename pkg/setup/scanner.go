package setup

import (
	"os"
	"path/filepath"
	"strings"
)

const maxDepth = 3

// Scan analyzes a project directory and returns ProjectInfo.
func Scan(projectDir string) (*ProjectInfo, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		Name:    filepath.Base(absDir),
		RootDir: absDir,
	}

	info.Languages = detectLanguages(absDir)
	info.Frameworks = detectFrameworks(absDir)
	info.BuildFiles = detectBuildFiles(absDir)
	info.EntryPoints = detectEntryPoints(absDir, info.Languages)
	info.TestConfig = detectTestConfig(absDir, info.Languages, info.BuildFiles)
	info.Structure = scanDirectoryTree(absDir, 0)
	info.Conventions = AnalyzeConventions(absDir, info.Languages)
	info.Workspaces = DetectWorkspaces(absDir)

	return info, nil
}

func scanDirectoryTree(dir string, depth int) []DirEntry {
	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var tree []DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden dirs and common non-essential dirs
		if strings.HasPrefix(name, ".") || isIgnoredDir(name) {
			continue
		}

		rel, _ := filepath.Rel(filepath.Dir(dir), filepath.Join(dir, name))
		if depth == 0 {
			rel = name
		}

		entry := DirEntry{
			Name:        name,
			Path:        rel,
			Description: inferDirDescription(name),
			Children:    scanDirectoryTree(filepath.Join(dir, name), depth+1),
		}
		tree = append(tree, entry)
	}
	return tree
}
