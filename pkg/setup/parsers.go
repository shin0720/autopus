package setup

import (
	"bufio"
	"os"
	"strings"
)

// parseMakefileTargets extracts all target names and their first recipe line from a Makefile.
// It parses both simple and .PHONY targets, skips internal/dot-prefixed targets,
// and handles variable assignments and multi-line recipes.
func parseMakefileTargets(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	targets := make(map[string]string)
	phonyTargets := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	var currentTarget string
	var currentRecipe []string

	flushTarget := func() {
		if currentTarget != "" && len(currentRecipe) > 0 {
			// Store actual recipe for the target
			recipe := strings.Join(currentRecipe, " && ")
			// Strip @ prefix (silent execution marker)
			recipe = strings.TrimPrefix(recipe, "@")
			targets[currentTarget] = recipe
		} else if currentTarget != "" {
			// Target with no meaningful recipe — use make <target>
			targets[currentTarget] = "make " + currentTarget
		}
		currentTarget = ""
		currentRecipe = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Recipe line (starts with tab)
		if strings.HasPrefix(line, "\t") {
			recipe := strings.TrimSpace(line)
			if currentTarget != "" && recipe != "" && !strings.HasPrefix(recipe, "#") {
				// Skip echo-only lines that are just logging
				if !strings.HasPrefix(recipe, "@echo") && !strings.HasPrefix(recipe, "echo ") {
					currentRecipe = append(currentRecipe, strings.TrimPrefix(recipe, "@"))
				}
			}
			continue
		}

		// Non-recipe line — flush previous target
		flushTarget()

		// Skip comments, empty lines, variable assignments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, "=") && !strings.Contains(trimmed, ":") {
			continue
		}

		// .PHONY declaration
		if strings.HasPrefix(trimmed, ".PHONY:") {
			phonies := strings.TrimPrefix(trimmed, ".PHONY:")
			for _, p := range strings.Fields(phonies) {
				phonyTargets[p] = true
			}
			continue
		}

		// Skip other dot-directives
		if strings.HasPrefix(trimmed, ".") {
			continue
		}

		// Target line: "target: deps" or "target:"
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			name := strings.TrimSpace(parts[0])
			// Skip targets with spaces (likely variable expansions) or paths
			if name != "" && !strings.Contains(name, " ") && !strings.Contains(name, "/") &&
				!strings.Contains(name, "$") {
				currentTarget = name
			}
		}
	}
	flushTarget()

	return targets
}

func parsePyprojectScripts(path string) map[string]string {
	// Simple TOML parsing for [tool.poetry.scripts] or [project.scripts]
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	cmds := make(map[string]string)
	scanner := bufio.NewScanner(f)
	inScripts := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[tool.poetry.scripts]" || line == "[project.scripts]" {
			inScripts = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inScripts = false
			continue
		}
		if inScripts && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			name := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			cmds[name] = value
		}
	}
	return cmds
}
