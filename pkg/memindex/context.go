package memindex

import (
	"fmt"
	"strings"
)

func renderContext(query string, results []SearchResult, budgetTokens int) ([]SearchResult, int, string) {
	var b strings.Builder
	fmt.Fprintf(&b, "## Quality Recall\n\n")
	fmt.Fprintf(&b, "Query: %s\n\n", safeText(query))
	selected := make([]SearchResult, 0, len(results))
	for _, result := range results {
		line := contextLine(result)
		if approxTokens(b.String()+line) > budgetTokens && len(selected) > 0 {
			break
		}
		if approxTokens(b.String()+line) > budgetTokens {
			break
		}
		b.WriteString(line)
		selected = append(selected, result)
	}
	omitted := len(results) - len(selected)
	if omitted > 0 {
		fmt.Fprintf(&b, "\nomitted_results: %d\n", omitted)
	}
	return selected, omitted, b.String()
}

func contextLine(result SearchResult) string {
	parts := []string{
		fmt.Sprintf("- [%d] %s", result.Rank, safeText(result.Title)),
		fmt.Sprintf("  source_ref: %s", safeText(result.SourceRef)),
		fmt.Sprintf("  source_type: %s", result.SourceType),
		fmt.Sprintf("  freshness: %s", result.FreshnessState),
		fmt.Sprintf("  failure_pattern: %s", compact(safeText(result.Title), 180)),
		fmt.Sprintf("  summary: %s", compact(safeText(result.Summary), 260)),
	}
	if result.Severity != "" {
		parts = append(parts, fmt.Sprintf("  severity: %s", result.Severity))
	}
	if len(result.AcceptanceIDs) > 0 {
		parts = append(parts, fmt.Sprintf("  acceptance_refs: %s", safeText(strings.Join(result.AcceptanceIDs, ", "))))
	}
	parts = append(parts, "  next_action: verify source refs before reusing this pattern")
	return strings.Join(parts, "\n") + "\n"
}

func approxTokens(value string) int {
	if value == "" {
		return 0
	}
	return len([]rune(value))/4 + 1
}
