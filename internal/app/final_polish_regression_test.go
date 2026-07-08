package app

import (
	"os"
	"strings"
	"testing"
)

func TestFlatpakDefaultsToUserScope(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "apt")
	withEnv(t, "INSTALLY_FLATPAK_SYSTEM", "")
	p := BuildPlan([]Task{{Kind: "flatpak", Items: []string{"org.mozilla.firefox"}}}, Options{Yes: true, DryRun: true})
	out := JSON(p)
	if !strings.Contains(out, "--user") {
		t.Fatalf("flatpak should default to user scope to avoid unnecessary admin prompts: %s", out)
	}
	if strings.Contains(out, "--system") {
		t.Fatalf("flatpak should not use system scope by default: %s", out)
	}
}

func TestFlatpakSystemScopeOptIn(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "apt")
	withEnv(t, "INSTALLY_FLATPAK_SYSTEM", "1")
	p := BuildPlan([]Task{{Kind: "flatpak", Items: []string{"org.mozilla.firefox"}}}, Options{Yes: true, DryRun: true})
	out := JSON(p)
	if !strings.Contains(out, "--system") {
		t.Fatalf("flatpak system scope opt-in should be represented: %s", out)
	}
}

func TestVirusTotalKeyFileLoadsAndIsCleaned(t *testing.T) {
	dir := t.TempDir()
	withEnv(t, "INSTALLY_DATA_DIR", dir)
	keyFile, err := writeEphemeralKeyFile("secret-key-123")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("INSTALLY_VT_API_KEY", "")
	t.Setenv("INSTALLY_VT_KEY_FILE", keyFile)
	if got := SecurityOptionsFromEnv().VirusTotalKey; got != "secret-key-123" {
		t.Fatalf("key file was not loaded, got %q", got)
	}
	cleanupSecretFiles(map[string]string{"INSTALLY_VT_KEY_FILE": keyFile})
	if _, err := os.Stat(keyFile); !os.IsNotExist(err) {
		t.Fatalf("ephemeral key file should be removed, stat err=%v", err)
	}
}

func TestChildSecurityEnvUsesKeyFileNotRawKey(t *testing.T) {
	dir := t.TempDir()
	withEnv(t, "INSTALLY_DATA_DIR", dir)
	env := envForChildSecurity(Options{VirusTotalKey: "very-secret-key", VirusTotalUpload: true})
	line := commandLine(CommandSpec{Cmd: []string{"instally", "--install-url-safe", "https://example.com/app.AppImage"}, Env: env})
	defer cleanupSecretFiles(env)
	if strings.Contains(line, "very-secret-key") || strings.Contains(line, "INSTALLY_VT_API_KEY") {
		t.Fatalf("command line leaked raw VirusTotal key: %s", line)
	}
	if !strings.Contains(line, "INSTALLY_VT_KEY_FILE=***") {
		t.Fatalf("expected masked key-file env, got: %s", line)
	}
}
