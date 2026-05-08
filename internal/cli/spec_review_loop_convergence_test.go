package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insajin/autopus-adk/pkg/config"
	"github.com/insajin/autopus-adk/pkg/orchestra"
	"github.com/insajin/autopus-adk/pkg/spec"
)

func TestRunSpecReviewLoop_ConvergesOnDegradedPassWithOnlyAdvisoryFindings(t *testing.T) {
	dir := t.TempDir()
	specID := "SPEC-REVIEW-ADVISORY-001"
	specDir := scaffoldReviewSpec(t, dir, specID)
	doc, err := spec.Load(specDir)
	require.NoError(t, err)

	callCount := 0
	origRunner := specReviewRunOrchestra
	specReviewRunOrchestra = func(_ context.Context, _ orchestra.OrchestraConfig) (*orchestra.OrchestraResult, error) {
		callCount++
		return &orchestra.OrchestraResult{
			Responses: []orchestra.ProviderResponse{{
				Provider: "claude",
				Output:   `{"verdict":"PASS","summary":"Advisory only","findings":[{"severity":"suggestion","category":"completeness","scope_ref":"SPEC-REVIEW-ADVISORY-001","location":"SPEC-REVIEW-ADVISORY-001","description":"Consider adding a small example.","suggestion":"Optional wording improvement."}]}`,
			}},
			FailedProviders: []orchestra.FailedProvider{
				{Name: "codex", FailureClass: "timeout", Error: "timed out"},
				{Name: "gemini", FailureClass: "execution_error", Error: "empty output"},
			},
		}, nil
	}
	defer func() { specReviewRunOrchestra = origRunner }()

	result, err := runSpecReviewLoop(reviewLoopParams(specID, specDir), doc, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, spec.VerdictPass, result.Verdict)
	assert.Equal(t, 1, callCount, "advisory-only feedback must not force another review iteration")

	findings, err := spec.LoadFindings(specDir)
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, spec.FindingStatusDeferred, findings[0].Status)
	assert.False(t, spec.IsActiveBlockingFinding(findings[0]))
}

func TestRunSpecReviewLoop_PromotesReviseWithOnlyAdvisoryFindings(t *testing.T) {
	dir := t.TempDir()
	specID := "SPEC-REVIEW-SUGGESTION-001"
	specDir := scaffoldReviewSpec(t, dir, specID)
	doc, err := spec.Load(specDir)
	require.NoError(t, err)

	origRunner := specReviewRunOrchestra
	specReviewRunOrchestra = func(_ context.Context, _ orchestra.OrchestraConfig) (*orchestra.OrchestraResult, error) {
		return &orchestra.OrchestraResult{
			Responses: []orchestra.ProviderResponse{
				{
					Provider: "claude",
					Output:   `{"verdict":"REVISE","summary":"Only advisory feedback","findings":[{"severity":"suggestion","category":"style","scope_ref":"plan.md","location":"plan.md","description":"Consider tightening the wording.","suggestion":"Optional wording improvement."}]}`,
				},
				{Provider: "gemini", Output: `{"verdict":"PASS","summary":"ok","findings":[]}`},
			},
		}, nil
	}
	defer func() { specReviewRunOrchestra = origRunner }()

	result, err := runSpecReviewLoop(reviewLoopParams(specID, specDir), doc, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, spec.VerdictPass, result.Verdict)
}

func reviewLoopParams(specID, specDir string) specReviewLoopParams {
	return specReviewLoopParams{
		ctx:          context.Background(),
		specID:       specID,
		specDir:      specDir,
		strategy:     "debate",
		timeout:      10,
		maxRevisions: 3,
		threshold:    0.67,
		gate:         config.ReviewGateConf{},
		providers: []orchestra.ProviderConfig{
			{Name: "claude", Binary: "claude"},
			{Name: "codex", Binary: "codex"},
			{Name: "gemini", Binary: "gemini"},
		},
	}
}
