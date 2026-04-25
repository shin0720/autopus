package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func detectLanguages(dir string) []Language {
	var langs []Language

	// Go
	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		ver := ""
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				ver = strings.TrimPrefix(line, "go ")
				break
			}
		}
		langs = append(langs, Language{
			Name:       "Go",
			Version:    ver,
			BuildFiles: []string{"go.mod"},
		})
	}

	// TypeScript / JavaScript
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			DevDeps map[string]string `json:"devDependencies"`
			Deps    map[string]string `json:"dependencies"`
		}
		_ = json.Unmarshal(data, &pkg)

		if _, ok := pkg.DevDeps["typescript"]; ok {
			langs = append(langs, Language{
				Name:       "TypeScript",
				Version:    pkg.DevDeps["typescript"],
				BuildFiles: []string{"package.json", "tsconfig.json"},
			})
		} else {
			langs = append(langs, Language{
				Name:       "JavaScript",
				BuildFiles: []string{"package.json"},
			})
		}
	}

	// Python
	if fileExists(filepath.Join(dir, "pyproject.toml")) ||
		fileExists(filepath.Join(dir, "setup.py")) ||
		fileExists(filepath.Join(dir, "requirements.txt")) {

		bf := []string{}
		for _, f := range []string{"pyproject.toml", "setup.py", "requirements.txt"} {
			if fileExists(filepath.Join(dir, f)) {
				bf = append(bf, f)
			}
		}
		langs = append(langs, Language{
			Name:       "Python",
			BuildFiles: bf,
		})
	}

	// Rust
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		langs = append(langs, Language{
			Name:       "Rust",
			BuildFiles: []string{"Cargo.toml"},
		})
	}

	return langs
}

func detectFrameworks(dir string) []Framework {
	var frameworks []Framework

	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			Deps    map[string]string `json:"dependencies"`
			DevDeps map[string]string `json:"devDependencies"`
		}
		_ = json.Unmarshal(data, &pkg)

		knownFrameworks := map[string]string{
			"react":   "React",
			"vue":     "Vue",
			"next":    "Next.js",
			"express": "Express",
			"nestjs":  "NestJS",
			"angular": "Angular",
		}
		allDeps := mergeMaps(pkg.Deps, pkg.DevDeps)
		for key, name := range knownFrameworks {
			if ver, ok := allDeps[key]; ok {
				frameworks = append(frameworks, Framework{Name: name, Version: ver})
			}
		}
	}

	return frameworks
}

func detectBuildFiles(dir string) []BuildFile {
	var buildFiles []BuildFile

	// Makefile
	if fileExists(filepath.Join(dir, "Makefile")) {
		bf := BuildFile{
			Path:     "Makefile",
			Type:     "makefile",
			Commands: parseMakefileTargets(filepath.Join(dir, "Makefile")),
		}
		buildFiles = append(buildFiles, bf)
	}

	// package.json scripts
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		_ = json.Unmarshal(data, &pkg)
		if len(pkg.Scripts) > 0 {
			buildFiles = append(buildFiles, BuildFile{
				Path:     "package.json",
				Type:     "package.json",
				Commands: pkg.Scripts,
			})
		}
	}

	// go.mod (standard go commands)
	if fileExists(filepath.Join(dir, "go.mod")) {
		buildFiles = append(buildFiles, BuildFile{
			Path: "go.mod",
			Type: "go.mod",
			Commands: map[string]string{
				"build": "go build ./...",
				"test":  "go test ./...",
				"vet":   "go vet ./...",
			},
		})
	}

	// pyproject.toml
	if fileExists(filepath.Join(dir, "pyproject.toml")) {
		cmds := parsePyprojectScripts(filepath.Join(dir, "pyproject.toml"))
		if len(cmds) > 0 {
			buildFiles = append(buildFiles, BuildFile{
				Path:     "pyproject.toml",
				Type:     "pyproject.toml",
				Commands: cmds,
			})
		}
	}

	// Cargo.toml
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		buildFiles = append(buildFiles, BuildFile{
			Path: "Cargo.toml",
			Type: "cargo.toml",
			Commands: map[string]string{
				"build": "cargo build",
				"test":  "cargo test",
				"check": "cargo check",
				"clippy": "cargo clippy",
			},
		})
	}

	// docker-compose.yml
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if fileExists(filepath.Join(dir, name)) {
			buildFiles = append(buildFiles, BuildFile{
				Path: name,
				Type: "docker-compose",
				Commands: map[string]string{
					"up":   "docker compose up -d",
					"down": "docker compose down",
					"logs": "docker compose logs -f",
				},
			})
			break
		}
	}

	return buildFiles
}

func detectEntryPoints(dir string, langs []Language) []EntryPoint {
	var eps []EntryPoint

	for _, lang := range langs {
		switch lang.Name {
		case "Go":
			// cmd/ directories
			cmdDir := filepath.Join(dir, "cmd")
			if entries, err := os.ReadDir(cmdDir); err == nil {
				for _, e := range entries {
					if e.IsDir() {
						mainFile := filepath.Join("cmd", e.Name(), "main.go")
						if fileExists(filepath.Join(dir, mainFile)) {
							eps = append(eps, EntryPoint{
								Path:        mainFile,
								Description: e.Name() + " CLI entry point",
							})
						}
					}
				}
			}
			// root main.go
			if fileExists(filepath.Join(dir, "main.go")) {
				eps = append(eps, EntryPoint{
					Path:        "main.go",
					Description: "Main entry point",
				})
			}
		case "TypeScript", "JavaScript":
			for _, f := range []string{"src/index.ts", "src/index.js", "src/main.ts", "src/main.js", "index.ts", "index.js"} {
				if fileExists(filepath.Join(dir, f)) {
					eps = append(eps, EntryPoint{
						Path:        f,
						Description: "Application entry point",
					})
				}
			}
		case "Python":
			for _, f := range []string{"main.py", "app.py", "src/main.py", "manage.py"} {
				if fileExists(filepath.Join(dir, f)) {
					eps = append(eps, EntryPoint{
						Path:        f,
						Description: "Application entry point",
					})
				}
			}
		}
	}
	return eps
}

func detectTestConfig(dir string, langs []Language, buildFiles []BuildFile) TestConfiguration {
	tc := TestConfiguration{}

	for _, lang := range langs {
		switch lang.Name {
		case "Go":
			tc.Framework = "go test"
			tc.Command = "go test -race ./..."
			tc.Dirs = findDirsWithSuffix(dir, "_test.go")
			// Check for coverage in Makefile
			for _, bf := range buildFiles {
				for _, cmd := range bf.Commands {
					if strings.Contains(cmd, "-cover") || strings.Contains(cmd, "--cov") {
						tc.CoverageOn = true
						break
					}
				}
			}
		case "TypeScript", "JavaScript":
			for _, bf := range buildFiles {
				if cmd, ok := bf.Commands["test"]; ok {
					tc.Command = "npm test"
					if strings.Contains(cmd, "jest") {
						tc.Framework = "Jest"
					} else if strings.Contains(cmd, "vitest") {
						tc.Framework = "Vitest"
					} else if strings.Contains(cmd, "mocha") {
						tc.Framework = "Mocha"
					} else {
						tc.Framework = "npm test"
					}
					break
				}
			}
			for _, d := range []string{"test", "tests", "__tests__", "spec"} {
				if fileExists(filepath.Join(dir, d)) {
					tc.Dirs = append(tc.Dirs, d)
				}
			}
		case "Python":
			tc.Framework = "pytest"
			tc.Command = "pytest"
			for _, d := range []string{"tests", "test"} {
				if fileExists(filepath.Join(dir, d)) {
					tc.Dirs = append(tc.Dirs, d)
				}
			}
		}
		if tc.Framework != "" {
			break
		}
	}
	return tc
}
