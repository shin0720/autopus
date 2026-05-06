package orchestra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type safetyEvidenceBackend struct{}

func (safetyEvidenceBackend) Execute(_ context.Context, req ProviderRequest) (*ProviderResponse, error) {
	switch req.Provider {
	case "codex":
		return &ProviderResponse{
			Provider: "codex",
			Output:   "SECRET_TOKEN=raw-output",
			Error:    "raw stderr with /Users/example/private",
			Duration: 420 * time.Second,
			TimedOut: true,
		}, nil
	case "claude":
		return &ProviderResponse{
			Provider: "claude",
			Output:   `{"analysis":"complete"}`,
			Duration: 11 * time.Second,
		}, nil
	case "gemini":
		return &ProviderResponse{
			Provider: "gemini",
			Output:   `{"analysis":"complete"}`,
			Duration: 12 * time.Second,
		}, nil
	default:
		return nil, assert.AnError
	}
}

func (safetyEvidenceBackend) Name() string {
	return "safety-evidence"
}

func TestExecuteParallel_SafetyTimeoutEvidenceMarksContinuedProviders(t *testing.T) {
	t.Parallel()

	providers := []ProviderConfig{
		{Name: "codex", Binary: "codex", ExecutionTimeout: 420 * time.Second},
		{Name: "claude", Binary: "claude"},
		{Name: "gemini", Binary: "gemini"},
	}

	successes, failed, err := executeParallel(
		context.Background(),
		safetyEvidenceBackend{},
		providers,
		"prompt",
		"",
		"debater_r1",
		1,
		600,
	)

	require.NoError(t, err)
	require.Len(t, successes, 2)
	assert.Equal(t, "claude", successes[0].Provider)
	require.Len(t, failed, 1)

	codex := failed[0]
	assert.Equal(t, "codex", codex.Name)
	assert.Equal(t, "debater_r1", codex.Role)
	assert.Equal(t, "provider_execution_timeout", codex.TimeoutSource)
	assert.Equal(t, 420*time.Second, codex.ConfiguredDuration)
	assert.Equal(t, 420*time.Second, codex.ElapsedDuration)
	assert.Equal(t, "timeout", codex.FailureClass)
	assert.NotEmpty(t, codex.NextRemediation)
	assert.True(t, codex.OtherProvidersContinued)
	assert.Empty(t, codex.OutputPreview)
	assert.Empty(t, codex.StderrPreview)
	assert.NotContains(t, codex.Error, "SECRET_TOKEN")
	assert.NotContains(t, codex.Error, "/Users/example/private")

	data, marshalErr := json.Marshal(codex)
	require.NoError(t, marshalErr)
	text := string(data)
	assert.Contains(t, text, `"provider":"codex"`)
	assert.Contains(t, text, `"role":"debater_r1"`)
	assert.Contains(t, text, `"failure_class":"timeout"`)
	assert.Contains(t, text, `"timeout_source":"provider_execution_timeout"`)
	assert.Contains(t, text, `"configured_duration":420000000000`)
	assert.Contains(t, text, `"elapsed_duration":420000000000`)
	assert.Contains(t, text, `"other_providers_continued":true`)
	assert.NotContains(t, text, "SECRET_TOKEN")
	assert.NotContains(t, text, "/Users/example/private")
}
