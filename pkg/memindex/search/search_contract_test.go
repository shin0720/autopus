package search_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insajin/autopus-adk/internal/cli"
)

func TestMemSearch_TopKIncludesProvenanceAndDeterministicTieOrdering(t *testing.T) {
	t.Parallel()

	projectDir := makeSearchFixture(t)
	_, rebuildRaw, err := runAutoJSON(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, rebuildRaw)

	payload, searchRaw, err := runAutoJSON(t, "mem", "search", "review failure pattern tie-token", "--project-dir", projectDir, "--top-k", "3", "--format", "json")
	require.NoError(t, err, searchRaw)
	results := asSlice(t, jsonData(t, payload)["results"])
	require.Len(t, results, 3)

	first := asMap(t, results[0])
	for _, field := range []string{"source_type", "source_ref", "source_hash", "rank", "freshness_state", "snippet_digest", "redaction_status"} {
		assert.NotEmpty(t, first[field], "search result must include %s", field)
	}
	assert.Equal(t, "fresh", first["freshness_state"])
	assert.LessOrEqual(t, len(first["snippet_digest"].(string)), 160)

	gotOrder := make([]string, 0, len(results))
	for _, raw := range results {
		result := asMap(t, raw)
		gotOrder = append(gotOrder, result["source_type"].(string)+":"+result["source_ref"].(string))
	}
	assert.Equal(t, []string{
		"learning:L-TIE",
		"review_failure:.autopus/project/reviews/review-tie.md",
		"spec:.autopus/specs/SPEC-TIE-001/spec.md",
	}, gotOrder)
}

func TestMemSearch_RequireFreshAndCorruptProjectionFailClosed(t *testing.T) {
	t.Parallel()

	projectDir := makeSearchFixture(t)
	_, rebuildRaw, err := runAutoJSON(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, rebuildRaw)

	specPath := filepath.Join(projectDir, ".autopus", "specs", "SPEC-TIE-001", "spec.md")
	require.NoError(t, os.WriteFile(specPath, []byte("# SPEC-TIE-001\n\nold decision changed after rebuild\n"), 0o644))
	_, staleRaw, err := runAutoJSON(t, "mem", "search", "old decision", "--require-fresh", "--project-dir", projectDir, "--format", "json")
	require.Error(t, err, staleRaw)
	assert.ErrorContains(t, err, "stale-source")
	assert.NotContains(t, staleRaw, `"freshness_state":"fresh"`)

	corruptPath := filepath.Join(projectDir, ".autopus", "runtime", "memindex", "autopus-mem.sqlite")
	require.NoError(t, os.MkdirAll(filepath.Dir(corruptPath), 0o755))
	require.NoError(t, os.WriteFile(corruptPath, []byte("not a sqlite database"), 0o644))
	_, corruptRaw, err := runAutoJSON(t, "mem", "search", "anything", "--project-dir", projectDir, "--format", "json")
	require.Error(t, err, corruptRaw)
	assert.ErrorContains(t, err, "projection-corrupt")
	assert.NotContains(t, corruptRaw, "fabricated")
}

func TestMemSearch_UsesFTSRelevanceBeforeDeterministicTieBreakers(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, ".autopus", "specs", "SPEC-REL-001", "spec.md"), `# SPEC-REL-001

needle relevance needle relevance needle relevance needle relevance needle relevance
`)
	writeFile(t, filepath.Join(projectDir, ".autopus", "learnings", "pipeline.jsonl"),
		`{"id":"L-REL","timestamp":"2026-05-06T00:00:00Z","type":"review_issue","phase":"review","spec_id":"SPEC-REL-001","files":["pkg/review/check.go"],"packages":["pkg/review"],"pattern":"needle relevance","resolution":"single mention","severity":"medium","reuse_count":1}`+"\n")

	_, rebuildRaw, err := runAutoJSON(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, rebuildRaw)
	payload, searchRaw, err := runAutoJSON(t, "mem", "search", "needle relevance", "--project-dir", projectDir, "--top-k", "2", "--format", "json")
	require.NoError(t, err, searchRaw)

	results := asSlice(t, jsonData(t, payload)["results"])
	require.Len(t, results, 2)
	first := asMap(t, results[0])
	assert.Equal(t, "spec", first["source_type"])
	assert.Equal(t, ".autopus/specs/SPEC-REL-001/spec.md", first["source_ref"])
}

func makeSearchFixture(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, ".autopus", "project", "reviews", "review-tie.md"), `---
timestamp: 2026-05-06T00:00:00Z
---
# Review Failure

review failure pattern tie-token appears in repeated review feedback.
`)
	writeFile(t, filepath.Join(projectDir, ".autopus", "specs", "SPEC-TIE-001", "spec.md"), `---
timestamp: 2026-05-06T00:00:00Z
---
# SPEC-TIE-001

Decision: review failure pattern tie-token should use deterministic ordering.
`)
	writeFile(t, filepath.Join(projectDir, ".autopus", "learnings", "pipeline.jsonl"),
		`{"id":"L-TIE","timestamp":"2026-05-06T00:00:00Z","type":"review_issue","phase":"review","spec_id":"SPEC-TIE-001","files":["pkg/review/check.go"],"packages":["pkg/review"],"pattern":"review failure pattern tie-token review failure pattern tie-token","resolution":"sort by source type timestamp source ref row id","severity":"medium","reuse_count":1}`+"\n")
	return projectDir
}

func runAutoJSON(t *testing.T, args ...string) (map[string]any, string, error) {
	t.Helper()

	var out bytes.Buffer
	cmd := cli.NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		return nil, out.String(), err
	}

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload), out.String())
	return payload, out.String(), nil
}

func jsonData(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	assert.Equal(t, "ok", payload["status"])
	return asMap(t, payload["data"])
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	got, ok := value.(map[string]any)
	require.True(t, ok, "expected map[string]any, got %T", value)
	return got
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()
	got, ok := value.([]any)
	require.True(t, ok, "expected []any, got %T", value)
	return got
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
