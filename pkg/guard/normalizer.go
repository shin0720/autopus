// Package guard implements SB8 command guards. P1 M1: command normalization.
//
// This wraps security.NormalizeCommand (reused, not duplicated) and adds
// executable normalization plus shell metacharacter / pipe detection.
package guard

import (
	"path/filepath"
	"strings"

	"github.com/insajin/autopus-adk/pkg/worker/security"
)

// NormalizedCommand holds the result of normalizing a command for SB8 guards.
// OriginalArgs is preserved verbatim; comparison fields are derived separately.
type NormalizedCommand struct {
	OriginalExecutable       string
	NormalizedExecutable     string
	OriginalArgs             []string
	NormalizedArgsForCompare []string
	CompareString            string
	HasPipe                  bool
	Metacharacters           []string
}

var executableExtensions = []string{".exe", ".cmd", ".bat", ".com"}

// NormalizeExecutable reduces an executable reference to a bare, lowercase name:
// strips directory prefix (both / and \) and a trailing executable extension.
func NormalizeExecutable(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\\", "/")
	s = filepath.Base(s)
	lower := strings.ToLower(s)
	for _, ext := range executableExtensions {
		if strings.HasSuffix(lower, ext) {
			s = s[:len(s)-len(ext)]
			break
		}
	}
	return strings.ToLower(s)
}

// ContainsPipe reports whether input contains a single pipe "|" that is not
// part of a "||" sequence (|| is treated as a shell metacharacter instead).
func ContainsPipe(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] != '|' {
			continue
		}
		prevPipe := i > 0 && input[i-1] == '|'
		nextPipe := i+1 < len(input) && input[i+1] == '|'
		if prevPipe || nextPipe {
			continue
		}
		return true
	}
	return false
}

// DetectShellMetacharacters returns the shell metacharacters present in input.
func DetectShellMetacharacters(input string) []string {
	tokens := []string{"&&", "||", ";", "`", "$("}
	var found []string
	for _, t := range tokens {
		if strings.Contains(input, t) {
			found = append(found, t)
		}
	}
	return found
}

// IsStructuredCommand reports whether the command is a plain executable+args
// form free of shell strings, pipes, and metacharacters.
func IsStructuredCommand(executable string, args []string) bool {
	if strings.TrimSpace(executable) == "" {
		return false
	}
	if strings.ContainsAny(executable, " \t") {
		return false
	}
	joined := executable + " " + strings.Join(args, " ")
	if ContainsPipe(joined) {
		return false
	}
	if len(DetectShellMetacharacters(joined)) > 0 {
		return false
	}
	return true
}

// NormalizeCommand normalizes an executable+args pair. OriginalArgs is preserved
// verbatim; security.NormalizeCommand is reused for null-byte/NFC/path/whitespace.
func NormalizeCommand(executable string, args []string) NormalizedCommand {
	nc := NormalizedCommand{
		OriginalExecutable:   executable,
		OriginalArgs:         append([]string(nil), args...),
		NormalizedExecutable: NormalizeExecutable(executable),
	}

	joined := strings.TrimSpace(executable + " " + strings.Join(args, " "))
	normalized, err := security.NormalizeCommand(joined)
	if err != nil {
		normalized = joined
	}

	nc.HasPipe = ContainsPipe(normalized)
	nc.Metacharacters = DetectShellMetacharacters(normalized)

	fields := strings.Fields(normalized)
	if len(fields) > 1 {
		nc.NormalizedArgsForCompare = append([]string(nil), fields[1:]...)
	}
	parts := append([]string{nc.NormalizedExecutable}, nc.NormalizedArgsForCompare...)
	nc.CompareString = strings.TrimSpace(strings.Join(parts, " "))
	return nc
}
