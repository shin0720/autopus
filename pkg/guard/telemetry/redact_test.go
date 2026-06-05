package telemetry

import (
	"strings"
	"testing"
)

func TestRedact_AKIA(t *testing.T) {
	in := "aws s3 cp x s3://b AKIAABCDEFGHIJKLMNOP --profile prod"
	out, flag, ok := RedactPreview(in)
	if !ok {
		t.Fatal("redact ok=false")
	}
	if !flag {
		t.Error("flag must be true when content changed")
	}
	if strings.Contains(out, "AKIAABCDEFGHIJKLMNOP") {
		t.Errorf("AWS key leaked: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected [REDACTED] marker: %q", out)
	}
}

func TestRedact_OpenAIKey(t *testing.T) {
	in := "curl -H 'x' sk-abcdef0123456789ABCDEFXYZ"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "sk-abcdef0123456789ABCDEFXYZ") {
		t.Errorf("sk- key leaked: %q", out)
	}
}

func TestRedact_GitHubToken(t *testing.T) {
	tokens := []string{
		"ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII",
		"gho_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII",
		"ghs_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII",
		"ghu_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII",
	}
	for _, tok := range tokens {
		in := "git -c http.extraheader='Authorization: token " + tok + "' fetch"
		out, _, ok := RedactPreview(in)
		if !ok {
			t.Fatalf("redact failed for %s", tok)
		}
		if strings.Contains(out, tok) {
			t.Errorf("token %s leaked: %q", tok, out)
		}
	}
}

func TestRedact_GitLabPAT(t *testing.T) {
	in := "git push https://oauth2:glpat-ABCDEFGHIJ1234567890@gitlab.example.com/x.git"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "glpat-ABCDEFGHIJ1234567890") {
		t.Errorf("glpat- token leaked: %q", out)
	}
}

func TestRedact_JWT(t *testing.T) {
	jwt := "eyJabcdefghijklmnopqrstuv.eyJabcdefghij.signature01234567"
	in := "curl -H 'Authorization: Bearer " + jwt + "'"
	out, _, ok := RedactPreview(in)
	if !ok {
		t.Fatal("ok=false")
	}
	if strings.Contains(out, jwt) {
		t.Errorf("JWT leaked: %q", out)
	}
}

func TestRedact_Bearer(t *testing.T) {
	in := "curl -H 'Bearer abcdef0123456789xyz'"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "abcdef0123456789xyz") {
		t.Errorf("Bearer token leaked: %q", out)
	}
}

func TestRedact_AuthorizationHeader(t *testing.T) {
	in := "curl -H 'Authorization: Basic dXNlcjpwYXNz=='"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "dXNlcjpwYXNz") {
		t.Errorf("Authorization header value leaked: %q", out)
	}
}

func TestRedact_HomePathWindows(t *testing.T) {
	in := `go test C:\Users\shin0720\workspace\autopus`
	out, _, ok := RedactPreview(in)
	if !ok {
		t.Fatal("ok=false")
	}
	if strings.Contains(out, "shin0720") {
		t.Errorf("Windows home leaked: %q", out)
	}
	if !strings.Contains(out, `C:\Users\[USER]\`) {
		t.Errorf("[USER] marker missing: %q", out)
	}
}

func TestRedact_HomePathLinux(t *testing.T) {
	in := "ls /home/shin/.config"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "/home/shin/") {
		t.Errorf("Linux home leaked: %q", out)
	}
	if !strings.Contains(out, "/home/[USER]/") {
		t.Errorf("[USER] marker missing: %q", out)
	}
}

func TestRedact_HomePathDarwin(t *testing.T) {
	in := "ls /Users/alice/.ssh"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "/Users/alice/") {
		t.Errorf("Darwin home leaked: %q", out)
	}
}

func TestRedact_EnvValuePATH(t *testing.T) {
	in := "env PATH=/usr/local/bin:/usr/bin make ci"
	out, _, ok := RedactPreview(in)
	if !ok {
		t.Fatal("ok=false")
	}
	if strings.Contains(out, "/usr/local/bin") {
		t.Errorf("PATH value leaked: %q", out)
	}
	if !strings.Contains(out, "PATH=[ENV]") {
		t.Errorf("[ENV] marker missing for PATH: %q", out)
	}
}

func TestRedact_EnvValueAWS(t *testing.T) {
	in := "AWS_ACCESS_KEY_ID=AKIAEXAMPLE12345678 aws s3 ls"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "AKIAEXAMPLE12345678") {
		t.Errorf("AWS_* env value leaked: %q", out)
	}
}

func TestRedact_EnvValueTokenSuffix(t *testing.T) {
	in := "MY_TOKEN=verysecretvalue123 deploy"
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "verysecretvalue123") {
		t.Errorf("_TOKEN suffix env value leaked: %q", out)
	}
}

func TestRedact_GenericEnvShapedKey(t *testing.T) {
	// uppercase identifier, value >= 6 chars -> redacted
	in := "API_HOST=production-api-host go run ."
	out, _, ok := RedactPreview(in)
	if !ok || strings.Contains(out, "production-api-host") {
		t.Errorf("generic env key/value leaked: %q", out)
	}
}

func TestRedact_CliFlagWithEqualsNotEnvShape(t *testing.T) {
	// "--option=somevalue" has '-' in key -> not env-shaped -> NOT redacted
	in := "go test --option=somevalue ./..."
	out, _, ok := RedactPreview(in)
	if !ok {
		t.Fatal("ok=false")
	}
	if !strings.Contains(out, "--option=somevalue") {
		t.Errorf("CLI flag must not be redacted: %q", out)
	}
}

func TestRedact_TruncationAt120Runes(t *testing.T) {
	long := strings.Repeat("abcdefghij", 20) // 200 ASCII runes
	out, _, ok := RedactPreview(long)
	if !ok {
		t.Fatal("ok=false")
	}
	if got := len([]rune(out)); got > 120 {
		t.Errorf("preview exceeded 120 runes: %d", got)
	}
	if !strings.HasSuffix(out, "…") {
		t.Errorf("truncated output must end with …: %q", out)
	}
}

func TestRedact_TruncationPreservesMultibyteRunes(t *testing.T) {
	long := strings.Repeat("가", 200) // 200 Korean runes
	out, _, ok := RedactPreview(long)
	if !ok {
		t.Fatal("ok=false")
	}
	if got := len([]rune(out)); got > 120 {
		t.Errorf("multibyte preview exceeded 120 runes: %d", got)
	}
}

func TestRedact_PanicReturnsNotOk(t *testing.T) {
	redactionFailureProbeForTest = true
	defer func() { redactionFailureProbeForTest = false }()
	_, _, ok := RedactPreview("anything")
	if ok {
		t.Error("RedactPreview must return ok=false on induced panic")
	}
}

func TestRedact_NoChangeNoFlag(t *testing.T) {
	in := "git status -sb"
	out, flag, ok := RedactPreview(in)
	if !ok {
		t.Fatal("ok=false")
	}
	if out != in {
		t.Errorf("clean input must pass through unchanged, got %q", out)
	}
	if flag {
		t.Errorf("flag must be false when nothing was redacted")
	}
}

func TestRedact_OriginalSecretSubstringsAbsent(t *testing.T) {
	secrets := []string{
		"AKIAABCDEFGHIJKLMNOP",
		"sk-abcdef0123456789ABCDEFXYZ",
		"ghp_AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIII",
		"glpat-ABCDEFGHIJ1234567890",
	}
	combined := strings.Join(secrets, " ")
	out, _, ok := RedactPreview(combined)
	if !ok {
		t.Fatal("ok=false")
	}
	for _, s := range secrets {
		if strings.Contains(out, s) {
			t.Errorf("secret %q leaked: %q", s, out)
		}
	}
}
