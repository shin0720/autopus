package orchestra

import "github.com/insajin/autopus-adk/pkg/worker/compress"

// CompressContext applies the shared structured compaction contract to
// orchestra context payloads before they are forwarded into long prompts.
func (cs *ContextSummarizer) CompressContext(label, text, provider string) compress.CompactionResult {
	// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: keepRecent=2 mirrors the pipeline phase-transition compaction policy
	return compress.NewDefaultCompressor(2).CompressDetailed(label, text, provider)
}
