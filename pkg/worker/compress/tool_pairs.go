package compress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	toolPairIDPattern  = regexp.MustCompile(`"pair_id"\s*:\s*"([^"]+)"`)
	toolOrdinalPattern = regexp.MustCompile(`"ordinal"\s*:\s*([0-9]+)`)
)

type toolBlock struct {
	kind    string
	start   int
	end     int
	text    string
	pairID  string
	ordinal string
	key     string
}

type toolPair struct {
	key     string
	pairID  string
	ordinal string
	call    *toolBlock
	result  *toolBlock
	start   int
	end     int
}

type replacement struct {
	start, end int
	text       string
}

func findToolBlocks(text string) []toolBlock {
	var blocks []toolBlock
	for _, match := range toolCallPattern.FindAllStringIndex(text, -1) {
		blocks = append(blocks, newToolBlock("call", text, match))
	}
	for _, match := range toolResultPattern.FindAllStringIndex(text, -1) {
		blocks = append(blocks, newToolBlock("result", text, match))
	}
	for _, match := range findJSONToolBlockIndices(text) {
		blocks = append(blocks, newToolBlock(match.kind, text, []int{match.start, match.end}))
	}
	blocks = dedupeToolBlocks(blocks)
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].start < blocks[j].start
	})
	assignToolKeys(blocks)
	return blocks
}

type jsonToolBlockIndex struct {
	kind  string
	start int
	end   int
}

func findJSONToolBlockIndices(text string) []jsonToolBlockIndex {
	var matches []jsonToolBlockIndex
	for offset := 0; offset < len(text); offset++ {
		if text[offset] != '{' {
			continue
		}
		var payload map[string]any
		dec := json.NewDecoder(bytes.NewReader([]byte(text[offset:])))
		if err := dec.Decode(&payload); err != nil {
			continue
		}
		kind, ok := payload["type"].(string)
		if !ok {
			continue
		}
		if kind != "tool_call" && kind != "tool_result" {
			continue
		}
		matches = append(matches, jsonToolBlockIndex{
			kind:  strings.TrimPrefix(kind, "tool_"),
			start: offset,
			end:   offset + int(dec.InputOffset()),
		})
	}
	return matches
}

func dedupeToolBlocks(blocks []toolBlock) []toolBlock {
	seen := map[string]bool{}
	var out []toolBlock
	for _, block := range blocks {
		key := fmt.Sprintf("%d:%d:%s", block.start, block.end, block.kind)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, block)
	}
	return out
}

func newToolBlock(kind, text string, match []int) toolBlock {
	blockText := text[match[0]:match[1]]
	pairID := firstSubmatch(toolPairIDPattern, blockText)
	ordinal := firstSubmatch(toolOrdinalPattern, blockText)
	key := ""
	if pairID != "" {
		key = "pair:" + pairID
	} else if ordinal != "" {
		key = "ordinal:" + ordinal
	}
	return toolBlock{
		kind:    kind,
		start:   match[0],
		end:     match[1],
		text:    blockText,
		pairID:  pairID,
		ordinal: ordinal,
		key:     key,
	}
}

func firstSubmatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: fallback sequence keys pair unlabelled calls with following results; do not prune them independently
func assignToolKeys(blocks []toolBlock) {
	pendingCall := -1
	sequence := 1
	for i := range blocks {
		if blocks[i].kind == "call" {
			if blocks[i].key == "" {
				blocks[i].key = fmt.Sprintf("sequence:%d", sequence)
				sequence++
			}
			pendingCall = i
			continue
		}
		if blocks[i].key == "" && pendingCall >= 0 {
			blocks[i].key = blocks[pendingCall].key
			pendingCall = -1
			continue
		}
		if blocks[i].key == "" {
			blocks[i].key = fmt.Sprintf("standalone:%d", sequence)
			sequence++
		}
	}
}

func hasToolKinds(blocks []toolBlock) (bool, bool) {
	hasCalls := false
	hasResults := false
	for _, block := range blocks {
		hasCalls = hasCalls || block.kind == "call"
		hasResults = hasResults || block.kind == "result"
	}
	return hasCalls, hasResults
}

func pruneToolPairs(text string, blocks []toolBlock, keepRecent int) pruneDetails {
	pairs := collectToolPairs(blocks)
	completePairs := completeToolPairs(pairs)
	sort.Slice(completePairs, func(i, j int) bool {
		return completePairs[i].end < completePairs[j].end
	})

	kept := map[string]bool{}
	for i := max(0, len(completePairs)-keepRecent); i < len(completePairs); i++ {
		kept[completePairs[i].key] = true
	}

	var replacements []replacement
	details := pruneDetails{Text: text}
	for _, pair := range pairs {
		if pair.call != nil && pair.result != nil {
			if !kept[pair.key] {
				replacements = append(replacements, replacement{
					start: pair.start,
					end:   pair.end,
					text:  prunedPairPlaceholder(pair),
				})
				details.PrunedPairCount++
			} else {
				replacements = append(replacements,
					replacement{start: pair.call.start, end: pair.call.end, text: preservedToolBlock(pair.call)},
					replacement{start: pair.result.start, end: pair.result.end, text: preservedToolBlock(pair.result)},
				)
			}
			continue
		}
		replacements = append(replacements, replacement{
			start: pair.start,
			end:   pair.end,
			text:  incompletePairPlaceholder(pair),
		})
		details.IncompletePairCount++
	}
	details.Text = applyReplacements(text, replacements)
	if details.PrunedPairCount > 0 {
		details.ReasonCodes = append(details.ReasonCodes, ReasonToolPairPruned)
	}
	if details.IncompletePairCount > 0 {
		details.ReasonCodes = append(details.ReasonCodes, ReasonIncompleteToolPair)
	}
	return details
}

func collectToolPairs(blocks []toolBlock) map[string]*toolPair {
	pairs := map[string]*toolPair{}
	for i := range blocks {
		block := &blocks[i]
		pair := pairs[block.key]
		if pair == nil {
			pair = &toolPair{key: block.key, start: block.start, end: block.end}
			pairs[block.key] = pair
		}
		pair.start = min(pair.start, block.start)
		pair.end = max(pair.end, block.end)
		if block.pairID != "" {
			pair.pairID = block.pairID
		}
		if block.ordinal != "" {
			pair.ordinal = block.ordinal
		}
		if block.kind == "call" {
			pair.call = block
		} else {
			pair.result = block
		}
	}
	return pairs
}

func completeToolPairs(pairs map[string]*toolPair) []*toolPair {
	var complete []*toolPair
	for _, pair := range pairs {
		if pair.call != nil && pair.result != nil {
			complete = append(complete, pair)
		}
	}
	return complete
}

func prunedPairPlaceholder(pair *toolPair) string {
	return fmt.Sprintf("[tool_pair pruned: pair=%s ordinal=%s]", pairRef(pair), ordinalRef(pair))
}

func incompletePairPlaceholder(pair *toolPair) string {
	reason := "missing_result"
	if pair.call == nil {
		reason = "missing_call"
	}
	return fmt.Sprintf("[tool_pair incomplete: pair=%s ordinal=%s reason=%s]", pairRef(pair), ordinalRef(pair), reason)
}

func preservedToolBlock(block *toolBlock) string {
	return fmt.Sprintf("<tool_%s>{\"pair_id\":\"%s\",\"ordinal\":\"%s\",\"body\":\"[omitted]\",\"reason\":\"provider_payload_omitted\"}</tool_%s>",
		block.kind,
		pairRefFromBlock(*block),
		ordinalRefFromBlock(*block),
		block.kind,
	)
}

func pairRef(pair *toolPair) string {
	if pair.pairID != "" {
		return pair.pairID
	}
	return pair.key
}

func ordinalRef(pair *toolPair) string {
	if pair.ordinal != "" {
		return pair.ordinal
	}
	return "unknown"
}

func applyReplacements(text string, replacements []replacement) string {
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start > replacements[j].start
	})
	result := text
	for _, repl := range replacements {
		result = result[:repl.start] + repl.text + result[repl.end:]
	}
	return result
}
