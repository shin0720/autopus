package guard

import "testing"

func TestProviderIDMap_ClaudeToClaudeCode(t *testing.T) {
	d := NormalizeProviderID("claude")
	if !d.Mapped || d.YAMLProviderID != "claude-code" || !d.AdapterPresent {
		t.Errorf("claude should map to claude-code with adapter present, got %+v", d)
	}
}

func TestProviderIDMap_CodexToCodex(t *testing.T) {
	d := NormalizeProviderID("codex")
	if !d.Mapped || d.YAMLProviderID != "codex" || !d.AdapterPresent {
		t.Errorf("codex should map to codex, got %+v", d)
	}
}

func TestProviderIDMap_GeminiToGeminiCli(t *testing.T) {
	d := NormalizeProviderID("gemini")
	if !d.Mapped || d.YAMLProviderID != "gemini-cli" || !d.AdapterPresent {
		t.Errorf("gemini should map to gemini-cli, got %+v", d)
	}
}

func TestProviderIDMap_ClaudeCodeStaysClaudeCode(t *testing.T) {
	d := NormalizeProviderID("claude-code")
	if !d.Mapped || d.YAMLProviderID != "claude-code" || !d.AdapterPresent {
		t.Errorf("claude-code should stay claude-code, got %+v", d)
	}
}

func TestProviderIDMap_GeminiCliStaysGeminiCli(t *testing.T) {
	d := NormalizeProviderID("gemini-cli")
	if !d.Mapped || d.YAMLProviderID != "gemini-cli" {
		t.Errorf("gemini-cli should stay gemini-cli, got %+v", d)
	}
}

func TestProviderIDMap_OpencodeKnownButNoAdapter(t *testing.T) {
	d := NormalizeProviderID("opencode")
	if !d.Mapped || d.YAMLProviderID != "opencode" || d.AdapterPresent {
		t.Errorf("opencode should be a known YAML id with NO adapter, got %+v", d)
	}
}

func TestProviderIDMap_UnknownFailsClosed(t *testing.T) {
	d := NormalizeProviderID("totally-unknown")
	if d.Mapped {
		t.Errorf("unknown provider must be unmapped/fail-closed, got %+v", d)
	}
}

func TestProviderIDMap_EmptyNeutral(t *testing.T) {
	d := NormalizeProviderID("")
	if d.Mapped || d.YAMLProviderID != "" {
		t.Errorf("empty provider must be neutral/unmapped, got %+v", d)
	}
}

func TestProviderIDMap_CaseNormalized(t *testing.T) {
	if d := NormalizeProviderID("  Claude "); !d.Mapped || d.YAMLProviderID != "claude-code" {
		t.Errorf("case/space should normalize to claude-code, got %+v", d)
	}
	if d := NormalizeProviderID("GEMINI"); !d.Mapped || d.YAMLProviderID != "gemini-cli" {
		t.Errorf("GEMINI should normalize to gemini-cli, got %+v", d)
	}
}

func TestProviderIDMap_Helpers(t *testing.T) {
	if !IsKnownAdapterProviderID("claude") || IsKnownAdapterProviderID("opencode") {
		t.Error("adapter id set mismatch (opencode has no adapter)")
	}
	if !IsKnownYAMLProviderID("opencode") || !IsKnownYAMLProviderID("claude-code") || IsKnownYAMLProviderID("claude") {
		t.Error("YAML id set mismatch")
	}
	if y, ok := MapAdapterProviderID("gemini"); !ok || y != "gemini-cli" {
		t.Errorf("MapAdapterProviderID(gemini) = %q,%v", y, ok)
	}
}

// mapping helper existence must NOT enable M3/M4: a ProviderID present without a
// ProviderBindingSet still skips M4 in the facade (allow candidate).
func TestProviderIDMap_DoesNotEnableEnforcement(t *testing.T) {
	d := EvaluateCommandGuard(CommandGuardRequest{
		Executable: "git", Args: []string{"status", "-sb"}, ProviderID: "claude-code",
	})
	if !d.Allowed || d.Phase != PhaseAllow {
		t.Errorf("provider-id present without ruleset must not enforce M4, got %+v", d)
	}
}
