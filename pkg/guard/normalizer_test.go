package guard

import (
	"reflect"
	"testing"

	"github.com/shin0720/auto-adk/pkg/worker/security"
)

func TestNormalizeExecutable(t *testing.T) {
	cases := map[string]string{
		"git.exe":                      "git",
		"git":                          "git",
		`C:\Users\x\git.exe`:           "git",
		"./bin/auto":                   "auto",
		"PowerShell.EXE":               "powershell",
		`C:\WINDOWS\system32\auto.exe`: "auto",
	}
	for in, want := range cases {
		if got := NormalizeExecutable(in); got != want {
			t.Errorf("NormalizeExecutable(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNormalizeCommand_MultiSpace(t *testing.T) { // T05: git  add
	nc := NormalizeCommand("git", []string{"", "add"})
	if nc.CompareString != "git add" {
		t.Errorf("CompareString=%q want %q", nc.CompareString, "git add")
	}
}

func TestNormalizeCommand_ExeStrip(t *testing.T) { // T06: git.exe add
	nc := NormalizeCommand("git.exe", []string{"add"})
	if nc.NormalizedExecutable != "git" || nc.CompareString != "git add" {
		t.Errorf("got exe=%q compare=%q", nc.NormalizedExecutable, nc.CompareString)
	}
}

func TestNormalizeCommand_PathExe(t *testing.T) { // C:\...\git.exe add
	nc := NormalizeCommand(`C:\Users\x\git.exe`, []string{"add"})
	if nc.CompareString != "git add" {
		t.Errorf("CompareString=%q want %q", nc.CompareString, "git add")
	}
}

func TestNormalizeCommand_PathBasename(t *testing.T) { // ./bin/auto update
	nc := NormalizeCommand("./bin/auto", []string{"update"})
	if nc.CompareString != "auto update" {
		t.Errorf("CompareString=%q want %q", nc.CompareString, "auto update")
	}
}

func TestNormalizeCommand_PreservesOriginalArgs(t *testing.T) {
	args := []string{"A B/c.txt", "D"}
	nc := NormalizeCommand("cp", args)
	if !reflect.DeepEqual(nc.OriginalArgs, args) {
		t.Errorf("OriginalArgs=%v want %v (must be preserved verbatim)", nc.OriginalArgs, args)
	}
}

func TestDetectShellMetacharacters(t *testing.T) {
	cases := map[string]string{
		"a && b": "&&",
		"a || b": "||",
		"a ; b":  ";",
		"a `b`":  "`",
		"a $(b)": "$(",
	}
	for in, token := range cases {
		got := DetectShellMetacharacters(in)
		found := false
		for _, g := range got {
			if g == token {
				found = true
			}
		}
		if !found {
			t.Errorf("DetectShellMetacharacters(%q)=%v want contains %q", in, got, token)
		}
	}
}

func TestContainsPipe(t *testing.T) {
	if !ContainsPipe("a | b") {
		t.Error("expected single pipe in 'a | b'")
	}
	if ContainsPipe("a || b") {
		t.Error("'||' must not count as single pipe")
	}
	if ContainsPipe("ab") {
		t.Error("no pipe in 'ab'")
	}
}

func TestIsStructuredCommand(t *testing.T) {
	if !IsStructuredCommand("git", []string{"status", "-sb"}) {
		t.Error("git status -sb should be structured")
	}
	if IsStructuredCommand("sh", []string{"-c", "ls && rm"}) {
		t.Error("metacharacter command should not be structured")
	}
	if IsStructuredCommand("sh", []string{"-c", "a | b"}) {
		t.Error("pipe command should not be structured")
	}
}

func TestReuse_SecurityNormalizeRegression(t *testing.T) {
	got, err := security.NormalizeCommand("git\x00  add")
	if err != nil {
		t.Fatalf("security.NormalizeCommand err: %v", err)
	}
	if got != "git add" {
		t.Errorf("security.NormalizeCommand reuse=%q want %q", got, "git add")
	}
}
