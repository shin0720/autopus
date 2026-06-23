package worker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shin0720/auto-adk/pkg/guard"
)

// T12 runtime Makefile-source carve-out (worker-only). The ONLY os.ReadFile
// site introduced for SB8: a make-command invocation triggers a read-only load
// of the workspace Makefile under strict policy. Other code paths must NOT add
// file I/O.
//
// Policy (must hold or fail closed):
//   - cmd.Path basename is "make" (or "make.exe") — non-make cmds are unaffected.
//   - cmd.Args target is the first non-flag arg; "-f"/"--file"/"--makefile"
//     (user-specified makefile) is rejected.
//   - workDir = cmd.Dir; empty workDir is rejected.
//   - Filename restricted to Makefile / makefile / GNUmakefile (standard order).
//   - Resolved path stays inside the workDir boundary (filepath.Rel check).
//   - os.Lstat: only regular files; symlinks / dirs / special files rejected.
//   - Size <= 1MB; oversize rejected.
//   - os.ReadFile errors -> rejected.
//   - InspectMakeTarget runs the static analysis (M5/M6 reuse). Dangerous recipe
//     or unknown target -> denied.

const maxMakefileSize int64 = 1 << 20 // 1 MiB

var standardMakefileNames = []string{"Makefile", "makefile", "GNUmakefile"}

// MakefileSourceDecision is the worker-side T12 runtime decision.
type MakefileSourceDecision struct {
	Allowed     bool
	Target      string
	MakefileTag string // which standard filename was loaded
	MatchedRule string
	Reason      string
}

func isMakeExecutable(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Path == "" {
		return false
	}
	name := strings.ToLower(filepath.Base(cmd.Path))
	name = strings.TrimSuffix(name, ".exe")
	return name == "make"
}

// extractMakeTarget returns the first non-flag target, or an error for
// -f/--file/--makefile (user-specified) or when no target is given. Flag args
// like "-j4" / "--debug" are skipped.
func extractMakeTarget(args []string) (string, error) {
	// args[0] is the program name; iterate from args[1:].
	for i := 1; i < len(args); i++ {
		a := args[i]
		if a == "-f" || a == "--file" || a == "--makefile" {
			return "", errors.New("user-specified makefile (-f/--file/--makefile) not supported (fail-closed)")
		}
		if strings.HasPrefix(a, "--file=") || strings.HasPrefix(a, "--makefile=") {
			return "", errors.New("user-specified makefile (--file=/--makefile=) not supported (fail-closed)")
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a, nil
	}
	return "", errors.New("no make target specified (fail-closed)")
}

// readBoundedMakefile resolves and reads a standard Makefile inside workDir,
// returning the content and which filename matched. Every failure is fail-closed.
func readBoundedMakefile(workDir string) (string, string, error) {
	if strings.TrimSpace(workDir) == "" {
		return "", "", errors.New("worker workDir not set (fail-closed)")
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return "", "", fmt.Errorf("workDir abs (fail-closed): %w", err)
	}
	absWork = filepath.Clean(absWork)

	var lastErr error
	for _, name := range standardMakefileNames {
		p := filepath.Clean(filepath.Join(absWork, name))
		rel, relErr := filepath.Rel(absWork, p)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		info, lerr := os.Lstat(p)
		if lerr != nil {
			continue // try next standard name
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("symlink Makefile rejected (fail-closed): %s", name)
		}
		if !info.Mode().IsRegular() {
			return "", "", fmt.Errorf("non-regular file %s (fail-closed)", name)
		}
		if info.Size() > maxMakefileSize {
			return "", "", fmt.Errorf("%s exceeds %d bytes (fail-closed): %d", name, maxMakefileSize, info.Size())
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			lastErr = fmt.Errorf("unreadable Makefile %s (fail-closed): %w", name, rerr)
			continue
		}
		return string(data), name, nil
	}
	if lastErr != nil {
		return "", "", lastErr
	}
	return "", "", errors.New("no Makefile/makefile/GNUmakefile in workDir (fail-closed)")
}

// inspectMakeCommandFromWorker runs the full carve-out: target extraction,
// bounded Makefile load, then InspectMakeTarget. Any policy violation or
// dangerous recipe yields Allowed=false.
func inspectMakeCommandFromWorker(cmd *exec.Cmd) MakefileSourceDecision {
	target, err := extractMakeTarget(cmd.Args)
	if err != nil {
		return MakefileSourceDecision{Allowed: false, Reason: err.Error()}
	}
	text, name, err := readBoundedMakefile(cmd.Dir)
	if err != nil {
		return MakefileSourceDecision{Allowed: false, Target: target, Reason: err.Error()}
	}
	d := guard.InspectMakeTarget(text, target)
	if !d.Allowed {
		return MakefileSourceDecision{
			Allowed:     false,
			Target:      target,
			MakefileTag: name,
			MatchedRule: d.MatchedRule,
			Reason:      d.Reason + " (Makefile=" + name + ")",
		}
	}
	return MakefileSourceDecision{Allowed: true, Target: target, MakefileTag: name, Reason: "make target safe (Makefile=" + name + ")"}
}
