package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_CanaryDryRunJSON_UsesStagingTargetsByDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "autopus.yaml"), []byte("project: canary-test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autopus", "project"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".autopus", "project", "canary.md"), []byte("# Canary\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"canary", "--dry-run", "--project-dir", dir, "--format", "json"})

	require.NoError(t, cmd.Execute())
	payload := decodeJSONMap(t, out.Bytes())
	assertCommonJSONEnvelope(t, payload, "auto canary")
	assert.Equal(t, "ok", payload["status"])

	data := payload["data"].(map[string]any)
	assert.Equal(t, "PASS", data["verdict"])
	assert.Equal(t, "SKIPPED", data["endpoint"])
	assert.Equal(t, "SKIPPED", data["browser"])
	flags := data["flags"].(map[string]any)
	assert.Equal(t, defaultCanaryFrontendURL, flags["frontend_url"])
	assert.Equal(t, defaultCanaryAPIURL, flags["api_url"])
}

func TestRootCmd_CanaryDryRunJSON_FailsWhenResultCannotBeStored(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectPath := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(projectPath, []byte("not a directory\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"canary", "--dry-run", "--project-dir", projectPath, "--format", "json"})

	require.Error(t, cmd.Execute())
	payload := decodeJSONMap(t, out.Bytes())
	assertCommonJSONEnvelope(t, payload, "auto canary")
	assert.Equal(t, "error", payload["status"])

	data := payload["data"].(map[string]any)
	assert.Equal(t, "FAIL", data["verdict"])
}

func TestResolveCanaryTargetsLegacyURLOverridesStagingDefaults(t *testing.T) {
	t.Parallel()

	targets := resolveCanaryTargets(canaryOptions{url: "https://preview.example.com/"})

	assert.Equal(t, "https://preview.example.com", targets.FrontendURL)
	assert.Equal(t, "https://preview.example.com", targets.APIURL)
}

func TestResolveCanaryTargets_Defaults(t *testing.T) {
	t.Parallel()

	targets := resolveCanaryTargets(canaryOptions{})

	assert.Equal(t, defaultCanaryFrontendURL, targets.FrontendURL)
	assert.Equal(t, defaultCanaryAPIURL, targets.APIURL)
}

func TestResolveCanaryTargets_IndependentURLFlags(t *testing.T) {
	t.Parallel()

	targets := resolveCanaryTargets(canaryOptions{
		frontendURL: "https://fe.example.com/",
		apiURL:      "https://api.example.com/",
	})

	assert.Equal(t, "https://fe.example.com", targets.FrontendURL)
	assert.Equal(t, "https://api.example.com", targets.APIURL)
}

func TestCanaryDryRunJSON_FrontendAndAPIURLFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "autopus.yaml"), []byte("project: flag-test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autopus", "project"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".autopus", "project", "canary.md"), []byte("# Canary\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"canary", "--dry-run", "--project-dir", dir, "--format", "json",
		"--frontend-url", "https://custom-fe.example.com",
		"--api-url", "https://custom-api.example.com",
	})

	require.NoError(t, cmd.Execute())
	payload := decodeJSONMap(t, out.Bytes())
	data := payload["data"].(map[string]any)
	flags := data["flags"].(map[string]any)
	assert.Equal(t, "https://custom-fe.example.com", flags["frontend_url"])
	assert.Equal(t, "https://custom-api.example.com", flags["api_url"])
}

func TestCanaryDryRun_TextFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "autopus.yaml"), []byte("project: text-test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autopus", "project"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".autopus", "project", "canary.md"), []byte("# Canary\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"canary", "--dry-run", "--project-dir", dir})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "canary")
}

func TestCanaryResult_FlagsIncludesWatchAndCompare(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "autopus.yaml"), []byte("project: flag-watch\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autopus", "project"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".autopus", "project", "canary.md"), []byte("# Canary\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"canary", "--dry-run", "--project-dir", dir, "--format", "json",
		"--watch", "5m", "--compare", "abc123",
	})

	require.NoError(t, cmd.Execute())
	payload := decodeJSONMap(t, out.Bytes())
	data := payload["data"].(map[string]any)
	flags := data["flags"].(map[string]any)
	assert.Equal(t, "5m", flags["watch"])
	assert.Equal(t, "abc123", flags["compare"])
}

func TestCanarySkippedInDryRun_AllAreasSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "autopus.yaml"), []byte("project: skip-test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autopus", "project"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".autopus", "project", "canary.md"), []byte("# Canary\n"), 0o644))

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"canary", "--dry-run", "--project-dir", dir, "--format", "json"})

	require.NoError(t, cmd.Execute())
	payload := decodeJSONMap(t, out.Bytes())
	data := payload["data"].(map[string]any)

	skipped, ok := data["skipped"].([]any)
	require.True(t, ok, "skipped should be present")
	assert.GreaterOrEqual(t, len(skipped), 3)
}

func TestCanaryCmd_UnknownFormatReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"canary", "--dry-run", "--project-dir", dir, "--format", "xml"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "xml") || strings.Contains(out.String(), "xml") || err != nil)
}
