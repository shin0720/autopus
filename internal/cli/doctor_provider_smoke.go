package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/insajin/autopus-adk/internal/cli/tui"
	"github.com/insajin/autopus-adk/pkg/config"
	"github.com/insajin/autopus-adk/pkg/orchestra"
)

const providerSmokeMarker = "AUTOPUS_PROVIDER_SMOKE_OK"

type providerSmokeResult struct {
	Provider string
	Status   string
	Detail   string
}

var providerSmokeBackendFactory = orchestra.NewSubprocessBackendImpl

func checkProviderTransportSmokeText(w io.Writer, cfg *config.HarnessConfig, opts doctorOptions) bool {
	tui.SectionHeader(w, "Provider Transport")
	if !opts.providerSmoke {
		tui.SKIP(w, "provider transport smoke: skipped (run 'auto doctor --provider-smoke')")
		return true
	}

	results := runProviderTransportSmoke(context.Background(), cfg, opts.providerSmokeTimeout)
	allOK := true
	for _, result := range results {
		switch result.Status {
		case "pass":
			tui.OK(w, fmt.Sprintf("%s transport: %s", result.Provider, result.Detail))
		case "warn":
			tui.SKIP(w, fmt.Sprintf("%s transport: %s", result.Provider, result.Detail))
		default:
			tui.FAIL(w, fmt.Sprintf("%s transport: %s", result.Provider, result.Detail))
			allOK = false
		}
	}
	return allOK
}

func (r *doctorJSONReport) collectProviderTransportSmokeChecks(cfg *config.HarnessConfig, opts doctorOptions) {
	if !opts.providerSmoke {
		r.checks = append(r.checks, jsonCheck{
			ID:       "doctor.provider_transport.smoke",
			Severity: "info",
			Status:   "skip",
			Detail:   "provider transport smoke skipped; run 'auto doctor --provider-smoke'",
		})
		return
	}

	results := runProviderTransportSmoke(context.Background(), cfg, opts.providerSmokeTimeout)
	for _, result := range results {
		check := jsonCheck{
			ID:       "doctor.provider_transport." + result.Provider,
			Severity: "info",
			Status:   result.Status,
			Detail:   result.Detail,
		}
		if result.Status == "fail" {
			check.Severity = "error"
			r.status = jsonStatusWarn
			r.warnings = append(r.warnings, jsonMessage{
				Code:    "provider_transport_failed",
				Message: fmt.Sprintf("%s provider transport failed: %s", result.Provider, result.Detail),
			})
		}
		if result.Status == "warn" {
			check.Severity = "warning"
			r.status = jsonStatusWarn
		}
		r.checks = append(r.checks, check)
	}
}

func runProviderTransportSmoke(ctx context.Context, cfg *config.HarnessConfig, timeout time.Duration) []providerSmokeResult {
	if cfg == nil {
		return []providerSmokeResult{{Provider: "review_gate", Status: "fail", Detail: "config unavailable"}}
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	names := resolveSpecReviewProviderNames(cfg, false)
	providers := configureSpecReviewProviders(specReviewConfigProviders(cfg, names))
	if len(providers) == 0 {
		return []providerSmokeResult{{Provider: "review_gate", Status: "fail", Detail: "no installed review providers"}}
	}

	backend := providerSmokeBackendFactory()
	results := make([]providerSmokeResult, 0, len(providers))
	for _, provider := range providers {
		provider.OutputFormat = "text"
		resp, err := backend.Execute(ctx, orchestra.ProviderRequest{
			Provider: provider.Name,
			Prompt:   providerSmokePrompt(),
			Timeout:  timeout,
			Config:   provider,
		})
		results = append(results, classifyProviderSmokeResult(provider.Name, resp, err))
	}
	return results
}

func providerSmokePrompt() string {
	return "Return ONLY this exact token and no other text: " + providerSmokeMarker
}

func classifyProviderSmokeResult(provider string, resp *orchestra.ProviderResponse, err error) providerSmokeResult {
	if err != nil {
		return providerSmokeResult{Provider: provider, Status: "fail", Detail: err.Error()}
	}
	if resp == nil {
		return providerSmokeResult{Provider: provider, Status: "fail", Detail: "provider returned nil response"}
	}
	if resp.TimedOut {
		return providerSmokeResult{Provider: provider, Status: "fail", Detail: fmt.Sprintf("provider timed out after %s", resp.Duration)}
	}
	if resp.EmptyOutput || strings.TrimSpace(resp.Output) == "" {
		return providerSmokeResult{Provider: provider, Status: "fail", Detail: "provider returned empty output"}
	}
	if !strings.Contains(resp.Output, providerSmokeMarker) {
		return providerSmokeResult{Provider: provider, Status: "warn", Detail: "transport returned output but smoke marker was missing"}
	}
	return providerSmokeResult{Provider: provider, Status: "pass", Detail: "smoke marker returned"}
}
