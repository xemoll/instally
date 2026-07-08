package app

import (
	"strings"
	"testing"
)

func TestCompatibilityMatrix30Systems(t *testing.T) {
	profiles := CompatibilityProfiles30()
	if len(profiles) != 30 {
		t.Fatalf("expected 30 profiles, got %d", len(profiles))
	}
	apps := []string{"vscode", "firefox", "discord", "telegram", "git", "curl", "node", "go", "rust", "python", "java", "docker", "ollama", "opencode", "claude-code"}
	for _, prof := range profiles {
		t.Run(prof.Name, func(t *testing.T) {
			withCompatEnv(prof, func() {
				plan := BuildPlan(ParseMultiItems(strings.Join(apps, ",")), Options{Yes: true, DryRun: true, AllowUnknown: true, ContinueOnError: true})
				if len(plan.Commands) == 0 {
					t.Fatalf("no install commands for %s warnings=%v", prof.Name, plan.Warnings)
				}
				var all string
				for _, c := range plan.Commands {
					all += commandLine(c) + "\n"
				}
				for _, forbidden := range []string{"\x00", " --bad", "curl | sh"} {
					if strings.Contains(all, forbidden) {
						t.Fatalf("unsafe command fragment %q in plan:\n%s", forbidden, all)
					}
				}
			})
		})
	}
}

func TestNativePackageNormalizationByManager(t *testing.T) {
	cases := []struct {
		pm   string
		in   []string
		want []string
	}{
		{"apt", []string{"go", "rust", "python", "python-pip", "jdk-openjdk", "docker"}, []string{"golang-go", "rustc", "cargo", "python3", "python3-pip", "default-jdk", "docker.io"}},
		{"dnf", []string{"go", "rust", "python", "jdk-openjdk", "docker"}, []string{"golang", "rust", "cargo", "python3", "java-21-openjdk-devel", "moby-engine"}},
		{"apk", []string{"python", "python-pip", "jdk-openjdk", "rust"}, []string{"python3", "py3-pip", "openjdk21", "rust", "cargo"}},
		{"pacman", []string{"go", "rust", "python", "python-pip", "jdk-openjdk"}, []string{"go", "rust", "python", "python-pip", "jdk-openjdk"}},
	}
	for _, tc := range cases {
		got := normalizeNativePackagesForManager(tc.pm, tc.in)
		for _, w := range tc.want {
			found := false
			for _, g := range got {
				if g == w {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s normalization missing %s in %#v", tc.pm, w, got)
			}
		}
	}
}

func TestCompatibilityMatrixReportSummary(t *testing.T) {
	report := CompatibilityMatrixReport()
	if !strings.Contains(report, "summary: passed=30 failed=0 total=30") {
		t.Fatalf("unexpected matrix summary:\n%s", report)
	}
}

func TestMultiInstallRejectsSemicolonInsteadOfSplitting(t *testing.T) {
	tasks := ParseMultiItems("vscode,bad;name")
	plan := BuildPlan(tasks, Options{Yes: true, DryRun: true})
	var all string
	for _, c := range plan.Commands {
		all += commandLine(c) + "\n"
	}
	if strings.Contains(all, "bad name") || strings.Contains(all, "bad;name") {
		t.Fatalf("semicolon input reached command plan:\n%s\nwarnings=%v", all, plan.Warnings)
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected warning for semicolon input")
	}
}
