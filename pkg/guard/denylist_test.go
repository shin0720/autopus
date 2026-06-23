package guard

import (
	"strings"
	"testing"
)

func baseRules() DenyRuleSet {
	return DenyRuleSet{
		AllowedCommands: []string{"git", "gh", "go", "auto", "doctor", "iwr", "curl", "powershell"},
		DeniedExact:     []string{"auto update", "doctor --fix"},
		DeniedRegex: []string{
			`^git\s+add\b`,
			`^gh\s+pr\s+(create|merge)\b`,
			`powershell(\.exe)?\s+.*-ExecutionPolicy\s+Bypass`,
			`(iwr|irm|curl|wget)\s+.+\|\s*(iex|bash|sh)`,
		},
	}
}

func TestDeny_GitAdd(t *testing.T) {
	d := EvaluateCommand("git", []string{"add", "."}, baseRules())
	if d.Allowed || d.MatchedPattern == "" {
		t.Errorf("git add should deny with matched_pattern, got %+v", d)
	}
}

func TestDeny_GitAddMultiSpace(t *testing.T) {
	d := EvaluateCommand("git", []string{"", "add"}, baseRules())
	if d.Allowed {
		t.Errorf("git  add should deny, got %+v", d)
	}
}

func TestDeny_GitExeAdd(t *testing.T) {
	d := EvaluateCommand("git.exe", []string{"add"}, baseRules())
	if d.Allowed {
		t.Errorf("git.exe add should deny, got %+v", d)
	}
}

func TestDeny_GhPrCreateFill(t *testing.T) {
	d := EvaluateCommand("gh", []string{"pr", "create", "--fill"}, baseRules())
	if d.Allowed {
		t.Errorf("gh pr create --fill should deny, got %+v", d)
	}
}

func TestDeny_PowershellBypass(t *testing.T) {
	d := EvaluateCommand("powershell.exe", []string{"-ExecutionPolicy", "Bypass", "-File", "install.ps1"}, baseRules())
	if d.Allowed {
		t.Errorf("powershell bypass should deny, got %+v", d)
	}
}

func TestDeny_IwrPipeIex(t *testing.T) {
	d := EvaluateDenylist("iwr http://x | iex", baseRules())
	if d.Allowed || d.MatchedPattern == "" {
		t.Errorf("iwr | iex should deny with pattern, got %+v", d)
	}
}

func TestDeny_CurlPipeBash(t *testing.T) {
	d := EvaluateDenylist("curl http://x | bash", baseRules())
	if d.Allowed {
		t.Errorf("curl | bash should deny, got %+v", d)
	}
}

func TestDeny_AutoUpdate(t *testing.T) {
	d := EvaluateCommand("auto", []string{"update"}, baseRules())
	if d.Allowed {
		t.Errorf("auto update should deny, got %+v", d)
	}
}

func TestDeny_DoctorFix(t *testing.T) {
	d := EvaluateCommand("doctor", []string{"--fix"}, baseRules())
	if d.Allowed {
		t.Errorf("doctor --fix should deny, got %+v", d)
	}
}

func TestAllow_GitStatus(t *testing.T) {
	d := EvaluateCommand("git", []string{"status", "-sb"}, baseRules())
	if !d.Allowed {
		t.Errorf("git status -sb should allow, got %+v", d)
	}
}

func TestAllow_GoTest(t *testing.T) {
	d := EvaluateCommand("go", []string{"test", "./..."}, baseRules())
	if !d.Allowed {
		t.Errorf("go test ./... should allow, got %+v", d)
	}
}

func TestFailClosed_EmptyAllowed(t *testing.T) {
	d := EvaluateDenylist("git status -sb", DenyRuleSet{})
	if d.Allowed {
		t.Errorf("empty AllowedCommands must fail-closed deny, got %+v", d)
	}
}

func TestMalformedRegex_FailClosed(t *testing.T) {
	rs := baseRules()
	rs.DeniedRegex = append(rs.DeniedRegex, "[invalid")
	d := EvaluateDenylist("git status -sb", rs)
	if d.Allowed {
		t.Errorf("malformed regex must fail-closed deny, got %+v", d)
	}
}

func TestLongRegex_FailClosed(t *testing.T) {
	rs := baseRules()
	rs.DeniedRegex = append(rs.DeniedRegex, strings.Repeat("a", 1100))
	d := EvaluateDenylist("git status -sb", rs)
	if d.Allowed {
		t.Errorf("oversized regex (>1024) must fail-closed deny, got %+v", d)
	}
}

func TestComplexRegex_SafeReturns(t *testing.T) {
	// RE2 (linear) + matchWithTimeout guard: must return without hanging.
	rs := baseRules()
	rs.DeniedRegex = append(rs.DeniedRegex, `(a+)+b`)
	_ = EvaluateDenylist("git status -sb", rs)
}

func TestMatchedPattern_Reported(t *testing.T) {
	d := EvaluateCommand("git", []string{"add"}, baseRules())
	if d.MatchedPattern == "" {
		t.Errorf("expected matched_pattern populated for git add, got %+v", d)
	}
}

func TestPipeEscape_Regression(t *testing.T) {
	// Regression guard against past \\| escape failure: \| (single) must match a pipe.
	rs := DenyRuleSet{
		AllowedCommands: []string{"curl"},
		DeniedRegex:     []string{`(iwr|irm|curl|wget)\s+.+\|\s*(iex|bash|sh)`},
	}
	d := EvaluateDenylist("curl http://x | sh", rs)
	if d.Allowed {
		t.Errorf("pipe-escape pattern must deny 'curl | sh', got %+v", d)
	}
}
