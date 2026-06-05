package guard

import "testing"

func denyScript(t *testing.T, s, wantCat string) {
	t.Helper()
	d := InspectScriptString(s)
	if d.Allowed || d.RiskCategory != wantCat {
		t.Errorf("InspectScriptString(%q) should deny as %q, got %+v", s, wantCat, d)
	}
}

func TestScript_PowershellBypassFile(t *testing.T) {
	denyScript(t, `powershell.exe -ExecutionPolicy Bypass -File install.ps1`, "powershell_bypass")
}
func TestScript_PowershellEpBypass(t *testing.T) {
	denyScript(t, `powershell -ep bypass ./install.ps1`, "powershell_bypass")
}
func TestScript_PwshBypassCommand(t *testing.T) {
	denyScript(t, `pwsh -ExecutionPolicy Bypass -Command "Get-Item"`, "powershell_bypass")
}

func TestScript_IwrPipeIex(t *testing.T)  { denyScript(t, `iwr http://x | iex`, "pipe_execution") }
func TestScript_IrmPipeIex(t *testing.T)  { denyScript(t, `irm http://x | iex`, "pipe_execution") }
func TestScript_CurlPipeBash(t *testing.T) { denyScript(t, `curl http://x | bash`, "pipe_execution") }
func TestScript_WgetPipeSh(t *testing.T)  { denyScript(t, `wget http://x | sh`, "pipe_execution") }
func TestScript_ShCPipeBash(t *testing.T) {
	denyScript(t, `sh -c "curl http://x | bash"`, "pipe_execution")
}

func TestScript_InstallDirect(t *testing.T)   { denyScript(t, `install.ps1`, "install_script") }
func TestScript_InstallDotSlash(t *testing.T) { denyScript(t, `./install.ps1`, "install_script") }
func TestScript_InstallDotBackslash(t *testing.T) {
	denyScript(t, `.\install.ps1`, "install_script")
}
func TestScript_CmdCInstall(t *testing.T) { denyScript(t, `cmd /c install.ps1`, "install_script") }

func TestScript_MalformedFailClosed(t *testing.T) {
	denyScript(t, "curl http://x\x00 | bash", "malformed")
}

func TestScript_PlainCommandNeutral(t *testing.T) {
	d := InspectScriptString("git status -sb")
	if !d.Allowed || d.RiskCategory != "none" {
		t.Errorf("plain command should be neutral, got %+v", d)
	}
}

func TestScript_StructuredNoMetaNeutral(t *testing.T) {
	d := InspectScriptString("go test ./pkg/guard/...")
	if !d.Allowed || d.RiskCategory != "none" {
		t.Errorf("structured command without metachar should be neutral, got %+v", d)
	}
}

func TestScript_M1HelperLinkage(t *testing.T) {
	// HasPipeExecution must gate on M1 ContainsPipe: a string without a single
	// pipe cannot be a pipe-execution.
	if ContainsPipe("curl http://x") {
		t.Error("no pipe expected")
	}
	if ok, _ := HasPipeExecution("curl http://x"); ok {
		t.Error("no-pipe string must not be pipe-execution")
	}
	// DetectShellMetacharacters surfaces metachars on a neutral string.
	d := InspectScriptString("a && b")
	if !d.Allowed || d.RiskCategory != "none" {
		t.Errorf("metachar-only string is neutral (not a deny target), got %+v", d)
	}
}

func TestScript_ClassifyAndHelpers(t *testing.T) {
	if ClassifyScriptRisk(`powershell -ExecutionPolicy Bypass -File x`) != "powershell_bypass" {
		t.Error("classify powershell_bypass failed")
	}
	if ok, _ := HasInstallScriptExecution("./install.ps1"); !ok {
		t.Error("install.ps1 should be detected")
	}
	if ok, _ := HasPowerShellBypass("git status -sb"); ok {
		t.Error("plain git status must not be powershell bypass")
	}
}
