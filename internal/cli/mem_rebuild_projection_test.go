package cli_test

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

func TestMemRebuild_DeterministicProjectionOnlyAndFTS5Probe(t *testing.T) {
	t.Parallel()

	projectDir := makeProjectionFixture(t)

	first, firstRaw, err := runAutoJSON(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, firstRaw)
	second, secondRaw, err := runAutoJSON(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, secondRaw)

	firstData := jsonData(t, first)
	secondData := jsonData(t, second)
	assert.Equal(t, "autopus.mem_index.v1", firstData["schema_version"])
	assert.Equal(t, firstData["schema_version"], secondData["schema_version"])
	assert.Equal(t, true, firstData["projection_only"])
	assert.Equal(t, firstData["counts_by_source_kind"], secondData["counts_by_source_kind"])
	assert.Equal(t, firstData["source_hashes"], secondData["source_hashes"])

	counts := asMap(t, firstData["counts_by_source_kind"])
	assert.Equal(t, float64(1), counts["project_doc"])
	assert.Equal(t, float64(1), counts["spec"])
	assert.Equal(t, float64(1), counts["learning"])

	indexPath, ok := firstData["index_path"].(string)
	require.True(t, ok, "index_path must be a string")
	assert.FileExists(t, indexPath)
	slashPath := filepath.ToSlash(indexPath)
	assert.Contains(t, slashPath, ".autopus/runtime/memindex/")
	assert.NotContains(t, slashPath, ".autopus/project/")
	assert.NotContains(t, slashPath, ".autopus/specs/")

	fts5Probe := asMap(t, firstData["fts5_probe"])
	assert.Equal(t, "ok", fts5Probe["status"])
	assert.Equal(t, true, fts5Probe["probed_before_write"])
}

func makeProjectionFixture(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, ".autopus", "project", "product.md"), `# Product

Decision memory should be rebuildable from canonical project documents.
`)
	writeFile(t, filepath.Join(projectDir, ".autopus", "specs", "SPEC-TEST-001", "spec.md"), `# SPEC-TEST-001

**Status**: approved
**Created**: 2026-05-06

## Decision
Use SQLite FTS5 only as a projection.
`)
	writeFile(t, filepath.Join(projectDir, ".autopus", "learnings", "pipeline.jsonl"),
		`{"id":"L-001","timestamp":"2026-05-06T00:00:00Z","type":"gate_fail","phase":"review","spec_id":"SPEC-TEST-001","files":["pkg/memindex/index.go"],"packages":["pkg/memindex"],"pattern":"projection rebuild mismatch","resolution":"sort source refs before hashing","severity":"high","reuse_count":2}`+"\n")
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

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
