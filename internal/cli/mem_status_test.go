package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStatusJSON_ReportsProjectionHealthAndRebuildRecommendation(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	specPath := filepath.Join(projectDir, ".autopus", "specs", "SPEC-STATUS-001", "spec.md")
	writeMemStatusFile(t, specPath, "# SPEC-STATUS-001\n\nstable status fixture\n")
	writeMemStatusFile(t, filepath.Join(projectDir, ".autopus", "project", "product.md"), "# Product\n\nStatus fixture.\n")
	writeMemStatusFile(t, filepath.Join(projectDir, ".codex", "trace.json"), "{}\n")
	writeMemStatusFile(t, filepath.Join(projectDir, ".opencode", "trace.json"), "{}\n")

	_, rebuildRaw, err := runMemRoot(t, "mem", "rebuild", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, rebuildRaw)
	require.NoError(t, os.WriteFile(specPath, []byte("# SPEC-STATUS-001\n\nchanged after rebuild\n"), 0o644))

	out, statusRaw, err := runMemRoot(t, "mem", "status", "--project-dir", projectDir, "--format", "json")
	require.NoError(t, err, statusRaw)
	payload := decodeJSONMap(t, []byte(out))
	assertCommonJSONEnvelope(t, payload, "auto mem status")
	data := payload["data"].(map[string]any)

	assert.Equal(t, "autopus.mem_index.v1", data["schema_version"])
	assert.Equal(t, true, data["projection_only"])
	assert.NotEmpty(t, data["index_path"])
	counts := data["counts_by_source_kind"].(map[string]any)
	assert.Equal(t, float64(1), counts["spec"])
	assert.Equal(t, float64(1), counts["project_doc"])
	skipped := data["skipped_counts_by_reason"].(map[string]any)
	assert.Equal(t, float64(2), skipped["generated_surface"])
	assert.Contains(t, data["stale_refs"], ".autopus/specs/SPEC-STATUS-001/spec.md")
	corrupt := data["corrupt_state"].(map[string]any)
	assert.Equal(t, false, corrupt["is_corrupt"])
	assert.Equal(t, true, data["rebuild_recommended"])
}

func TestMemRebuildRejectsIndexPathOutsideRuntime(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeMemStatusFile(t, filepath.Join(projectDir, ".autopus", "project", "product.md"), "# Product\n\nIndex path fixture.\n")
	outside := filepath.Join(projectDir, "..", "outside.sqlite")

	out, _, err := runMemRoot(t, "mem", "rebuild", "--project-dir", projectDir, "--index-path", "../outside.sqlite", "--format", "json")
	require.Error(t, err)
	payload := decodeJSONMap(t, []byte(out))
	assertCommonJSONEnvelope(t, payload, "auto mem rebuild")
	assert.Equal(t, "error", payload["status"])
	jsonErr := payload["error"].(map[string]any)
	assert.Equal(t, "index-path-outside-runtime", jsonErr["code"])
	assert.NoFileExists(t, outside)
}

func TestMemContextMissingQueryUsesJSONEnvelope(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	out, _, err := runMemRoot(t, "mem", "context", "--project-dir", projectDir, "--format", "json")
	require.Error(t, err)
	payload := decodeJSONMap(t, []byte(out))
	assertCommonJSONEnvelope(t, payload, "auto mem context")
	assert.Equal(t, "error", payload["status"])
	jsonErr := payload["error"].(map[string]any)
	assert.Equal(t, "mem_context_missing_query", jsonErr["code"])
}

func runMemRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), out.String(), err
}

func writeMemStatusFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
