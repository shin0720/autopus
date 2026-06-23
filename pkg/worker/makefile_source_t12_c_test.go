package worker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// T12-C-2 worker-level missing-file fail-closed fixture.
//
// Scope: this fixture exercises ONLY the worker-side pure functions
// readBoundedMakefile / inspectMakeCommandFromWorker. It is the worker-level
// counterpart to the guard-level make_inspector_t12_c_test.go (T12-C-1).
//
// Why worker-level is required: guard.InspectMakeTarget("", target) returns a
// neutral ALLOW for empty makefile text, so "file absent" is NOT a deny at the
// guard layer. The fail-closed deny for a missing Makefile happens only in the
// worker file-sourcing step (readBoundedMakefile), which this fixture pins.
//
// Telemetry mapping note (NOT asserted here): when this deny flows through the
// worker hook (command_guard_hook.go), it is recorded as Phase=PhaseScriptInspector
// with t12FailClosed=true, which maps to guard_id "M6" in the telemetry record.
// The pure MakefileSourceDecision has NO GuardID field, so guard_id is documented
// here only and never asserted directly.
//
// Safety invariants: every exec.Cmd below is built as a struct literal (no
// exec.Command lookup) and is NEVER started — no cmd.Start / cmd.Run / cmd.Output
// / cmd.CombinedOutput, no actual make, no make --dry-run, no subprocess, no
// network. All filesystem setup is confined to t.TempDir() (auto-cleaned).

// makeCmdInDir builds a *exec.Cmd that looks like "make <args...>" running in dir,
// WITHOUT performing a PATH lookup and WITHOUT ever being executed. cmd.Path is a
// bare "make" token so isMakeExecutable recognizes it; cmd.Args[0] is the program
// name (skipped by extractMakeTarget) and the rest are the make targets.
func makeCmdInDir(dir string, args ...string) *exec.Cmd {
	full := append([]string{"make"}, args...)
	return &exec.Cmd{Path: "make", Args: full, Dir: dir}
}

func TestWorkerMissingFileT12C(t *testing.T) {
	cases := []struct {
		name string
		// category: missing-file | non-regular
		category string
		// falsePositiveRisk: policy_expected | needs_review (documentation only)
		falsePositiveRisk string
		setup             func(t *testing.T, dir string)
		target            string
		wantReason        string
	}{
		{
			name:              "MF-01_missing_file_empty_directory",
			category:          "missing-file",
			falsePositiveRisk: "policy_expected",
			setup:             nil, // empty t.TempDir(): no files at all
			target:            "release",
			wantReason:        "no Makefile",
		},
		{
			name:              "MF-02_missing_file_no_recognized_names",
			category:          "missing-file",
			falsePositiveRisk: "policy_expected",
			setup: func(t *testing.T, dir string) {
				// Decoy: a non-standard name must NOT satisfy the standard set.
				decoy := filepath.Join(dir, "Makefile.txt")
				if err := os.WriteFile(decoy, []byte("build:\n\tgo build\n"), 0o644); err != nil {
					t.Fatalf("setup MF-02 write decoy: %v", err)
				}
			},
			target:     "build",
			wantReason: "no Makefile",
		},
		{
			name:              "MF-03_non_regular_makefile_directory",
			category:          "non-regular",
			falsePositiveRisk: "needs_review",
			setup: func(t *testing.T, dir string) {
				// A directory named "Makefile" is non-regular -> conservative deny.
				if err := os.Mkdir(filepath.Join(dir, "Makefile"), 0o755); err != nil {
					t.Fatalf("setup MF-03 mkdir: %v", err)
				}
			},
			target:     "release",
			wantReason: "non-regular file Makefile",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, dir)
			}

			cmd := makeCmdInDir(dir, tc.target)
			dec := inspectMakeCommandFromWorker(cmd)

			// fail-closed: a missing / non-regular Makefile must be denied.
			if dec.Allowed {
				t.Fatalf("%s: expected Allowed=false (fail-closed), got Allowed=true (reason=%q)", tc.category, dec.Reason)
			}
			if !strings.Contains(dec.Reason, tc.wantReason) {
				t.Fatalf("%s: Reason=%q does not contain %q", tc.category, dec.Reason, tc.wantReason)
			}
			// Target is extracted before the file-sourcing step, so it survives the deny.
			if dec.Target != tc.target {
				t.Fatalf("%s: Target=%q, want %q", tc.category, dec.Target, tc.target)
			}
			// No standard Makefile was loaded on a fail-closed path.
			if dec.MakefileTag != "" {
				t.Fatalf("%s: MakefileTag=%q, want empty on fail-closed", tc.category, dec.MakefileTag)
			}
		})
	}
}

// TestWorkerMissingFileT12CReadBounded asserts the underlying readBoundedMakefile
// branch directly (independent of target extraction), so the missing-file and
// non-regular fail-closed strings are pinned at their source.
func TestWorkerMissingFileT12CReadBounded(t *testing.T) {
	t.Run("empty_directory", func(t *testing.T) {
		dir := t.TempDir()
		text, tag, err := readBoundedMakefile(dir)
		if err == nil {
			t.Fatalf("expected error for empty dir, got nil (text=%q tag=%q)", text, tag)
		}
		if !strings.Contains(err.Error(), "no Makefile") {
			t.Fatalf("err=%q does not contain %q", err.Error(), "no Makefile")
		}
		if text != "" || tag != "" {
			t.Fatalf("expected empty text/tag on fail-closed, got text=%q tag=%q", text, tag)
		}
	})

	t.Run("non_regular_directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "Makefile"), 0o755); err != nil {
			t.Fatalf("mkdir Makefile dir: %v", err)
		}
		_, _, err := readBoundedMakefile(dir)
		if err == nil {
			t.Fatalf("expected error for non-regular Makefile, got nil")
		}
		if !strings.Contains(err.Error(), "non-regular file Makefile") {
			t.Fatalf("err=%q does not contain %q", err.Error(), "non-regular file Makefile")
		}
	})

	t.Run("empty_workdir_string", func(t *testing.T) {
		// Distinct input from missing-file: an unset workDir fails closed too.
		_, _, err := readBoundedMakefile("")
		if err == nil {
			t.Fatalf("expected error for empty workDir, got nil")
		}
		if !strings.Contains(err.Error(), "workDir not set") {
			t.Fatalf("err=%q does not contain %q", err.Error(), "workDir not set")
		}
	})
}
