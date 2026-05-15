package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// agentAutoContextGlobs maps agentID to glob patterns scanned for auto-context files.
var agentAutoContextGlobs = map[string][]string{
	"arch": {"개발용/*.md", "개발용/**/*.md", ".autopus/specs/*.md"},
	"spec": {"개발용/*.md", ".autopus/specs/*.md"},
	"plan": {"개발용/*.md"},
	"exec": {"개발용/*.md", ".autopus/specs/*.md"},
	"deep": {"개발용/*.md", ".autopus/specs/*.md"},
	"dbug": {"개발용/*.md"},
	"revw": {"개발용/*.md"},
	"test": {"개발용/*.md"},
}

// gatherAutoContextFiles returns relative file paths matching the agent's globs,
// skipping paths already listed in explicit.
func gatherAutoContextFiles(root, agentID string, explicit []string) []string {
	patterns, ok := agentAutoContextGlobs[agentID]
	if !ok {
		return nil
	}
	have := make(map[string]bool, len(explicit))
	for _, p := range explicit {
		have[p] = true
	}
	var result []string
	for _, pat := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pat)))
		if err != nil {
			continue
		}
		for _, abs := range matches {
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if !have[rel] {
				have[rel] = true
				result = append(result, rel)
			}
		}
	}
	return result
}

// devAgentIDs are the agents that must complete ALL checklist items before handoff.
var devAgentIDs = map[string]bool{
	"exec": true,
	"deep": true,
	"dbug": true,
}

// specLineRe matches "✅ SPEC-XXX 완료" in agent output.
var specLineRe = regexp.MustCompile(`✅\s+(SPEC-[A-Z0-9_\-]+)\s+완료`)

// buildSpecChecklist scans context file contents for SPEC IDs and returns a checklist.
func buildSpecChecklist(root string, contextPaths []string) []checklistItem {
	seen := map[string]bool{}
	var items []checklistItem
	idRe := regexp.MustCompile(`SPEC-[A-Z0-9_\-]+`)
	for _, p := range contextPaths {
		for _, m := range idRe.FindAllString(p, -1) {
			if !seen[m] {
				seen[m] = true
				items = append(items, checklistItem{ID: m, Label: m})
			}
		}
		data, _ := readWorkspaceFile(root, p)
		if len(data) > 4096 {
			data = data[:4096]
		}
		for _, m := range idRe.FindAllString(string(data), -1) {
			if !seen[m] {
				seen[m] = true
				items = append(items, checklistItem{ID: m, Label: m})
			}
		}
	}
	return items
}

// applyChecklistDone marks items done based on ✅ SPEC-XXX 완료 lines in output.
func applyChecklistDone(items []checklistItem, output string) []checklistItem {
	doneIDs := map[string]bool{}
	for _, m := range specLineRe.FindAllStringSubmatch(output, -1) {
		doneIDs[m[1]] = true
	}
	updated := make([]checklistItem, len(items))
	copy(updated, items)
	for i, it := range updated {
		if doneIDs[it.ID] {
			updated[i].Done = true
		}
	}
	return updated
}

// allChecklistDone reports whether every item in the list is marked done.
func allChecklistDone(items []checklistItem) bool {
	if len(items) == 0 {
		return true
	}
	for _, it := range items {
		if !it.Done {
			return false
		}
	}
	return true
}

// buildDevPrompt wraps the user prompt with the full SPEC checklist and completion rules.
func buildDevPrompt(base string, items []checklistItem) string {
	if len(items) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n---\n[구현 체크리스트]\n")
	sb.WriteString("아래 항목을 모두 구현해야 작업이 완료됩니다. 하나만 완료했다고 멈추지 마세요.\n\n")
	for i, it := range items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, it.Label))
	}
	sb.WriteString("\n[완료 보고 규칙]\n")
	sb.WriteString("각 항목 구현 후 반드시 다음 형식으로 보고하세요:\n")
	sb.WriteString("  ✅ SPEC-XXX 완료: (구현 내용 한 줄 요약)\n")
	sb.WriteString("\n모든 항목을 완료한 후에만 '## 작업 요약'을 작성하세요.\n")
	return sb.String()
}
