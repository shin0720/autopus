package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanaryHTTPCheck_ReturnsTrue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	assert.True(t, canaryHTTPCheck(context.Background(), srv.URL+"/health"))
}

func TestCanaryHTTPCheck_ReturnsFalseOn404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	assert.False(t, canaryHTTPCheck(context.Background(), srv.URL+"/missing"))
}

func TestCanaryHTTPCheck_ReturnsFalseOnBadURL(t *testing.T) {
	t.Parallel()

	assert.False(t, canaryHTTPCheck(context.Background(), "http://127.0.0.1:0/unreachable"))
}

func TestCanaryHTTPCheck_Returns500AsFail(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	assert.False(t, canaryHTTPCheck(context.Background(), srv.URL+"/health"))
}

func TestCanaryHTTPCheck_Returns302AsPass(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	assert.True(t, canaryHTTPCheck(context.Background(), srv.URL+"/health"))
}

func TestRunCanaryEndpointChecks_AllPass(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := &canaryResult{}
	status := runCanaryEndpointChecks(context.Background(), srv.URL, result)

	assert.Equal(t, "PASS", status)
	assert.Len(t, result.Targets, 2)
}

func TestRunCanaryEndpointChecks_FailsOnBadURL(t *testing.T) {
	t.Parallel()

	result := &canaryResult{}
	status := runCanaryEndpointChecks(context.Background(), "http://127.0.0.1:0", result)

	assert.Equal(t, "FAIL", status)
}

func TestRunCanaryRemoteBrowser_SkippedWhenNoPkgJson(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result := &canaryResult{}

	status := runCanaryRemoteBrowser(context.Background(), dir, "https://example.com", result)

	assert.Equal(t, "SKIPPED", status)
	require.Len(t, result.Skipped, 1)
	assert.Equal(t, "browser", result.Skipped[0].Area)
}

func TestRunCanaryRemoteBrowser_SkipWithEmptyFrontendDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result := &canaryResult{}

	status := runCanaryRemoteBrowser(context.Background(), dir, "https://staging.example.com", result)

	assert.Equal(t, "SKIPPED", status)
}

func TestRunCanaryExternal_EmptyCommand(t *testing.T) {
	t.Parallel()

	result := runCanaryExternal(context.Background(), "T1", "empty", t.TempDir())

	assert.Equal(t, "FAIL", result.Status)
	assert.Equal(t, "empty command", result.Detail)
}

func TestRunCanaryExternal_FailsOnBadCommand(t *testing.T) {
	t.Parallel()

	result := runCanaryExternal(context.Background(), "T99", "no-such-binary-xyz", t.TempDir(), "no-such-binary-xyz")

	assert.Equal(t, "FAIL", result.Status)
	assert.Equal(t, "T99", result.ID)
}

func TestRunCanaryExternal_PassesOnTrueCommand(t *testing.T) {
	t.Parallel()

	result := runCanaryExternal(context.Background(), "T0", "echo ok", t.TempDir(), "echo", "ok")

	assert.Equal(t, "PASS", result.Status)
	assert.Empty(t, result.Detail)
}

func TestCanaryRunCanaryExternal_TimeoutReturnsFailDetail(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := runCanaryExternal(ctx, "T-timeout", "sleep 10", t.TempDir(), "sleep", "10")

	assert.Equal(t, "FAIL", result.Status)
}
