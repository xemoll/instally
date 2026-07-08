package app

import (
	"os"
	"strings"
	"testing"
)

func withEnv(t *testing.T, key, val string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Setenv(key, val); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestRejectOptionLikePackageNames(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "pacman")
	p := BuildPlan([]Task{{Kind: "pkg", Items: []string{"git", "--noconfirm", "bad;name", "curl"}}}, Options{Yes: true, DryRun: true})
	cmds := JSON(p.Commands)
	if strings.Contains(cmds, "bad;name") {
		t.Fatalf("unsafe package-like item leaked into commands:\n%s", cmds)
	}
	if countInCommands(p.Commands, "--noconfirm") != 1 {
		t.Fatalf("expected only pacman yes flag --noconfirm, got commands:\n%s", cmds)
	}
	out := JSON(p)
	if !strings.Contains(out, "git") || !strings.Contains(out, "curl") {
		t.Fatalf("valid package names missing from plan:\n%s", out)
	}
	if len(p.Warnings) < 2 {
		t.Fatalf("expected warnings for rejected items, got %#v", p.Warnings)
	}
}

func TestRejectBadURLAtPlanTime(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "url", Items: []string{"file:///etc/passwd", "http://127.0.0.1/app.AppImage"}}}, Options{DryRun: true})
	if len(p.Commands) != 0 {
		t.Fatalf("bad URLs should not create commands: %#v", p.Commands)
	}
	if len(p.Warnings) < 2 {
		t.Fatalf("expected warnings for rejected URLs: %#v", p.Warnings)
	}
}

func TestGitTargetValidation(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "git", Items: []string{"https://github.com/cli/cli", "file:///tmp/repo", "https://github.com/owner/repo;rm -rf /"}}}, Options{DryRun: true})
	cmds := JSON(p.Commands)
	if strings.Contains(cmds, "file:///tmp/repo") || strings.Contains(cmds, "rm -rf") {
		t.Fatalf("unsafe git target leaked into commands:\n%s", cmds)
	}
	out := JSON(p)
	if !strings.Contains(cmds, "github.com/cli/cli") {
		t.Fatalf("valid git target missing from plan:\n%s", out)
	}
}

func TestGitHubDryRunShowsActualLocalInstallPlan(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "pacman")
	// We cannot hit GitHub in unit tests. Exercise the dry-run branch by using
	// the lower level local plan shape expected after a selected asset.
	p := BuildPlan([]Task{{Kind: "local", Items: []string{"/tmp/app.AppImage"}}}, Options{Yes: true, DryRun: true, NoSecurity: true})
	out := JSON(p)
	if strings.Contains(out, "--install-local-safe") {
		t.Fatalf("NoSecurity local plan should show the real installer, not a second scanner wrapper:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "appimage") {
		t.Fatalf("expected AppImage install plan, got:\n%s", out)
	}
}

func TestLargeUniversalAppSetPlansOnMajorPlatforms(t *testing.T) {
	apps := "vscode, discord, telegram, firefox, brave, obs, vlc, blender, gimp, krita, steam, docker, node, go, rust, ollama, opencode, claude-code, git, curl, fastfetch, btop, qbittorrent"
	matrix := []struct{ os, pm string }{{"linux", "pacman"}, {"linux", "apt"}, {"windows", "winget"}, {"darwin", "brew"}}
	for _, m := range matrix {
		t.Run(m.os+"-"+m.pm, func(t *testing.T) {
			withEnv(t, "INSTALLY_FORCE_OS", m.os)
			withEnv(t, "INSTALLY_FORCE_PM", m.pm)
			p := BuildPlan(ParseMultiItems(apps), Options{Yes: true, DryRun: true, AllowUnknown: true})
			if len(p.Commands) < 8 {
				t.Fatalf("expected a substantial plan for %s/%s, got %d commands warnings=%v", m.os, m.pm, len(p.Commands), p.Warnings)
			}
			out := JSON(p)
			if strings.Contains(out, "\n") && strings.Contains(out, "INSTALLY_VT_API_KEY") {
				t.Fatalf("VT key/env should not be present without explicit key:\n%s", out)
			}
		})
	}
}

func countInCommands(commands []CommandSpec, needle string) int {
	n := 0
	for _, c := range commands {
		for _, part := range c.Cmd {
			if part == needle {
				n++
			}
		}
	}
	return n
}

func TestRunPlanWarningsOnlyFails(t *testing.T) {
	p := BuildPlan(ParseMultiItems("bad;name"), Options{Yes: true, DryRun: true})
	res := RunPlan(p, true)
	if res.OK {
		t.Fatalf("warnings-only plan should fail, output=%q warnings=%v", res.Output, p.Warnings)
	}
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code for warnings-only plan")
	}
}

func TestPackageManagerRefreshAttachedBroadly(t *testing.T) {
	cases := []struct {
		os   string
		pm   string
		want string
	}{
		{"linux", "apt", "apt-get update"},
		{"linux", "pacman", "pacman -Sy"},
		{"linux", "dnf", "dnf makecache --refresh"},
		{"linux", "zypper", "zypper refresh"},
		{"linux", "apk", "apk update"},
		{"linux", "xbps", "xbps-install -S"},
		{"linux", "eopkg", "eopkg ur"},
		{"linux", "emerge", "emerge --sync"},
		{"linux", "packagekit", "pkcon refresh"},
		{"darwin", "brew", "brew update"},
		{"darwin", "port", "port selfupdate"},
		{"windows", "winget", "winget source update"},
		{"windows", "scoop", "scoop update"},
	}
	for _, tc := range cases {
		t.Run(tc.os+"-"+tc.pm, func(t *testing.T) {
			withEnv(t, "INSTALLY_FORCE_OS", tc.os)
			withEnv(t, "INSTALLY_FORCE_PM", tc.pm)
			p := BuildPlan([]Task{{Kind: "pkg", Items: []string{"git"}}}, Options{Yes: true, DryRun: true})
			if len(p.Commands) == 0 {
				t.Fatalf("expected command, got warnings=%v", p.Warnings)
			}
			found := false
			for _, c := range p.Commands {
				if shellJoin(c.Refresh) == tc.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing refresh %q in plan: %s", tc.want, JSON(p.Commands))
			}
		})
	}
}

func TestDryRunShowsRefreshRetryHint(t *testing.T) {
	withEnv(t, "INSTALLY_FORCE_OS", "linux")
	withEnv(t, "INSTALLY_FORCE_PM", "apt")
	p := BuildPlan([]Task{{Kind: "pkg", Items: []string{"git"}}}, Options{Yes: true, DryRun: true})
	res := RunPlan(p, true)
	if !strings.Contains(res.Output, "on failure: apt-get update && retry once") {
		t.Fatalf("dry-run should show refresh/retry hint, got:\n%s", res.Output)
	}
}

func TestDiagnoseCommonPackageManagerFailures(t *testing.T) {
	c := CommandSpec{Title: "test", Cmd: []string{"apt-get", "install", "missing"}, Refresh: []string{"apt-get", "update"}}
	d := diagnoseCommandFailure(c, "E: Unable to locate package missing")
	if !strings.Contains(d, "refreshed apt metadata") || !strings.Contains(d, "package still was not found") {
		t.Fatalf("unexpected diagnostic: %q", d)
	}
}
