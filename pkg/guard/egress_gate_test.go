package guard

import "testing"

func baseEgressRule() EgressRuleSet {
	return EgressRuleSet{
		AllowedHosts:     []string{"pkg.go.dev", "proxy.golang.org"},
		AllowedProtocols: []string{"https"},
		AllowedPurposes:  []string{"documentation_lookup", "package_metadata_check"},
		BlockedDomains:   []string{"evil.com"},
	}
}

func denyEgressCase(t *testing.T, rawURL, purpose string) EgressDecision {
	t.Helper()
	d := EvaluateEgress(EgressRequest{URL: rawURL, Purpose: purpose}, baseEgressRule())
	if d.Allowed {
		t.Errorf("EvaluateEgress(%q,%q) should deny, got %+v", rawURL, purpose, d)
	}
	return d
}

func TestEgress_HttpsAllowedHostAllow(t *testing.T) {
	d := EvaluateEgress(EgressRequest{URL: "https://pkg.go.dev/net/url", Purpose: "documentation_lookup"}, baseEgressRule())
	if !d.Allowed || d.Host != "pkg.go.dev" || d.Protocol != "https" {
		t.Errorf("https allowed host should be allow candidate, got %+v", d)
	}
}

func TestEgress_AllowedHostWrongPurposeDeny(t *testing.T) {
	denyEgressCase(t, "https://pkg.go.dev/x", "telemetry_upload")
}
func TestEgress_HttpDeny(t *testing.T)        { denyEgressCase(t, "http://pkg.go.dev", "documentation_lookup") }
func TestEgress_UnknownHostDeny(t *testing.T) { denyEgressCase(t, "https://unknown.example.com", "documentation_lookup") }
func TestEgress_EmptyURLDeny(t *testing.T)    { denyEgressCase(t, "", "documentation_lookup") }
func TestEgress_MalformedURLDeny(t *testing.T) {
	denyEgressCase(t, "https://%zz.com", "documentation_lookup")
}
func TestEgress_NoSchemeDeny(t *testing.T)    { denyEgressCase(t, "notaurl", "documentation_lookup") }
func TestEgress_LocalhostDeny(t *testing.T)   { denyEgressCase(t, "https://localhost/x", "documentation_lookup") }
func TestEgress_LoopbackDeny(t *testing.T)    { denyEgressCase(t, "https://127.0.0.1/x", "documentation_lookup") }
func TestEgress_UnspecifiedDeny(t *testing.T) { denyEgressCase(t, "https://0.0.0.0/x", "documentation_lookup") }
func TestEgress_Private10Deny(t *testing.T)   { denyEgressCase(t, "https://10.0.0.5/x", "documentation_lookup") }
func TestEgress_Private192Deny(t *testing.T)  { denyEgressCase(t, "https://192.168.1.1/x", "documentation_lookup") }
func TestEgress_Private172Deny(t *testing.T)  { denyEgressCase(t, "https://172.16.0.1/x", "documentation_lookup") }
func TestEgress_LinkLocalDeny(t *testing.T)   { denyEgressCase(t, "https://169.254.1.1/x", "documentation_lookup") }
func TestEgress_FileSchemeDeny(t *testing.T)  { denyEgressCase(t, "file:///etc/passwd", "documentation_lookup") }
func TestEgress_FtpSchemeDeny(t *testing.T)   { denyEgressCase(t, "ftp://pkg.go.dev/x", "documentation_lookup") }

func TestEgress_APIEndpointDenyReason(t *testing.T) {
	d := denyEgressCase(t, "https://api.openai.com/v1/chat/completions", "documentation_lookup")
	if d.Reason != ReasonAPIForbidden {
		t.Errorf("AI API egress must deny reason=api_use_forbidden, got %+v", d)
	}
}

func TestEgress_BlockedDomainDeny(t *testing.T) {
	denyEgressCase(t, "https://evil.com/x", "documentation_lookup")
	denyEgressCase(t, "https://sub.evil.com/x", "documentation_lookup") // subdomain
}

func TestEgress_UppercaseHostNormalize(t *testing.T) {
	d := EvaluateEgress(EgressRequest{URL: "https://PKG.GO.DEV/net/url", Purpose: "documentation_lookup"}, baseEgressRule())
	if !d.Allowed || d.Host != "pkg.go.dev" {
		t.Errorf("uppercase host should normalize and allow, got %+v", d)
	}
}

func TestEgress_PathCannotBypassHostAllow(t *testing.T) {
	// Host is judged by Hostname(), so an allowed host in the path/query never
	// bypasses the gate.
	d := denyEgressCase(t, "https://attacker.com/pkg.go.dev?x=proxy.golang.org", "documentation_lookup")
	if d.Host == "pkg.go.dev" || d.Host == "proxy.golang.org" {
		t.Errorf("host must be attacker.com (from authority), got %+v", d)
	}
}

func TestEgress_Helpers(t *testing.T) {
	if NormalizeHost("  PKG.GO.Dev ") != "pkg.go.dev" {
		t.Error("NormalizeHost should lowercase/trim")
	}
	if !IsPrivateOrLocalAddress("127.0.0.1") || !IsPrivateOrLocalAddress("localhost") || IsPrivateOrLocalAddress("8.8.8.8") {
		t.Error("IsPrivateOrLocalAddress mismatch")
	}
	if !IsBlockedDomain("sub.evil.com", []string{"evil.com"}) || IsBlockedDomain("good.com", []string{"evil.com"}) {
		t.Error("IsBlockedDomain mismatch")
	}
	if !IsAPIEndpoint("api.openai.com") || IsAPIEndpoint("pkg.go.dev") {
		t.Error("IsAPIEndpoint mismatch")
	}
}
