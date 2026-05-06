package compress

import (
	"strings"
	"testing"
)

func TestDefaultCompressor_BelowThreshold(t *testing.T) {
	c := NewDefaultCompressor(2)
	output := "short output that should not be compressed"
	result := c.Compress("executor", output, "claude")
	if result != output {
		t.Error("should return original when below threshold")
	}
}

func TestDefaultCompressor_AboveThreshold(t *testing.T) {
	c := NewDefaultCompressor(1)

	// claude threshold = 100000 tokens = 400000 chars
	largeOutput := "## Goal\nImplement feature X\n\n" +
		"## Progress\nDid many things\n\n" +
		strings.Repeat("<tool_result>some tool output here</tool_result>\n", 50) +
		strings.Repeat("detailed implementation notes ", 15000)

	result := c.Compress("executor", largeOutput, "claude")

	if result == largeOutput {
		t.Error("should compress when above threshold")
	}
	if !strings.Contains(result, "## Phase Summary: executor") {
		t.Error("compressed output should contain structured summary")
	}
	if EstimateTokens(result) > SummaryBudget("claude")+20 {
		t.Errorf("compressed output too large: %d tokens", EstimateTokens(result))
	}
}

func TestDefaultCompressor_KeepRecentTools(t *testing.T) {
	c := NewDefaultCompressor(2)

	// codex threshold = 64000 tokens = 256000 chars
	output := strings.Repeat("x", 260000) +
		"<tool_result>old1</tool_result>\n" +
		"<tool_result>old2</tool_result>\n" +
		"<tool_result>recent1</tool_result>\n" +
		"<tool_result>recent2</tool_result>\n"

	result := c.Compress("tester", output, "codex")
	if result == output {
		t.Error("should compress")
	}
}

func TestNopCompressor(t *testing.T) {
	c := NopCompressor{}
	output := strings.Repeat("a", 500000) // very large
	result := c.Compress("executor", output, "claude")
	if result != output {
		t.Error("NopCompressor should always return original")
	}
}

func TestNewDefaultCompressor_NegativeKeep(t *testing.T) {
	c := NewDefaultCompressor(-1)
	if c.KeepRecentTools != 0 {
		t.Errorf("negative keepRecent should default to 0, got %d", c.KeepRecentTools)
	}
}

func TestContextCompressorInterface(t *testing.T) {
	// Verify both types implement the interface.
	var _ ContextCompressor = &DefaultCompressor{}
	var _ ContextCompressor = NopCompressor{}
}

func TestDefaultCompressor_CompressDetailed_BelowThresholdEvent(t *testing.T) {
	c := NewDefaultCompressor(1)
	output := "short output"

	result := c.CompressDetailed("planner", output, "codex")

	if result.Output != output {
		t.Fatal("below-threshold output should pass through unchanged")
	}
	if result.Event.CompactionApplied {
		t.Fatal("below-threshold event should report no compaction")
	}
	if !containsString(result.Event.ReasonCodes, ReasonBelowThreshold) {
		t.Fatalf("missing below-threshold reason: %#v", result.Event.ReasonCodes)
	}
}

func TestDefaultCompressor_CompressDetailed_CompactionEventMetadata(t *testing.T) {
	oldWindow, hadWindow := ModelWindows["tiny-test"]
	ModelWindows["tiny-test"] = 10
	defer func() {
		if hadWindow {
			ModelWindows["tiny-test"] = oldWindow
			return
		}
		delete(ModelWindows, "tiny-test")
	}()

	output := strings.Join([]string{
		`index_export=true`,
		`## Goal`,
		`Ship SPEC-CONTEXT-COMPRESS-001 for AC-CCOMP-005 in pkg/worker/compress/compressor.go.`,
		`## Progress`,
		`<tool_call>{"pair_id":"pair-1","ordinal":1,"command":"old"}</tool_call>`,
		`<tool_result>{"pair_id":"pair-1","ordinal":1,"body":"old sk-test-abcdef123456 at /Users/alice/private.json SPEC-LEAK-999 AC-LEAK-999 pkg/private/leaked.go"}</tool_result>`,
		`<tool_call>{"pair_id":"pair-2","ordinal":2,"command":"new"}</tool_call>`,
		`<tool_result>{"pair_id":"pair-2","ordinal":2,"body":"new"}</tool_result>`,
	}, "\n")

	result := NewDefaultCompressor(1).CompressDetailed("executor", output, "tiny-test")

	if !result.Event.CompactionApplied {
		t.Fatal("event should report compaction")
	}
	if result.Event.SummaryID == "" {
		t.Fatal("event should include summary id")
	}
	if result.Event.PrunedPairCount != 1 {
		t.Fatalf("pruned pair count = %d, want 1", result.Event.PrunedPairCount)
	}
	for _, reason := range []string{ReasonThresholdExceeded, ReasonToolPairPruned, ReasonSecretRedacted, ReasonLocalPathRedacted, ReasonProviderPayloadOmitted, ReasonIndexEligible} {
		if !containsString(result.Event.ReasonCodes, reason) {
			t.Fatalf("missing reason %q in %#v", reason, result.Event.ReasonCodes)
		}
	}
	for _, section := range []string{"Goal", "Constraints", "Progress", "Decisions", "Relevant Files", "Next Steps", "Critical Context"} {
		if !containsString(result.Event.PreservedSections, section) {
			t.Fatalf("missing preserved section %q in %#v", section, result.Event.PreservedSections)
		}
	}
	for _, ref := range []string{"SPEC-CONTEXT-COMPRESS-001", "AC-CCOMP-005", "pkg/worker/compress/compressor.go", "pair-1"} {
		if !containsString(result.Event.SourceRefs, ref) {
			t.Fatalf("missing source ref %q in %#v", ref, result.Event.SourceRefs)
		}
	}
	for _, leakedRef := range []string{"SPEC-LEAK-999", "AC-LEAK-999", "pkg/private/leaked.go"} {
		if containsString(result.Event.SourceRefs, leakedRef) {
			t.Fatalf("provider-payload ref %q leaked into event source refs: %#v", leakedRef, result.Event.SourceRefs)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
