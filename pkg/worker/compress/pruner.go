package compress

import (
	"fmt"
	"regexp"
	"strings"
)

// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: tool block regexes are provider trace wire-format heuristics
// toolResultPattern matches tool call result blocks in LLM output.
// Supports common formats: <tool_result>...</tool_result>, ```tool_result...```,
// and JSON-style {"type":"tool_result",...}.
var toolResultPattern = regexp.MustCompile(
	`(?s)(<tool_result>.*?</tool_result>|` +
		"```tool_result\\b.*?```" +
		`)`,
)

// toolCallPattern matches tool call blocks.
var toolCallPattern = regexp.MustCompile(
	`(?s)(<tool_call>.*?</tool_call>|` +
		"```tool_call\\b.*?```" +
		`)`,
)

// PruneToolResults replaces old tool call/result blocks with placeholders.
// When both calls and results are present, it preserves or prunes complete pairs
// together so the output never contains orphaned tool blocks.
func PruneToolResults(text string, keepRecent int) string {
	return PruneToolResultsDetailed(text, keepRecent).Text
}

// @AX:ANCHOR: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: pair-aware pruning contract for complete tool call/result preservation
// @AX:REASON: Compressor and reporting helpers depend on this path to preserve recent pairs, replace incomplete pairs, and emit reason codes consistently.
func PruneToolResultsDetailed(text string, keepRecent int) pruneDetails {
	if keepRecent < 0 {
		keepRecent = 0
	}
	blocks := findToolBlocks(text)
	hasCalls, hasResults := hasToolKinds(blocks)
	if hasCalls || hasResults {
		return pruneToolPairs(text, blocks, keepRecent)
	}
	pruned := pruneBlocks(text, toolResultPattern, "[tool_result pruned]", keepRecent)
	pruned = pruneBlocks(pruned, toolCallPattern, "[tool_call pruned]", keepRecent)
	details := pruneDetails{Text: pruned}
	if pruned != text {
		details.ReasonCodes = append(details.ReasonCodes, ReasonToolPairPruned)
	}
	return details
}

// pruneBlocks replaces all but the last keepRecent matches of pattern
// with a placeholder string.
func pruneBlocks(text string, pattern *regexp.Regexp, placeholder string, keepRecent int) string {
	matches := pattern.FindAllStringIndex(text, -1)
	if len(matches) <= keepRecent {
		return text
	}

	toPrune := matches[:len(matches)-keepRecent]

	// Build result by replacing pruned matches from end to start
	// to preserve indices.
	result := text
	for i := len(toPrune) - 1; i >= 0; i-- {
		start, end := toPrune[i][0], toPrune[i][1]
		result = result[:start] + placeholder + result[end:]
	}
	return result
}

// PruneSummary returns a short description of what was pruned.
func PruneSummary(original, pruned string) string {
	origTokens := EstimateTokens(original)
	prunedTokens := EstimateTokens(pruned)
	saved := origTokens - prunedTokens
	if saved <= 0 {
		return "no tokens saved by pruning"
	}
	return fmt.Sprintf("pruned %d tokens (%d → %d)", saved, origTokens, prunedTokens)
}

// CountToolBlocks returns the number of tool_result and tool_call blocks.
func CountToolBlocks(text string) (results int, calls int) {
	for _, block := range findToolBlocks(text) {
		switch block.kind {
		case "result":
			results++
		case "call":
			calls++
		}
	}
	return
}

// PruneAndReport prunes tool blocks and returns the pruned text plus a report line.
func PruneAndReport(text string, keepRecent int) (string, string) {
	details := PruneToolResultsDetailed(text, keepRecent)
	pruned := details.Text
	if pruned == text {
		return text, ""
	}

	origResults, origCalls := CountToolBlocks(text)
	newResults, newCalls := CountToolBlocks(pruned)

	report := fmt.Sprintf(
		"pruned tool blocks: results %d→%d, calls %d→%d | %s",
		origResults, newResults, origCalls, newCalls,
		PruneSummary(text, pruned),
	)
	return pruned, strings.TrimSpace(report)
}
