// Package guard: SB8 P7 M8 network egress gate.
//
// Decision layer ONLY, default-deny. Given an egress request (URL + purpose) and
// a rule set, it decides allow-candidate / deny WITHOUT making any network call.
// It uses net/url and net for PARSING ONLY — no net.LookupHost, no http.Client,
// no http.Get/NewRequest, no DNS resolution. Host is judged via url.Hostname()
// (never substring-matched on the raw URL). It does NOT integrate a runner hook
// (M9) and does NOT execute curl/wget/iwr/irm. A pass yields an allow CANDIDATE
// only — the product is api_use:forbidden / CLI-only, so a final allow still
// requires the M9 hook gate.
package guard

import (
	"net"
	"net/url"
	"strings"
)

// ReasonAPIForbidden is the SPEC T14 deny reason for AI model API egress
// (log contract: "DENY network egress reason=api_use_forbidden").
const ReasonAPIForbidden = "api_use_forbidden"

// EgressRequest is a single egress attempt.
type EgressRequest struct {
	URL     string
	Purpose string
}

// EgressRuleSet is the default-deny egress policy.
type EgressRuleSet struct {
	AllowedHosts     []string
	AllowedProtocols []string
	AllowedPurposes  []string
	BlockedDomains   []string
}

// EgressDecision is the result of evaluating an egress request.
type EgressDecision struct {
	Allowed  bool
	Host     string
	Protocol string
	Reason   string
}

// aiAPIHosts are known AI model API endpoints denied as api_use_forbidden.
var aiAPIHosts = []string{
	"api.openai.com", "api.anthropic.com", "generativelanguage.googleapis.com",
	"api.cohere.ai", "api.mistral.ai", "api.x.ai", "api.groq.com",
}

// NormalizeHost lowercases and trims a host (url.Hostname has no port).
func NormalizeHost(h string) string { return strings.ToLower(strings.TrimSpace(h)) }

// IsPrivateOrLocalAddress reports whether host is a literal local/private IP or
// localhost. It parses literal IPs only — it performs NO DNS resolution.
func IsPrivateOrLocalAddress(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// IsBlockedDomain reports whether host equals or is a subdomain of a blocked one.
func IsBlockedDomain(host string, blocked []string) bool {
	for _, b := range blocked {
		b = NormalizeHost(b)
		if b == "" {
			continue
		}
		if host == b || strings.HasSuffix(host, "."+b) {
			return true
		}
	}
	return false
}

// IsAPIEndpoint reports whether host is a known AI model API endpoint.
func IsAPIEndpoint(host string) bool {
	for _, a := range aiAPIHosts {
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

func inList(v string, list []string) bool {
	for _, s := range list {
		if NormalizeHost(s) == v {
			return true
		}
	}
	return false
}

func denyEgress(host, protocol, reason string) EgressDecision {
	return EgressDecision{Allowed: false, Host: host, Protocol: protocol, Reason: reason}
}

// EvaluateEgress applies the default-deny egress policy. Empty/malformed URL,
// unknown host, disallowed protocol/purpose, private/local address, blocked
// domain, and AI API endpoints all deny. Only an explicitly allowed host +
// protocol + purpose yields an allow candidate.
func EvaluateEgress(req EgressRequest, rs EgressRuleSet) EgressDecision {
	if strings.TrimSpace(req.URL) == "" {
		return denyEgress("", "", "empty URL (fail-closed)")
	}
	u, err := url.Parse(req.URL)
	if err != nil {
		return denyEgress("", "", "malformed URL (fail-closed)")
	}
	protocol := strings.ToLower(u.Scheme)
	if protocol == "" {
		return denyEgress("", "", "malformed URL: no scheme (fail-closed)")
	}
	if !inList(protocol, rs.AllowedProtocols) {
		return denyEgress("", protocol, "protocol not allowed")
	}

	host := NormalizeHost(u.Hostname())
	if host == "" {
		return denyEgress("", protocol, "empty/unknown host (fail-closed)")
	}
	if IsPrivateOrLocalAddress(host) {
		return denyEgress(host, protocol, "private/local address denied")
	}
	if IsBlockedDomain(host, rs.BlockedDomains) {
		return denyEgress(host, protocol, "blocked domain denied")
	}
	if IsAPIEndpoint(host) {
		return denyEgress(host, protocol, ReasonAPIForbidden)
	}
	if !inList(host, rs.AllowedHosts) {
		return denyEgress(host, protocol, "unknown host denied (fail-closed)")
	}
	if !inList(req.Purpose, rs.AllowedPurposes) {
		return denyEgress(host, protocol, "purpose not allowed")
	}

	return EgressDecision{
		Allowed:  true,
		Host:     host,
		Protocol: protocol,
		Reason:   "egress allow candidate (final allow requires M9 hook gate)",
	}
}
