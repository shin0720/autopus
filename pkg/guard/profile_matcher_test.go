package guard

import "testing"

func baseProfiles() ProfileSet {
	return ProfileSet{
		"ccp_readonly": {
			AllowedCommands: []string{"git status -sb", "git branch -vv", "git"},
			DeniedRegex:     []string{`^git\s+add\b`},
		},
		"ccp_build_test": {
			AllowedCommands: []string{"go"},
			DeniedRegex:     []string{`go\s+mod\s+tidy`},
		},
	}
}

func TestProfile_AllowGitStatus(t *testing.T) {
	d := EvaluateProfile("ccp_readonly", "git", []string{"status", "-sb"}, baseProfiles())
	if !d.Allowed || d.ProfileID != "ccp_readonly" {
		t.Errorf("git status -sb should allow in ccp_readonly, got %+v", d)
	}
}

func TestProfile_AllowGoTest(t *testing.T) {
	d := EvaluateProfile("ccp_build_test", "go", []string{"test", "./..."}, baseProfiles())
	if !d.Allowed {
		t.Errorf("go test ./... should allow in ccp_build_test, got %+v", d)
	}
}

func TestProfile_DenyGitAdd(t *testing.T) {
	d := EvaluateProfile("ccp_readonly", "git", []string{"add", "."}, baseProfiles())
	if d.Allowed {
		t.Errorf("git add should deny in ccp_readonly, got %+v", d)
	}
}

func TestProfile_DenyPrecedesAllow(t *testing.T) {
	// "git add" matches allow prefix "git " but the denylist must win.
	d := EvaluateProfile("ccp_readonly", "git", []string{"add"}, baseProfiles())
	if d.Allowed || d.MatchedRule == "" {
		t.Errorf("denylist must precede allow, got %+v", d)
	}
}

func TestProfile_UnknownDeny(t *testing.T) {
	d := EvaluateProfile("ccp_nope", "git", []string{"status", "-sb"}, baseProfiles())
	if d.Allowed {
		t.Errorf("unknown profile must fail-closed deny, got %+v", d)
	}
}

func TestProfile_EmptyDeny(t *testing.T) {
	ps := ProfileSet{"empty": {AllowedCommands: nil}}
	d := EvaluateProfile("empty", "git", []string{"status"}, ps)
	if d.Allowed {
		t.Errorf("empty profile must fail-closed deny, got %+v", d)
	}
}

func TestProfile_AliasNormalization(t *testing.T) {
	// git.exe status -sb -> normalized "git status -sb"
	d := EvaluateProfile("ccp_readonly", "git.exe", []string{"status", "-sb"}, baseProfiles())
	if !d.Allowed {
		t.Errorf("git.exe status -sb should allow via M1 normalization, got %+v", d)
	}
}

func TestProfile_ArgsPrefixAllow(t *testing.T) {
	d := EvaluateProfile("ccp_build_test", "go", []string{"build", "./..."}, baseProfiles())
	if !d.Allowed || d.MatchedRule != "go" {
		t.Errorf("go build should allow via prefix rule 'go', got %+v", d)
	}
}

func TestProfile_ExactAllow(t *testing.T) {
	d := EvaluateProfile("ccp_readonly", "git", []string{"branch", "-vv"}, baseProfiles())
	if !d.Allowed || d.MatchedRule != "git branch -vv" {
		t.Errorf("git branch -vv should allow via exact rule, got %+v", d)
	}
}

func TestProfile_MatchedProfileAndRule(t *testing.T) {
	d := EvaluateProfile("ccp_readonly", "git", []string{"status", "-sb"}, baseProfiles())
	if d.ProfileID != "ccp_readonly" || d.MatchedRule == "" {
		t.Errorf("expected ProfileID + MatchedRule populated, got %+v", d)
	}
}

func TestMatchAllowedRule(t *testing.T) {
	allowed := []string{"git status -sb", "go"}
	if MatchAllowedRule("git status -sb", allowed) != "git status -sb" {
		t.Error("exact match failed")
	}
	if MatchAllowedRule("go test ./...", allowed) != "go" {
		t.Error("prefix match failed")
	}
	if MatchAllowedRule("rm -rf", allowed) != "" {
		t.Error("non-match should be empty")
	}
}
