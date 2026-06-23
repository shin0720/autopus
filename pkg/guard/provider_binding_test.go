package guard

import "testing"

// baseProviders models the YAML allowed_providers space (claude-code, codex,
// gemini-cli, opencode) — NOT adapter.Name(). baseProfiles() is reused from
// profile_matcher_test.go (same package).
func baseProviders() ProviderBindingSet {
	return ProviderBindingSet{
		"claude-code": {AllowedProfiles: []string{"ccp_readonly", "ccp_build_test"}, AllowedExecutables: []string{"git", "go"}},
		"codex":       {AllowedProfiles: []string{"ccp_readonly", "ccp_build_test"}, AllowedExecutables: []string{"git", "go"}},
		"gemini-cli":  {AllowedProfiles: []string{"ccp_readonly"}, AllowedExecutables: []string{"git"}},
		"opencode":    {AllowedProfiles: []string{"ccp_readonly"}, AllowedExecutables: []string{"git"}},
	}
}

func TestProvider_ClaudeCodeAllow(t *testing.T) {
	d := EvaluateProviderBinding("claude-code", "ccp_readonly", "git", []string{"status", "-sb"}, baseProviders(), baseProfiles())
	if !d.Allowed || d.ProviderID != "claude-code" || d.ProfileID != "ccp_readonly" {
		t.Errorf("claude-code + ccp_readonly + git status should allow, got %+v", d)
	}
}

func TestProvider_CodexAllow(t *testing.T) {
	d := EvaluateProviderBinding("codex", "ccp_build_test", "go", []string{"test", "./..."}, baseProviders(), baseProfiles())
	if !d.Allowed {
		t.Errorf("codex + ccp_build_test + go test should allow, got %+v", d)
	}
}

func TestProvider_UnknownDeny(t *testing.T) {
	d := EvaluateProviderBinding("nope-provider", "ccp_readonly", "git", []string{"status"}, baseProviders(), baseProfiles())
	if d.Allowed {
		t.Errorf("unknown provider must fail-closed deny, got %+v", d)
	}
}

func TestProvider_ProfileNotAllowedDeny(t *testing.T) {
	// gemini-cli allows only ccp_readonly, not ccp_build_test.
	d := EvaluateProviderBinding("gemini-cli", "ccp_build_test", "git", []string{"status"}, baseProviders(), baseProfiles())
	if d.Allowed {
		t.Errorf("profile not allowed for provider must deny, got %+v", d)
	}
}

func TestProvider_ExecutableNotAllowedDeny(t *testing.T) {
	// gemini-cli allows only the git executable, not go.
	d := EvaluateProviderBinding("gemini-cli", "ccp_readonly", "go", []string{"test"}, baseProviders(), baseProfiles())
	if d.Allowed {
		t.Errorf("executable not allowed for provider must deny, got %+v", d)
	}
}

func TestProvider_AliasNormalization(t *testing.T) {
	// git.exe -> normalized "git", matched against provider AllowedExecutables.
	d := EvaluateProviderBinding("claude-code", "ccp_readonly", "git.exe", []string{"status", "-sb"}, baseProviders(), baseProfiles())
	if !d.Allowed || d.Executable != "git" {
		t.Errorf("git.exe should normalize to git and allow, got %+v", d)
	}
}

func TestProvider_ProfileMatcherCombine(t *testing.T) {
	d := EvaluateProviderBinding("claude-code", "ccp_build_test", "go", []string{"build", "./..."}, baseProviders(), baseProfiles())
	if !d.Allowed || d.ProfileID != "ccp_build_test" {
		t.Errorf("provider+profile combine should allow via M3, got %+v", d)
	}
}

func TestProvider_DenylistPrecedesProviderAllow(t *testing.T) {
	// claude-code allows the git executable + ccp_readonly, but "git add" is
	// denied by M2 denylist. The denylist must win over the provider allow.
	d := EvaluateProviderBinding("claude-code", "ccp_readonly", "git", []string{"add", "."}, baseProviders(), baseProfiles())
	if d.Allowed || d.MatchedRule == "" {
		t.Errorf("denylist must precede provider allow, got %+v", d)
	}
}

func TestMatchProviderExecutable(t *testing.T) {
	allowed := []string{"git", "go"}
	if !MatchProviderExecutable("git", allowed) {
		t.Error("git should match")
	}
	if MatchProviderExecutable("rm", allowed) {
		t.Error("rm should not match")
	}
}
