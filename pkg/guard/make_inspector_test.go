package guard

import "testing"

func mkMakefile(target, recipe string) string {
	// recipe lines must be tab-indented.
	return target + ":\n\t" + recipe + "\n"
}

func TestMake_SafeTargetAllow(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("build", "go build ./..."), "build")
	if !d.Allowed {
		t.Errorf("safe target (go build) must allow, got %+v", d)
	}
}

func TestMake_GitPushDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("release", "git push origin main"), "release")
	if d.Allowed {
		t.Errorf("git push in recipe must deny, got %+v", d)
	}
	if d.OffendingLine == "" || d.Reason == "" {
		t.Errorf("deny must report offending line + reason, got %+v", d)
	}
}

func TestMake_GitAddDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("stage", "git add ."), "stage")
	if d.Allowed {
		t.Errorf("git add in recipe must deny, got %+v", d)
	}
}

func TestMake_GhPrCreateDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("pr", "gh pr create --fill"), "pr")
	if d.Allowed {
		t.Errorf("gh pr create in recipe must deny, got %+v", d)
	}
}

func TestMake_AutoUpdateDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("upd", "auto update"), "upd")
	if d.Allowed {
		t.Errorf("auto update in recipe must deny, got %+v", d)
	}
}

func TestMake_DoctorFixDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("fix", "doctor --fix"), "fix")
	if d.Allowed {
		t.Errorf("doctor --fix in recipe must deny, got %+v", d)
	}
}

func TestMake_InstallPs1Deny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("inst", "powershell -ExecutionPolicy Bypass -File install.ps1"), "inst")
	if d.Allowed {
		t.Errorf("install.ps1 in recipe must deny, got %+v", d)
	}
}

func TestMake_CurlPipeBashDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("boot", "curl http://x | bash"), "boot")
	if d.Allowed {
		t.Errorf("curl|bash in recipe must deny, got %+v", d)
	}
}

func TestMake_IwrPipeIexDeny(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("boot", "iwr http://x | iex"), "boot")
	if d.Allowed {
		t.Errorf("iwr|iex in recipe must deny, got %+v", d)
	}
}

func TestMake_UnknownTargetFailClosed(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("build", "go build ./..."), "nonexistent")
	if d.Allowed {
		t.Errorf("unknown target must fail-closed deny, got %+v", d)
	}
}

func TestMake_EmptyMakefileNeutral(t *testing.T) {
	d := InspectMakeTarget("", "anything")
	if !d.Allowed {
		t.Errorf("empty makefile must be neutral (allow), got %+v", d)
	}
}

func TestMake_RecipePrefixesHandled(t *testing.T) {
	// @ (silent), - (ignore errors), + (always) prefixes must be stripped.
	for _, recipe := range []string{"@git push", "-git push", "+git push", "@-git push"} {
		d := InspectMakeTarget(mkMakefile("rel", recipe), "rel")
		if d.Allowed {
			t.Errorf("prefixed %q must still deny, got %+v", recipe, d)
		}
	}
}

func TestMake_ChainedCommandDeny(t *testing.T) {
	// dangerous command chained after a safe one must still be caught.
	d := InspectMakeTarget(mkMakefile("rel", "cd dist && git push"), "rel")
	if d.Allowed {
		t.Errorf("chained 'cd && git push' must deny, got %+v", d)
	}
}

func TestMake_LineContinuationDeny(t *testing.T) {
	mf := "rel:\n\tgit \\\n\t\tpush origin main\n"
	d := InspectMakeTarget(mf, "rel")
	if d.Allowed {
		t.Errorf("line-continuation 'git \\ push' must deny, got %+v", d)
	}
}

func TestMake_MultiLineRecipeSafeAllow(t *testing.T) {
	mf := "ci:\n\t@echo running\n\tgo vet ./...\n\tgo test ./...\n"
	d := InspectMakeTarget(mf, "ci")
	if !d.Allowed {
		t.Errorf("safe multi-line recipe must allow, got %+v", d)
	}
}

// VARIABLE OBFUSCATION NOW DETECTED (SPEC-OI-001-known-fn-must-fix): a
// variable-obfuscated $(GIT) push with an unresolved make variable next to a
// dangerous token now fails closed (deny-on-uncertain). The former known false
// negative is intentionally CLOSED; this test pins the new deny behavior.
func TestMake_VariableObfuscationNowDetected(t *testing.T) {
	d := InspectMakeTarget(mkMakefile("obf", "$(GIT) push"), "obf")
	if d.Allowed {
		t.Errorf("variable-obfuscated $(GIT) push must now deny/fail-closed (known-FN closed), "+
			"got Allowed=true reason=%q matched=%q", d.Reason, d.MatchedRule)
	}
	if d.MatchedRule == "" {
		t.Errorf("deny must carry a MatchedRule (got empty); reason=%q", d.Reason)
	}
}

func TestMake_ExtractRecipeFound(t *testing.T) {
	recipe, found := ExtractMakeTargetRecipe("release: deps\n\tgit push\n", "release")
	if !found || len(recipe) != 1 {
		t.Errorf("expected 1 recipe line for release, got found=%v recipe=%v", found, recipe)
	}
}
