package compress

import "fmt"

// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: provider tool bodies are intentionally replaced with metadata-only placeholders before summary/event extraction
func omitToolPayloadBodies(text string) (string, bool) {
	blocks := findToolBlocks(text)
	if len(blocks) == 0 {
		return text, false
	}
	replacements := make([]replacement, 0, len(blocks))
	for _, block := range blocks {
		replacements = append(replacements, replacement{
			start: block.start,
			end:   block.end,
			text:  toolMetadataPlaceholder(block),
		})
	}
	return applyReplacements(text, replacements), true
}

func toolMetadataPlaceholder(block toolBlock) string {
	return fmt.Sprintf("[tool_%s metadata: pair=%s ordinal=%s body=omitted reason=provider_payload_omitted]",
		block.kind,
		pairRefFromBlock(block),
		ordinalRefFromBlock(block),
	)
}

func pairRefFromBlock(block toolBlock) string {
	if block.pairID != "" {
		return block.pairID
	}
	if block.key != "" {
		return block.key
	}
	return "unknown"
}

func ordinalRefFromBlock(block toolBlock) string {
	if block.ordinal != "" {
		return block.ordinal
	}
	return "unknown"
}
