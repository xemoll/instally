package app

import (
	"strings"
	"testing"
)

func TestNPMGlobalUsesUserPrefixOnLinux(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "apt")
	p := BuildPlan([]Task{{Kind: "npm", Items: []string{"opencode-ai"}}}, Options{Yes: true, DryRun: true})
	out := JSON(p)
	if !strings.Contains(out, "npm-global") || !strings.Contains(out, "--prefix") {
		t.Fatalf("npm global install should use user prefix, got: %s", out)
	}
	if strings.Contains(out, "sudo npm") {
		t.Fatalf("npm install should not require sudo by default: %s", out)
	}
}

func TestAIToolsAvoidCurlPipeShell(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "pacman")
	p := BuildPlan([]Task{{Kind: "ai-tools", Items: []string{"ai-tools"}}}, Options{Yes: true, DryRun: true})
	out := JSON(p)
	if strings.Contains(out, "curl |") || strings.Contains(out, "| sh") || strings.Contains(out, "| bash") {
		t.Fatalf("AI tools plan must not use curl|sh style execution: %s", out)
	}
	if !strings.Contains(out, "--install-url-safe") {
		t.Fatalf("AI tools plan should use safe URL installer for official scripts: %s", out)
	}
}

func TestClaudeAptAdminShellNoInlineSudo(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "apt")
	p := BuildPlan([]Task{{Kind: "official-claude-code", Items: []string{"claude-code"}}}, Options{Yes: true, DryRun: true})
	found := false
	for _, c := range p.Commands {
		if strings.Contains(c.Title, "Claude Code apt") {
			found = true
			if !c.Admin {
				t.Fatalf("Claude apt repository command must be marked admin")
			}
			if strings.Contains(c.Shell, "sudo ") {
				t.Fatalf("admin shell should be elevated by runner, not embed sudo: %s", c.Shell)
			}
		}
	}
	if !found {
		t.Fatalf("Claude apt command not found: %s", JSON(p))
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
