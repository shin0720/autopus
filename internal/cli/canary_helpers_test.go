package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrOrDefault(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("original error")
	assert.Equal(t, sentinel, errOrDefault(sentinel, "fallback"))
	assert.EqualError(t, errOrDefault(nil, "fallback message"), "fallback message")
}

func TestDefaultString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fallback", defaultString("", "fallback"))
	assert.Equal(t, "fallback", defaultString("   ", "fallback"))
	assert.Equal(t, "value", defaultString("value", "fallback"))
}

func TestDefaultString_NonEmptyReturnsSelf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "hello", defaultString("hello", "other"))
	assert.Equal(t, " x ", defaultString(" x ", "other"))
}

func TestCanarySummary(t *testing.T) {
	t.Parallel()

	result := canaryResult{
		Build:    "PASS",
		E2E:      "FAIL",
		Doctor:   "PASS",
		Endpoint: "SKIPPED",
		Browser:  "SKIPPED",
	}
	summary := canarySummary(result)

	assert.Equal(t, "PASS", summary["build"])
	assert.Equal(t, "FAIL", summary["e2e"])
	assert.Equal(t, "PASS", summary["doctor"])
	assert.Equal(t, "SKIPPED", summary["endpoint"])
	assert.Equal(t, "SKIPPED", summary["browser"])
}

func TestCanaryChecks(t *testing.T) {
	t.Parallel()

	result := canaryResult{
		Summary: map[string]string{
			"build": "PASS",
			"e2e":   "FAIL",
		},
	}
	checks := canaryChecks(result)

	require.Len(t, checks, 2)
	ids := make(map[string]string)
	for _, c := range checks {
		ids[c.ID] = c.Status
	}
	assert.Equal(t, "pass", ids["canary.build"])
	assert.Equal(t, "fail", ids["canary.e2e"])
}

func TestCanaryChecks_EmptySummary(t *testing.T) {
	t.Parallel()

	checks := canaryChecks(canaryResult{Summary: map[string]string{}})
	assert.Empty(t, checks)
}

func TestPrintCanaryText(t *testing.T) {
	t.Parallel()

	result := canaryResult{
		Verdict: "PASS",
		Summary: map[string]string{"build": "PASS"},
	}
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	printCanaryText(cmd, result)

	output := out.String()
	assert.Contains(t, output, "canary PASS")
	assert.Contains(t, output, "build: PASS")
}

func TestWriteCanaryLatest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result := canaryResult{
		Timestamp: "2024-01-01T00:00:00Z",
		Project:   "test-project",
		Verdict:   "PASS",
		Summary:   map[string]string{"build": "PASS"},
	}

	err := writeCanaryLatest(dir, result)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".autopus", "canary", "latest.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-project")
	assert.Contains(t, string(data), "PASS")
}

func TestWriteCanaryLatest_FailsOnBadDir(t *testing.T) {
	t.Parallel()

	notADir := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o644))

	err := writeCanaryLatest(notADir, canaryResult{Verdict: "PASS"})
	assert.Error(t, err)
}

func TestCanaryBuildTargets_ReturnsExpectedIDs(t *testing.T) {
	t.Parallel()

	targets := canaryBuildTargets("/tmp/workspace")

	ids := make([]string, 0, len(targets))
	for _, tgt := range targets {
		ids = append(ids, tgt.ID)
	}
	assert.Contains(t, ids, "H1")
	assert.Contains(t, ids, "H2")
	assert.Contains(t, ids, "H3")
	assert.Contains(t, ids, "H4")
	assert.Contains(t, ids, "H5a")
	assert.Contains(t, ids, "H5b")
}
