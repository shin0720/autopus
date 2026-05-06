package compress

import (
	"strings"
	"testing"
)

func TestPruneToolResults_KeepsRecent(t *testing.T) {
	text := `start
<tool_result>result 1</tool_result>
middle
<tool_result>result 2</tool_result>
end
<tool_result>result 3</tool_result>
final`

	pruned := PruneToolResults(text, 1)

	if strings.Contains(pruned, "result 1") {
		t.Error("should prune old tool_result 1")
	}
	if strings.Contains(pruned, "result 2") {
		t.Error("should prune old tool_result 2")
	}
	if strings.Contains(pruned, "<tool_result>") {
		t.Error("should not leave orphaned tool_result blocks")
	}
	if !strings.Contains(pruned, "[tool_pair incomplete:") {
		t.Error("should contain incomplete-pair placeholders")
	}
}

func TestPruneToolResults_NoopWhenBelowThreshold(t *testing.T) {
	text := `<tool_result>only one</tool_result>`
	pruned := PruneToolResults(text, 2)
	if strings.Contains(pruned, "<tool_result>") {
		t.Error("should replace orphaned result even when blocks <= keepRecent")
	}
	if !strings.Contains(pruned, "reason=missing_call") {
		t.Fatalf("should include incomplete-pair reason:\n%s", pruned)
	}
}

func TestPruneToolResults_PrunesToolCalls(t *testing.T) {
	text := `<tool_call>call 1</tool_call>
<tool_call>call 2</tool_call>
<tool_call>call 3</tool_call>`

	pruned := PruneToolResults(text, 1)

	if strings.Contains(pruned, "call 1") {
		t.Error("should prune old tool_call 1")
	}
	if strings.Contains(pruned, "<tool_call>") {
		t.Error("should not leave orphaned tool_call blocks")
	}
	if !strings.Contains(pruned, "[tool_pair incomplete:") {
		t.Error("should replace incomplete calls with explicit placeholders")
	}
}

func TestPruneToolResults_IncompletePairHasReason(t *testing.T) {
	text := `<tool_call>{"pair_id":"pair-missing","ordinal":9,"command":"call only"}</tool_call>`

	pruned := PruneToolResults(text, 1)

	if strings.Contains(pruned, "<tool_call>") {
		t.Fatalf("should not leave orphaned tool call:\n%s", pruned)
	}
	if !strings.Contains(pruned, "[tool_pair incomplete: pair=pair-missing ordinal=9 reason=missing_result]") {
		t.Fatalf("missing incomplete-pair reason:\n%s", pruned)
	}
}

func TestPruneToolResults_PairsExplicitCallIDWithFollowingResult(t *testing.T) {
	text := `<tool_call>{"pair_id":"pair-explicit","command":"call"}</tool_call>
<tool_result>{"body":"result without pair id"}</tool_result>`

	pruned := PruneToolResults(text, 1)

	if !strings.Contains(pruned, "pair-explicit") {
		t.Fatalf("explicit call id should pair with following result:\n%s", pruned)
	}
	results, calls := CountToolBlocks(pruned)
	if results != 1 || calls != 1 {
		t.Fatalf("expected complete pair to be preserved, got calls=%d results=%d:\n%s", calls, results, pruned)
	}
}

func TestPruneToolResults_JSONStyleToolBlocks(t *testing.T) {
	text := strings.Join([]string{
		`{"type":"tool_call","pair_id":"json-1","ordinal":1,"payload":{"command":"old"}}`,
		`{"type":"tool_result","pair_id":"json-1","ordinal":1,"body":"old {\"nested\":true}"}`,
		`{"type":"tool_call","pair_id":"json-2","ordinal":2,"payload":{"command":"recent"}}`,
		`{"type":"tool_result","pair_id":"json-2","ordinal":2,"body":"recent {\"nested\":true}"}`,
	}, "\n")

	pruned := PruneToolResults(text, 1)

	if strings.Contains(pruned, `"command":"old"`) || strings.Contains(pruned, "old {") {
		t.Fatalf("old JSON pair should be pruned:\n%s", pruned)
	}
	if !strings.Contains(pruned, "[tool_pair pruned: pair=json-1 ordinal=1]") {
		t.Fatalf("missing JSON pair placeholder:\n%s", pruned)
	}
	if !strings.Contains(pruned, "json-2") {
		t.Fatalf("recent JSON pair should be preserved:\n%s", pruned)
	}
	if strings.Contains(pruned, "recent {") {
		t.Fatalf("recent JSON pair should omit provider body:\n%s", pruned)
	}
	if !strings.Contains(pruned, "provider_payload_omitted") {
		t.Fatalf("recent JSON pair should preserve only safe metadata:\n%s", pruned)
	}
	results, calls := CountToolBlocks(pruned)
	if results != 1 || calls != 1 {
		t.Fatalf("expected one complete JSON pair, got calls=%d results=%d:\n%s", calls, results, pruned)
	}
}

func TestCountToolBlocks(t *testing.T) {
	text := `<tool_result>r1</tool_result>
<tool_call>c1</tool_call>
<tool_result>r2</tool_result>`

	results, calls := CountToolBlocks(text)
	if results != 2 {
		t.Errorf("results = %d, want 2", results)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestPruneSummary(t *testing.T) {
	original := strings.Repeat("a", 400) // 100 tokens
	pruned := strings.Repeat("a", 200)   // 50 tokens

	summary := PruneSummary(original, pruned)
	if !strings.Contains(summary, "50") {
		t.Errorf("summary should mention saved tokens: %s", summary)
	}
}

func TestPruneAndReport_NoPruning(t *testing.T) {
	text := "plain text without tool blocks"
	result, report := PruneAndReport(text, 2)
	if result != text {
		t.Error("should return original when nothing to prune")
	}
	if report != "" {
		t.Errorf("report should be empty, got: %s", report)
	}
}

func TestPruneAndReport_WithPruning(t *testing.T) {
	text := `<tool_result>r1</tool_result>
<tool_result>r2</tool_result>
<tool_result>r3</tool_result>`

	result, report := PruneAndReport(text, 1)
	if !strings.Contains(result, "[tool_pair incomplete:") {
		t.Error("should contain placeholder")
	}
	if report == "" {
		t.Error("report should not be empty")
	}
	if !strings.Contains(report, "pruned tool blocks") {
		t.Error("report should describe pruning")
	}
}
