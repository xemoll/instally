package app

import (
	"strings"
	"testing"
)

func TestDiscordOfficialURLKeepsDebExtension(t *testing.T) {
	t.Setenv("INSTALLY_SKIP_DNS_PRIVATE_CHECK", "1")
	p, err := PreviewURLCachePath("https://discord.com/api/download?platform=linux&format=deb")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "discord.deb") {
		t.Fatalf("expected discord.deb cache filename, got %s", p)
	}
}

func TestFirefoxDiscordPlanDebianUsesOfficialSources(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "apt")
	t.Setenv("INSTALLY_FORCE_OS_ID", "debian")
	t.Setenv("INSTALLY_SKIP_DNS_PRIVATE_CHECK", "1")
	p := BuildPlan(ParseMultiItems("firefox,discord"), Options{Yes: true, DryRun: true, AllowUnknown: true})
	var all string
	for _, c := range p.Commands {
		all += commandLine(c) + "\n"
	}
	for _, want := range []string{"packages.mozilla.org/apt", "35BAA0B33E9EB396F59CA838C0BA5CE6DC6315A3", "discord.com/api/download?platform=linux&format=deb"} {
		if !strings.Contains(all, want) {
			t.Fatalf("plan missing %s:\n%s", want, all)
		}
	}
}
