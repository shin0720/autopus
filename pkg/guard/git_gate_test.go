package guard

import "testing"

func denyCase(t *testing.T, exe string, args []string, wantCat string) {
	t.Helper()
	d := EvaluateGitGate(exe, args)
	if d.Allowed || d.Category != wantCat || d.MatchedRule == "" {
		t.Errorf("%s %v should deny in category %q, got %+v", exe, args, wantCat, d)
	}
}

func allowCandidate(t *testing.T, exe string, args []string, wantCat string) {
	t.Helper()
	d := EvaluateGitGate(exe, args)
	if !d.Allowed || d.Category != wantCat {
		t.Errorf("%s %v should be allow candidate in category %q, got %+v", exe, args, wantCat, d)
	}
}

func TestGitGate_GitAddDeny(t *testing.T)       { denyCase(t, "git", []string{"add", "."}, "git") }
func TestGitGate_GitCommitDeny(t *testing.T)    { denyCase(t, "git", []string{"commit", "-m", "x"}, "git") }
func TestGitGate_GitPushDeny(t *testing.T)      { denyCase(t, "git", []string{"push"}, "git") }
func TestGitGate_GitMergeDeny(t *testing.T)     { denyCase(t, "git", []string{"merge", "main"}, "git") }
func TestGitGate_GitRebaseDeny(t *testing.T)    { denyCase(t, "git", []string{"rebase", "main"}, "git") }
func TestGitGate_GitResetDeny(t *testing.T)     { denyCase(t, "git", []string{"reset", "--hard"}, "git") }
func TestGitGate_GitCleanDeny(t *testing.T)     { denyCase(t, "git", []string{"clean", "-fd"}, "git") }
func TestGitGate_GitRemoteSetURLDeny(t *testing.T) {
	denyCase(t, "git", []string{"remote", "set-url", "origin", "http://x"}, "git")
}
func TestGitGate_GitBranchDDeny(t *testing.T)   { denyCase(t, "git", []string{"branch", "-D", "feat"}, "git") }
func TestGitGate_GhPrCreateDeny(t *testing.T)   { denyCase(t, "gh", []string{"pr", "create", "--fill"}, "gh") }
func TestGitGate_GhPrMergeDeny(t *testing.T)    { denyCase(t, "gh", []string{"pr", "merge"}, "gh") }
func TestGitGate_AutoUpdateDeny(t *testing.T)   { denyCase(t, "auto", []string{"update"}, "auto") }
func TestGitGate_AutoInstallDeny(t *testing.T)  { denyCase(t, "auto", []string{"install"}, "auto") }
func TestGitGate_DoctorFixDeny(t *testing.T)    { denyCase(t, "doctor", []string{"--fix"}, "doctor") }

func TestGitGate_GitStatusAllow(t *testing.T)   { allowCandidate(t, "git", []string{"status", "-sb"}, "git") }
func TestGitGate_GitBranchShowCurrentAllow(t *testing.T) {
	allowCandidate(t, "git", []string{"branch", "--show-current"}, "git")
}
func TestGitGate_GitRemoteVAllow(t *testing.T)  { allowCandidate(t, "git", []string{"remote", "-v"}, "git") }

func TestGitGate_GoTestNeutral(t *testing.T) {
	// A non-category command is neutral: the gate neither allows nor denies it.
	d := EvaluateGitGate("go", []string{"test", "./pkg/guard/..."})
	if !d.Allowed || d.Category != "other" {
		t.Errorf("go test should be neutral (other), got %+v", d)
	}
}

func TestGitGate_AliasNormalizationDeny(t *testing.T) {
	// git.exe add -> normalized "git add" must still deny.
	denyCase(t, "git.exe", []string{"add"}, "git")
}

func TestGitGate_Helpers(t *testing.T) {
	if d, _ := IsDangerousGitCommand("git add ."); !d {
		t.Error("git add should be dangerous")
	}
	if d, _ := IsDangerousGitCommand("git status -sb"); d {
		t.Error("git status should not be dangerous")
	}
	if d, _ := IsDangerousGhCommand("gh pr create"); !d {
		t.Error("gh pr create should be dangerous")
	}
	if d, _ := IsDangerousAutoCommand("auto update"); !d {
		t.Error("auto update should be dangerous")
	}
	if d, _ := IsDangerousDoctorCommand("doctor --fix"); !d {
		t.Error("doctor --fix should be dangerous")
	}
	if ClassifyCommandCategory("go") != "other" {
		t.Error("go should classify as other")
	}
}
