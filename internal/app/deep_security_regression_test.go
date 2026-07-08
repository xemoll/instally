package app

import (
	"strings"
	"testing"
)

func TestHTTPDownloadsBlockedByDefault(t *testing.T) {
	_, err := PreviewURLCachePath("http://example.com/tool.AppImage")
	if err == nil || !strings.Contains(err.Error(), "plain HTTP") {
		t.Fatalf("expected plain HTTP block, got %v", err)
	}
}

func TestHTTPDownloadsCanBeExplicitlyAllowed(t *testing.T) {
	withEnv(t, "INSTALLY_ALLOW_INSECURE_HTTP", "1")
	_, err := PreviewURLCachePath("http://example.com/tool.AppImage")
	if err != nil {
		t.Fatalf("explicitly allowed HTTP should pass URL validation: %v", err)
	}
}

func TestInsecureGitBlockedByDefault(t *testing.T) {
	for _, raw := range []string{"http://github.com/cli/cli.git", "git://github.com/cli/cli.git"} {
		if err := validateGitTarget(raw); err == nil || !strings.Contains(err.Error(), "blocked by default") {
			t.Fatalf("expected insecure git block for %s, got %v", raw, err)
		}
	}
}

func TestOfficialScriptsUseTrustedModeNotAllowUnknown(t *testing.T) {
	cmd := officialScriptCmd("ollama", Options{Yes: true})
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "--trusted-official-script") {
		t.Fatalf("official scripts must use trusted mode: %v", cmd)
	}
	if strings.Contains(joined, "--allow-unknown") {
		t.Fatalf("official scripts should not use broad --allow-unknown: %v", cmd)
	}
}

func TestTrustedOfficialScriptAllowlist(t *testing.T) {
	ok := []string{"https://ollama.com/install.sh", "https://claude.ai/install.sh"}
	bad := []string{"http://ollama.com/install.sh", "https://evil.example/install.sh", "https://ollama.com/other.sh", "https://claude.ai/install.sh?x=1"}
	for _, u := range ok {
		if !isTrustedOfficialScriptURL(u) {
			t.Fatalf("expected trusted official URL: %s", u)
		}
	}
	for _, u := range bad {
		if isTrustedOfficialScriptURL(u) {
			t.Fatalf("unexpected trusted official URL: %s", u)
		}
	}
}

func TestChildSecurityEnvDoesNotCarryVTKey(t *testing.T) {
	env := envForChildSecurity(Options{VirusTotalKey: "secret", VirusTotalUpload: true, AllowUnknown: true})
	if _, ok := env["INSTALLY_VT_API_KEY"]; ok {
		t.Fatalf("child env leaked VT key: %#v", env)
	}
	if env["INSTALLY_VT_UPLOAD"] != "1" || env["INSTALLY_ALLOW_UNKNOWN"] != "1" {
		t.Fatalf("non-secret security env missing: %#v", env)
	}
}

func TestSanitizeNameNeverReturnsTraversalOrEmpty(t *testing.T) {
	for _, in := range []string{"", ".", "..", "../../evil", " a/b\\c:d ", "---"} {
		out := sanitizeName(in)
		if out == "" || out == "." || out == ".." || strings.Contains(out, "/") || strings.Contains(out, "\\") {
			t.Fatalf("unsafe sanitizeName(%q)=%q", in, out)
		}
	}
}

func TestOfficialScriptPlanDoesNotSetAllowUnknownEnv(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "apt")
	p := BuildPlan([]Task{{Kind: "app", Items: []string{"ollama"}}}, Options{Yes: true, DryRun: true})
	for _, c := range p.Commands {
		line := commandLine(c)
		if strings.Contains(line, "install-url-safe") && strings.Contains(line, "INSTALLY_ALLOW_UNKNOWN") {
			t.Fatalf("official script child command should not bypass trusted policy through allow-unknown env: %s", line)
		}
	}
}
