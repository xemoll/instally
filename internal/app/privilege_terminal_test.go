package app

import (
	"strings"
	"testing"
)

func TestNPMGlobalUsesUserPrefixOnLinux(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "apt")
	p := BuildPlan([]Task{{Kind: "npm", Items: []string{"pnpm"}}}, Options{Yes: true, DryRun: true})
	out := JSON(p)
	if !strings.Contains(out, "npm-global") || !strings.Contains(out, "--prefix") {
		t.Fatalf("npm global install should use user prefix, got: %s", out)
	}
	if strings.Contains(out, "sudo npm") {
		t.Fatalf("npm install should not require sudo by default: %s", out)
	}
}

func TestTerminalParseMultiFriendlyInput(t *testing.T) {
	tasks := ParseMultiItems("vscode, discord\ngithub:cli/cli\nhttps://example.com/app.AppImage")
	p := BuildPlan(tasks, Options{Yes: true, DryRun: true})
	out := JSON(p)
	for _, want := range []string{"vscode", "discord", "cli/cli", "app.AppImage"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan missing %q: %s", want, out)
		}
	}
}
