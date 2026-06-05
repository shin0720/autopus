package guard

import "testing"

func validBridgeSpec() RulesetBridgeSpec {
	return RulesetBridgeSpec{
		CommandProfiles: []BridgeCommandProfileSpec{
			{ID: "ccp_readonly", AllowedCommands: []string{"git status -sb", "git"}, DeniedRegex: []string{`^git\s+add\b`}},
			{ID: "ccp_build_test", AllowedCommands: []string{"go"}},
		},
		WorkerProfiles: []BridgeWorkerCommandProfileSpec{
			{ID: "cwp_W01", AllowedProviders: []string{"claude", "codex", "gemini", "opencode"}, CommandProfileRef: "ccp_readonly", AllowedExecutables: []string{"git"}},
			{ID: "cwp_W06", AllowedProviders: []string{"claude"}, CommandProfileRef: "ccp_build_test", AllowedExecutables: []string{"go"}},
		},
	}
}

func TestBridge_BuildsProfileSet(t *testing.T) {
	ps, err := BuildProfileSet(validBridgeSpec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := ps["ccp_readonly"]; !ok {
		t.Error("ccp_readonly missing from ProfileSet")
	}
	if len(ps) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(ps))
	}
}

func TestBridge_BuildsProviderBindingSet(t *testing.T) {
	pbs, err := BuildProviderBindingSet(validBridgeSpec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cc, ok := pbs["claude-code"]
	if !ok {
		t.Fatal("claude-code missing from ProviderBindingSet")
	}
	hasReadonly, hasBuild := false, false
	for _, p := range cc.AllowedProfiles {
		if p == "ccp_readonly" {
			hasReadonly = true
		}
		if p == "ccp_build_test" {
			hasBuild = true
		}
	}
	if !hasReadonly || !hasBuild {
		t.Errorf("claude-code should accumulate both profile refs, got %+v", cc)
	}
}

func TestBridge_AllowedProvidersNormalized(t *testing.T) {
	pbs, _ := BuildProviderBindingSet(validBridgeSpec())
	if _, ok := pbs["claude-code"]; !ok {
		t.Error("claude must normalize to claude-code")
	}
	if _, ok := pbs["gemini-cli"]; !ok {
		t.Error("gemini must normalize to gemini-cli")
	}
	if _, ok := pbs["codex"]; !ok {
		t.Error("codex must remain codex")
	}
	if _, ok := pbs["claude"]; ok {
		t.Error("raw adapter id 'claude' must NOT be a key")
	}
}

func TestBridge_OpencodePreserved(t *testing.T) {
	pbs, err := BuildProviderBindingSet(validBridgeSpec())
	if err != nil {
		t.Fatalf("opencode is a known YAML id; should not error: %v", err)
	}
	if _, ok := pbs["opencode"]; !ok {
		t.Error("opencode must be preserved as a known YAML provider id")
	}
}

func TestBridge_UnknownProviderError(t *testing.T) {
	spec := validBridgeSpec()
	spec.WorkerProfiles[0].AllowedProviders = []string{"claude", "totally-unknown"}
	if _, err := BuildProviderBindingSet(spec); err == nil {
		t.Error("unknown provider must return validation error")
	}
}

func TestBridge_MissingProfileRefError(t *testing.T) {
	spec := validBridgeSpec()
	spec.WorkerProfiles[0].CommandProfileRef = ""
	if _, err := BuildProviderBindingSet(spec); err == nil {
		t.Error("missing cli_command_profile_ref must return validation error")
	}
}

func TestBridge_DanglingProfileRefError(t *testing.T) {
	spec := validBridgeSpec()
	spec.WorkerProfiles[0].CommandProfileRef = "ccp_nonexistent"
	if _, err := BuildProviderBindingSet(spec); err == nil {
		t.Error("dangling ccp ref must return validation error")
	}
}

func TestBridge_EmptyAllowedProvidersError(t *testing.T) {
	spec := validBridgeSpec()
	spec.WorkerProfiles[0].AllowedProviders = nil
	if _, err := BuildProviderBindingSet(spec); err == nil {
		t.Error("empty allowed_providers must fail closed (validation error)")
	}
}

func TestBridge_EmptyProfilePreservedAndM3FailClosed(t *testing.T) {
	spec := RulesetBridgeSpec{
		CommandProfiles: []BridgeCommandProfileSpec{{ID: "ccp_empty"}}, // no AllowedCommands
	}
	ps, err := BuildProfileSet(spec)
	if err != nil {
		t.Fatalf("empty profile should be preserved, not error: %v", err)
	}
	if _, ok := ps["ccp_empty"]; !ok {
		t.Fatal("ccp_empty must be preserved")
	}
	// M3 must fail-closed deny on an empty AllowedCommands profile.
	d := EvaluateProfile("ccp_empty", "git", []string{"status"}, ps)
	if d.Allowed {
		t.Errorf("empty profile must M3 fail-closed deny, got %+v", d)
	}
}

func TestBridge_DuplicateProfileIDError(t *testing.T) {
	spec := RulesetBridgeSpec{CommandProfiles: []BridgeCommandProfileSpec{
		{ID: "ccp_x", AllowedCommands: []string{"git"}},
		{ID: "ccp_x", AllowedCommands: []string{"go"}},
	}}
	if _, err := BuildProfileSet(spec); err == nil {
		t.Error("duplicate profile id must return validation error")
	}
}

func TestBridge_DuplicateWorkerIDError(t *testing.T) {
	spec := validBridgeSpec()
	spec.WorkerProfiles = append(spec.WorkerProfiles, BridgeWorkerCommandProfileSpec{
		ID: "cwp_W01", AllowedProviders: []string{"claude"}, CommandProfileRef: "ccp_readonly",
	})
	if _, err := BuildProviderBindingSet(spec); err == nil {
		t.Error("duplicate worker id must return validation error")
	}
}

func TestBridge_BuildGuardRulesetsAndValidate(t *testing.T) {
	ps, pbs, err := BuildGuardRulesets(validBridgeSpec())
	if err != nil || ps == nil || pbs == nil {
		t.Fatalf("BuildGuardRulesets should succeed, got ps=%v pbs=%v err=%v", ps, pbs, err)
	}
	if err := ValidateRulesetBridgeSpec(validBridgeSpec()); err != nil {
		t.Errorf("valid spec must validate, got %v", err)
	}
	bad := validBridgeSpec()
	bad.WorkerProfiles[0].AllowedProviders = []string{"nope"}
	if err := ValidateRulesetBridgeSpec(bad); err == nil {
		t.Error("invalid spec must fail validation")
	}
}

// The bridge output integrates with the facade: built rule sets drive M3/M4
// exactly like fixtures, confirming the bridge produces usable rule sets.
func TestBridge_OutputDrivesFacade(t *testing.T) {
	ps, pbs, err := BuildGuardRulesets(validBridgeSpec())
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	// claude-code + ccp_readonly + git status -> allow.
	d := EvaluateCommandGuard(CommandGuardRequest{
		Executable: "git", Args: []string{"status", "-sb"},
		ProviderID: "claude-code", ProfileID: "ccp_readonly",
		Profiles: ps, Providers: pbs,
	})
	if !d.Allowed {
		t.Errorf("bridge-built rule sets should allow git status, got %+v", d)
	}
}
