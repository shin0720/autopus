// Package guard: SB8 T12 in-memory Make target inspector (pure static helper).
//
// InspectMakeTarget statically inspects a Makefile target's recipe for dangerous
// commands (git push/add, gh pr, auto, doctor, install.ps1, pipe-exec, powershell
// bypass) by REUSING the existing M5 (EvaluateGitGate) and M6 (InspectScriptString)
// detectors. It is PURE: it NEVER runs make (no make / no `make --dry-run`),
// reads NO files (makefileText is passed in), calls NO os.ReadFile, executes NO
// shell, and imports NO yaml. It does NOT wire into the facade/hook (runtime
// Makefile sourcing is a separate step requiring file I/O policy).
//
// VARIABLE HANDLING: simple one-step make variable assignments (NAME=value,
// NAME := value) are expanded once for $(NAME)/${NAME} before inspection. Anything
// that cannot be resolved this way — include directives, recursive $(MAKE),
// $(shell ...), conditionals, undefined or cyclic variables — is left UNRESOLVED
// and, when it sits next to a dangerous token, is denied (fail-closed). Full
// Makefile interpretation is NOT supported; this remains best-effort static
// analysis, but variable-obfuscated dangerous pushes now fail closed instead of
// being silently allowed.
package guard

import "strings"

// MakeTargetDecision is the result of statically inspecting a make target recipe.
type MakeTargetDecision struct {
	Allowed       bool
	Target        string
	MatchedRule   string // matched dangerous pattern/rule, when denied
	OffendingLine string // recipe line that triggered the deny
	Reason        string
}

// ExtractMakeTargetRecipe returns the tab-indented recipe lines for target (with
// line continuations merged). found=false if the target is not defined.
func ExtractMakeTargetRecipe(makefileText, target string) ([]string, bool) {
	lines := strings.Split(makefileText, "\n")
	prefix := strings.TrimSpace(target) + ":"
	var recipe []string
	inTarget := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !inTarget {
			if strings.HasPrefix(strings.TrimRight(line, " \t"), prefix) {
				inTarget = true
			}
			continue
		}
		if !strings.HasPrefix(line, "\t") {
			break // blank or column-0 line ends the rule
		}
		rl := strings.TrimPrefix(line, "\t")
		for strings.HasSuffix(strings.TrimRight(rl, " "), "\\") && i+1 < len(lines) {
			rl = strings.TrimSuffix(strings.TrimRight(rl, " "), "\\") + " " + strings.TrimSpace(lines[i+1])
			i++
		}
		recipe = append(recipe, rl)
	}
	return recipe, inTarget
}

// normalizeMakeRecipeLine strips the leading tab (already removed) and make recipe
// prefixes (@ silent, - ignore-errors, + always-run), returning the bare command.
func normalizeMakeRecipeLine(rl string) string {
	rl = strings.TrimSpace(rl)
	for len(rl) > 0 && (rl[0] == '@' || rl[0] == '-' || rl[0] == '+') {
		rl = strings.TrimSpace(rl[1:])
	}
	return rl
}

// splitRecipeSegments splits a recipe line on &&, ||, ; so a dangerous command
// chained after another (e.g. "cd x && git push") is still inspected. Single "|"
// is left intact so M6 pipe-execution detection sees the whole pipeline.
func splitRecipeSegments(line string) []string {
	r := strings.NewReplacer("&&", "\x00", "||", "\x00", ";", "\x00")
	return strings.Split(r.Replace(line), "\x00")
}

// inspectMakeRecipeLine reports whether a single (normalized) recipe line contains
// a dangerous command, reusing M6 (raw structure) then M5 (git/gh/auto/doctor).
func inspectMakeRecipeLine(line string) (bool, string, string) {
	if sd := InspectScriptString(line); !sd.Allowed {
		return true, sd.RiskCategory, sd.Reason
	}
	for _, seg := range splitRecipeSegments(line) {
		fields := strings.Fields(seg)
		if len(fields) == 0 {
			continue
		}
		if gd := EvaluateGitGate(fields[0], fields[1:]); !gd.Allowed {
			return true, gd.MatchedRule, gd.Reason
		}
	}
	return false, "", ""
}

// isSimpleVarName reports whether s is a plain make identifier (letters, digits,
// underscore). Anything else is not treated as a simple assignment target.
func isSimpleVarName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return true
}

// parseSimpleMakeVars extracts simple one-step variable assignments (NAME=value or
// NAME := value) from column-0 lines. Values are kept verbatim with NO recursive
// resolution, so a value that itself contains $(...) stays unresolved. Recipe lines
// (tab-indented), comments, append (+=) and other forms are ignored.
func parseSimpleMakeVars(makefileText string) map[string]string {
	vars := map[string]string{}
	for _, raw := range strings.Split(makefileText, "\n") {
		if strings.HasPrefix(raw, "\t") {
			continue // recipe line, not an assignment
		}
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 || (eq > 0 && line[eq-1] == '+') { // skip "+=" append form
			continue
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[:eq]), ":"))
		if !isSimpleVarName(name) {
			continue
		}
		vars[name] = strings.TrimSpace(line[eq+1:])
	}
	return vars
}

// expandSimpleMakeVarsOnce substitutes $(NAME) and ${NAME} for the known simple
// variables EXACTLY ONCE. It does not recurse: a substituted value that still
// contains $(...) is left unresolved (the caller treats that as uncertain).
func expandSimpleMakeVarsOnce(vars map[string]string, line string) string {
	for name, val := range vars {
		line = strings.ReplaceAll(line, "$("+name+")", val)
		line = strings.ReplaceAll(line, "${"+name+"}", val)
	}
	return line
}

// hasUnresolvedMakeVar reports whether line still references a make variable.
func hasUnresolvedMakeVar(line string) bool {
	return strings.Contains(line, "$(") || strings.Contains(line, "${")
}

// hasDangerousContext reports whether line carries a dangerous mutation/install
// token or a shell separator. Combined with an unresolved variable this triggers
// the deny-on-uncertain fail-closed path.
func hasDangerousContext(line string) bool {
	lower := strings.ToLower(line)
	for _, tok := range []string{
		"push", "--force", "force", "remote", "install",
		"curl", "wget", "invoke-webrequest", "invoke-restmethod",
		" add", " commit", " reset", " clean", " rebase", " merge",
	} {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	for _, sep := range []string{"|", ";", "&&", "||"} {
		if strings.Contains(line, sep) {
			return true
		}
	}
	return false
}

// InspectMakeTarget statically inspects target's recipe. Empty makefileText is
// neutral (no make context). An unknown target fails closed. Recipe lines are
// expanded once for simple make variables, then inspected: a resolved dangerous
// command denies with reason "git_indirect via=make"; an unresolved variable next
// to a dangerous token denies via deny-on-uncertain (fail-closed).
func InspectMakeTarget(makefileText, target string) MakeTargetDecision {
	if strings.TrimSpace(makefileText) == "" {
		return MakeTargetDecision{Allowed: true, Target: target, Reason: "empty makefile (neutral; no make context)"}
	}
	recipe, found := ExtractMakeTargetRecipe(makefileText, target)
	if !found {
		return MakeTargetDecision{Allowed: false, Target: target, Reason: "unknown make target (fail-closed)"}
	}
	vars := parseSimpleMakeVars(makefileText)
	for _, raw := range recipe {
		line := normalizeMakeRecipeLine(raw)
		if line == "" {
			continue
		}
		expanded := expandSimpleMakeVarsOnce(vars, line)
		if dangerous, matched, _ := inspectMakeRecipeLine(expanded); dangerous {
			return MakeTargetDecision{
				Allowed:       false,
				Target:        target,
				MatchedRule:   matched,
				OffendingLine: line,
				Reason:        "git_indirect via=make target=" + target,
			}
		}
		if hasUnresolvedMakeVar(expanded) && hasDangerousContext(expanded) {
			return MakeTargetDecision{
				Allowed:       false,
				Target:        target,
				MatchedRule:   "deny_on_uncertain_make_var",
				OffendingLine: line,
				Reason:        "uncertain make variable with dangerous context (fail-closed) target=" + target,
			}
		}
	}
	return MakeTargetDecision{Allowed: true, Target: target, Reason: "no dangerous recipe command (allow candidate)"}
}
