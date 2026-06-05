package telemetry

import (
	"regexp"
	"strings"
)

// previewMaxRunes is the SB8 v1 hard limit for command_preview (rune count).
const previewMaxRunes = 120

// Token patterns. ReplaceAllString swallows panics inherently; the whole flow
// is also wrapped in a deferred recover in RedactPreview.
var (
	reAKIA  = regexp.MustCompile(`A(KIA|SIA)[0-9A-Z]{16}`)
	reSKKey = regexp.MustCompile(`sk-[A-Za-z0-9_\-]{16,}`)
	reGHTok = regexp.MustCompile(`gh[posu]_[A-Za-z0-9]{20,}`)
	reGLPat = regexp.MustCompile(`glpat-[A-Za-z0-9_\-]{16,}`)
	reJWT   = regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`)
	reBear  = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-]{16,}`)
	reAuth  = regexp.MustCompile(`(?i)Authorization:[^'"\n]+`)

	reHomeWin    = regexp.MustCompile(`(?i)([A-Z]:\\Users\\)([^\\/\s]+)`)
	reHomeLinux  = regexp.MustCompile(`(/home/)([^/\s]+)`)
	reHomeDarwin = regexp.MustCompile(`(/Users/)([^/\s]+)`)
)

// envWhitelist* entries trigger length-independent redaction. The generic
// KEY=VALUE rule (looksLikeEnvKey + value len >= 6) catches the rest.
var (
	envWhitelistExact  = []string{"PATH"}
	envWhitelistPrefix = []string{"AWS_"}
	envWhitelistSuffix = []string{"_TOKEN", "_SECRET", "_KEY", "_PASSWORD"}
)

// redactionFailureProbeForTest is a TEST-ONLY toggle that forces RedactPreview
// into its recover branch so the emit-skip path can be exercised.
var redactionFailureProbeForTest bool

// SetRedactionFailureProbe is a TEST-ONLY setter exposed for cross-package
// tests (worker/orchestra hook tests) that need to exercise the redaction
// emit-skip path. Production code never calls this.
func SetRedactionFailureProbe(v bool) { redactionFailureProbeForTest = v }

// RedactionFailureProbe is a TEST-ONLY getter for the redaction-failure toggle.
func RedactionFailureProbe() bool { return redactionFailureProbeForTest }

// RedactPreview applies token / home-path / env-value redaction in sequence and
// truncates to previewMaxRunes runes. Returns (redacted, flag, ok). On any
// recovered panic, ok=false and the caller MUST skip emit.
func RedactPreview(raw string) (out string, flag bool, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			out = ""
			flag = false
			ok = false
		}
	}()
	if redactionFailureProbeForTest {
		panic("test-induced redaction failure")
	}
	text := raw
	before := text
	text = redactTokens(text)
	text = redactHomePath(text)
	text = redactEnvValues(text)
	flag = text != before
	text = truncateRunes(text, previewMaxRunes)
	return text, flag, true
}

func redactTokens(s string) string {
	s = reAKIA.ReplaceAllString(s, "[REDACTED]")
	s = reSKKey.ReplaceAllString(s, "[REDACTED]")
	s = reGHTok.ReplaceAllString(s, "[REDACTED]")
	s = reGLPat.ReplaceAllString(s, "[REDACTED]")
	s = reJWT.ReplaceAllString(s, "[REDACTED]")
	s = reBear.ReplaceAllString(s, "[REDACTED]")
	s = reAuth.ReplaceAllString(s, "[REDACTED]")
	return s
}

func redactHomePath(s string) string {
	s = reHomeWin.ReplaceAllString(s, "${1}[USER]")
	s = reHomeLinux.ReplaceAllString(s, "${1}[USER]")
	s = reHomeDarwin.ReplaceAllString(s, "${1}[USER]")
	return s
}

// redactEnvValues finds KEY=VALUE space-delimited tokens. Whitelisted keys are
// redacted regardless of value length; env-shaped keys (uppercase identifier)
// are redacted when VALUE length >= 6. Other tokens (e.g. "--option=value")
// pass through to avoid lossy CLI-flag scrubbing.
func redactEnvValues(s string) string {
	parts := strings.Split(s, " ")
	for i, p := range parts {
		eq := strings.IndexByte(p, '=')
		if eq <= 0 || eq == len(p)-1 {
			continue
		}
		key := p[:eq]
		val := p[eq+1:]
		if shouldRedactEnv(key) {
			parts[i] = key + "=[ENV]"
			continue
		}
		if !looksLikeEnvKey(key) {
			continue
		}
		if len(val) < 6 {
			continue
		}
		parts[i] = key + "=[ENV]"
	}
	return strings.Join(parts, " ")
}

func looksLikeEnvKey(k string) bool {
	if k == "" {
		return false
	}
	first := k[0]
	if !((first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	for _, r := range k {
		if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func shouldRedactEnv(k string) bool {
	upper := strings.ToUpper(k)
	for _, w := range envWhitelistExact {
		if upper == w {
			return true
		}
	}
	for _, w := range envWhitelistPrefix {
		if strings.HasPrefix(upper, w) {
			return true
		}
	}
	for _, w := range envWhitelistSuffix {
		if strings.HasSuffix(upper, w) {
			return true
		}
	}
	return false
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}
