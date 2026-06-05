// Package guard: SB8 P5a M5 git/gh/auto/doctor command gate.
//
// Decision layer ONLY, structured-command based. It classifies a normalized
// command into a category (git/gh/auto/doctor/other) and denies the dangerous
// state-mutating subcommands via M2 EvaluateDenylist (deny-first). It reuses M1
// NormalizeCommand for input and M2 for matching. It does NOT execute git/gh/
// auto/doctor, does NOT integrate any runner hook (M9 owns that), and does NOT
// inspect raw shell strings / PowerShell / pipe-exec / install.ps1 structure
// (that is P5b M6). A final "allow" is decided by M3 profile, not here: a
// non-dangerous category command is reported as an allow candidate, and a
// non-category command is neutral (out of this gate's scope).
package guard

import "strings"

// GitGateDecision is the result of the git/gh/auto/doctor command gate.
type GitGateDecision struct {
	Allowed     bool   // false => dangerous command denied
	Category    string // git | gh | auto | doctor | other
	MatchedRule string // denied pattern when denied
	Reason      string
}

// categoryRules returns the per-category dangerous-command denylist. The single
// AllowedCommands entry is the baseline so a non-dangerous command of that
// category passes the M2 allow gate before the deny patterns are checked.
func categoryRules(category string) DenyRuleSet {
	switch category {
	case "git":
		return DenyRuleSet{
			AllowedCommands: []string{"git"},
			DeniedRegex: []string{
				`^git\s+(add|commit|push|merge|rebase|reset|clean)\b`,
				`^git\s+remote\s+set-url\b`,
				`^git\s+branch\s+-D\b`,
			},
		}
	case "gh":
		return DenyRuleSet{
			AllowedCommands: []string{"gh"},
			DeniedRegex:     []string{`^gh\s+pr\s+(create|merge)\b`},
		}
	case "auto":
		return DenyRuleSet{
			AllowedCommands: []string{"auto"},
			DeniedExact:     []string{"auto update", "auto install"},
		}
	case "doctor":
		return DenyRuleSet{
			AllowedCommands: []string{"doctor"},
			DeniedExact:     []string{"doctor --fix"},
		}
	default:
		return DenyRuleSet{}
	}
}

// ClassifyCommandCategory maps a normalized executable to its gate category.
func ClassifyCommandCategory(normExec string) string {
	switch normExec {
	case "git", "gh", "auto", "doctor":
		return normExec
	default:
		return "other"
	}
}

// dangerousIn evaluates compare against the category denylist (M2 deny-first).
// A baseline rejection (wrong category, no specific pattern matched) is NOT
// treated as dangerous — only a concrete denied-pattern match is.
func dangerousIn(category, compare string) (bool, string) {
	rs := categoryRules(category)
	if len(rs.AllowedCommands) == 0 {
		return false, ""
	}
	d := EvaluateDenylist(compare, rs)
	if !d.Allowed && d.MatchedPattern != "" {
		return true, d.MatchedPattern
	}
	return false, ""
}

// IsDangerousGitCommand reports whether a normalized "git ..." command is a
// state-mutating/dangerous git command. Precondition: the input is a git command.
func IsDangerousGitCommand(compare string) (bool, string) { return dangerousIn("git", compare) }

// IsDangerousGhCommand reports whether a normalized "gh ..." command is dangerous.
func IsDangerousGhCommand(compare string) (bool, string) { return dangerousIn("gh", compare) }

// IsDangerousAutoCommand reports whether a normalized "auto ..." command is dangerous.
func IsDangerousAutoCommand(compare string) (bool, string) { return dangerousIn("auto", compare) }

// IsDangerousDoctorCommand reports whether a normalized "doctor ..." command is dangerous.
func IsDangerousDoctorCommand(compare string) (bool, string) { return dangerousIn("doctor", compare) }

func isMutationGitVerb(verb string) bool {
	switch strings.ToLower(strings.TrimSpace(verb)) {
	case "push", "reset", "clean", "rebase", "merge", "add", "commit":
		return true
	default:
		return false
	}
}

func isReadOnlyGitAliasValue(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	switch strings.ToLower(fields[0]) {
	case "status", "log", "diff":
		return len(fields) == 1
	case "config":
		return len(fields) >= 2 && fields[1] == "--get"
	case "remote":
		return len(fields) == 2 && fields[1] == "-v"
	default:
		return false
	}
}

func aliasValueHasShellRisk(value string) bool {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(v, "!") {
		return true
	}
	for _, token := range []string{"&&", "||", ";", "|", "`", "$("} {
		if strings.Contains(v, token) {
			return true
		}
	}
	return false
}

func classifyGitAliasValue(value string) (dangerous bool, uncertain bool) {
	if aliasValueHasShellRisk(value) {
		return true, false
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return false, true
	}
	if isMutationGitVerb(fields[0]) {
		return true, false
	}
	if isReadOnlyGitAliasValue(fields) {
		return false, false
	}
	return false, true
}

func parseGitCommandLineAliases(args []string) (map[string]string, []string, bool) {
	aliases := map[string]string{}
	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		var config string
		switch {
		case arg == "-c":
			if i+1 >= len(args) {
				return aliases, remaining, true
			}
			i++
			config = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "-c"):
			config = strings.TrimSpace(strings.TrimPrefix(arg, "-c"))
		default:
			remaining = append(remaining, args[i:]...)
			return aliases, remaining, false
		}
		const prefix = "alias."
		if !strings.HasPrefix(config, prefix) {
			continue
		}
		keyValue := strings.TrimPrefix(config, prefix)
		name, value, ok := strings.Cut(keyValue, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !ok || name == "" || value == "" || strings.ContainsAny(name, " \t") {
			return aliases, remaining, true
		}
		aliases[name] = value
	}
	return aliases, remaining, false
}

// evaluateGitCommandLineAlias handles only command-line -c alias.<name>=<value>
// definitions from the same structured git invocation. It performs one-step
// expansion only; complex, shell, or ambiguous aliases fail closed before
// enforce. Full git config interpretation is intentionally not supported.
func evaluateGitCommandLineAlias(args []string) (bool, string, string) {
	aliases, remaining, uncertain := parseGitCommandLineAliases(args)
	if uncertain {
		return true, "git_alias_uncertain", "ambiguous git command-line alias denied (fail-closed)"
	}
	if len(aliases) == 0 {
		return false, "", ""
	}
	for _, value := range aliases {
		dangerous, valueUncertain := classifyGitAliasValue(value)
		if dangerous {
			return true, "git_alias_dangerous", "dangerous git command-line alias denied (fail-closed)"
		}
		if valueUncertain {
			return true, "git_alias_uncertain", "ambiguous git command-line alias denied (fail-closed)"
		}
	}
	if len(remaining) == 0 {
		return false, "", ""
	}
	invocation := strings.TrimSpace(remaining[0])
	value, ok := aliases[invocation]
	if !ok {
		return false, "", ""
	}
	expanded := append([]string{"git"}, strings.Fields(value)...)
	expanded = append(expanded, remaining[1:]...)
	compare := strings.TrimSpace(strings.Join(expanded, " "))
	if dangerous, pattern := dangerousIn("git", compare); dangerous {
		return true, pattern, "dangerous git command-line alias denied (deny-first)"
	}
	return false, "", ""
}

// EvaluateGitGate normalizes (M1) then gates git/gh/auto/doctor commands.
// Non-category commands are neutral (this gate neither allows nor denies them).
// Non-dangerous category commands are allow candidates (the profile, M3, decides
// the final allow). Dangerous commands are denied (deny-first via M2).
func EvaluateGitGate(executable string, args []string) GitGateDecision {
	nc := NormalizeCommand(executable, args)
	category := ClassifyCommandCategory(nc.NormalizedExecutable)
	if category == "other" {
		return GitGateDecision{Allowed: true, Category: "other", Reason: "out of git-gate scope (neutral)"}
	}

	if category == "git" {
		if dangerous, matched, reason := evaluateGitCommandLineAlias(nc.NormalizedArgsForCompare); dangerous {
			return GitGateDecision{
				Allowed:     false,
				Category:    category,
				MatchedRule: matched,
				Reason:      reason,
			}
		}
	}

	if dangerous, pattern := dangerousIn(category, nc.CompareString); dangerous {
		return GitGateDecision{
			Allowed:     false,
			Category:    category,
			MatchedRule: pattern,
			Reason:      "dangerous " + category + " command denied (deny-first)",
		}
	}

	return GitGateDecision{
		Allowed:  true,
		Category: category,
		Reason:   category + " command, no dangerous pattern (allow candidate; profile decides final allow)",
	}
}
