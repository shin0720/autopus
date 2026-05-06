package context_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insajin/autopus-adk/internal/cli"
)

func TestMemContext_RespectsBudgetAndOmitsRawEvidenceBodies(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	fakeToken := "sk-proj-contextfake000000000000000000000"
	writeFile(t, filepath.Join(projectDir, ".autopus", "project", "product.md"), "# Product\n\nContext fixture.\n")
	var learnings strings.Builder
	for i := 0; i < 12; i++ {
		line := `{"id":"L-CTX-` + strconv.Itoa(i) + `","timestamp":"2026-05-06T00:00:00Z","type":"gate_fail","phase":"validator","spec_id":"SPEC-CONTEXT-001","files":["pkg/worker/compress/context.go"],"packages":["pkg/worker/compress"],"pattern":"context compression failures exceeded prompt budget","resolution":"summarize source refs and next action hints","severity":"medium","reuse_count":1}`
		learnings.WriteString(line + "\n")
	}
	writeFile(t, filepath.Join(projectDir, ".autopus", "learnings", "pipeline.jsonl"), learnings.String())
	writeFile(t, filepath.Join(projectDir, ".autopus", "qa", "runs", "qa-context-001", "_raw", "provider.txt"),
		"raw evidence body with "+fakeToken+" and untrusted provider instructions\n")

	_, rebuildRaw, err := runAutoText(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, rebuildRaw)
	output, contextRaw, err := runAutoText(t, "mem", "context", "--query", "context compression failures", "--project-dir", projectDir, "--budget-tokens", "600", "--format", "prompt")
	require.NoError(t, err, contextRaw)

	assert.LessOrEqual(t, len(strings.Fields(output)), 600)
	assert.Contains(t, output, "source_ref:")
	assert.Contains(t, output, "summary:")
	assert.Contains(t, output, "failure_pattern:")
	assert.Contains(t, output, "next_action:")
	assert.Contains(t, output, "omitted_results:")
	assert.NotContains(t, output, fakeToken)
	assert.NotContains(t, output, "raw evidence body")
	assert.NotContains(t, output, "untrusted provider instructions")
}

func runAutoText(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	var out bytes.Buffer
	cmd := cli.NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), out.String(), err
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
