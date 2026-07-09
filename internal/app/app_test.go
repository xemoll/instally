package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBatch(t *testing.T) {
	tasks := ParseBatchText("pkg: git htop\nflatpak: com.visualstudio.code\ngithub: sharkdp/fd\nhttps://github.com/cli/cli\n")
	if len(tasks) < 3 {
		t.Fatalf("too few tasks: %#v", tasks)
	}
	foundGitHub := false
	for _, task := range tasks {
		if task.Kind == "github" {
			foundGitHub = true
		}
	}
	if !foundGitHub {
		t.Fatalf("github task not found: %#v", tasks)
	}
}

func TestForcedManagers(t *testing.T) {
	cases := []string{"pacman", "apt", "dnf", "zypper", "apk", "xbps", "eopkg", "brew", "winget", "scoop", "choco"}
	for _, pm := range cases {
		t.Run(pm, func(t *testing.T) {
			os.Setenv("INSTALLY_FORCE_PM", pm)
			defer os.Unsetenv("INSTALLY_FORCE_PM")
			p := BuildPlan([]Task{{Kind: "pkg", Items: []string{"git"}}}, Options{Yes: true, DryRun: true})
			if len(p.Commands) == 0 {
				t.Fatalf("no commands for %s", pm)
			}
			if !strings.Contains(commandLine(p.Commands[0]), pm) && pm != "brew" && pm != "scoop" && pm != "choco" {
				t.Fatalf("unexpected command: %s", commandLine(p.Commands[0]))
			}
		})
	}
}

func TestLocalHandlers(t *testing.T) {
	os.Setenv("INSTALLY_FORCE_PM", "apt")
	defer os.Unsetenv("INSTALLY_FORCE_PM")
	p := BuildPlan([]Task{{Kind: "local", Items: []string{"/tmp/test.deb"}}}, Options{Yes: true, DryRun: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[0]), "--install-local-safe") {
		t.Fatalf("bad safe local plan: %#v", p.Commands)
	}
	unsafe := BuildPlan([]Task{{Kind: "local", Items: []string{"/tmp/test.deb"}}}, Options{Yes: true, DryRun: true, NoSecurity: true})
	if len(unsafe.Commands) == 0 || !strings.Contains(commandLine(unsafe.Commands[0]), "apt") {
		t.Fatalf("bad deb install plan: %#v", unsafe.Commands)
	}
}

func TestGUIHTML(t *testing.T) {
	if !strings.Contains(HTMLForTests(), "Instally") || !strings.Contains(HTMLForTests(), "/api/run") || !strings.Contains(HTMLForTests(), "/api/upload-scan") {
		t.Fatal("GUI html missing essentials")
	}
}

func TestKnownAppAlias(t *testing.T) {
	task := AutoTask("vscode")
	if task.Kind != "app" || len(task.Items) != 1 || task.Items[0] != "vscode" {
		t.Fatalf("bad known app task: %#v", task)
	}
}

func TestGitHubSmartPlan(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "github", Items: []string{"cli/cli"}}}, Options{Yes: true, DryRun: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[0]), "--install-github-release") {
		t.Fatalf("bad github plan: %#v", p.Commands)
	}
}

func TestReleaseAssetScoring(t *testing.T) {
	linux := SystemInfo{Family: Linux, Manager: Manager{ID: "apt"}}
	if scoreAsset("tool-linux-amd64.AppImage", "", linux) <= 0 {
		t.Fatal("linux AppImage should score")
	}
	if scoreAsset("tool-windows-amd64.exe", "", linux) != 0 {
		t.Fatal("windows exe must not score for linux")
	}
	win := SystemInfo{Family: Windows, Manager: Manager{ID: "winget"}}
	if scoreAsset("tool-windows-x64.msi", "", win) <= 0 {
		t.Fatal("windows msi should score")
	}
	mac := SystemInfo{Family: Darwin, Manager: Manager{ID: "brew"}}
	if scoreAsset("tool-macos-universal.dmg", "", mac) <= 0 {
		t.Fatal("mac dmg should score")
	}
}

func TestKnownAppExpanded(t *testing.T) {
	for _, name := range []string{"obsidian", "github-desktop", "yt-dlp"} {
		task := AutoTask(name)
		if task.Kind != "app" {
			t.Fatalf("%s should be known app: %#v", name, task)
		}
	}
}

func TestReleaseAssetScoringMoreFormats(t *testing.T) {
	linux := SystemInfo{Family: Linux, Arch: "amd64", Manager: Manager{ID: "apt"}}
	if scoreAsset("tool-x86_64-unknown-linux-gnu.tar.zst", "", linux) <= 0 {
		t.Fatal("linux tar.zst should score")
	}
	win := SystemInfo{Family: Windows, Arch: "amd64", Manager: Manager{ID: "winget"}}
	if scoreAsset("tool-win64-portable.7z", "", win) <= 0 {
		t.Fatal("windows portable 7z should score")
	}
}

func TestSecurityScanLimitedWithoutScanner(t *testing.T) {
	f, err := os.CreateTemp("", "instally-safe-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString("hello")
	_ = f.Close()
	rep := ScanFile(f.Name(), SecurityOptions{})
	if rep.SHA256 == "" {
		t.Fatal("sha missing")
	}
	if rep.Status == "unsafe" || rep.Status == "error" {
		t.Fatalf("unexpected status: %#v", rep)
	}
}

func TestInspectInputText(t *testing.T) {
	res := InspectInputText("https://example.com/app.AppImage")
	if !res.OK || len(res.Sources) != 1 || !res.Sources[0].NeedsDownload || res.Sources[0].Kind != "url" {
		t.Fatalf("bad inspect result: %#v", res)
	}
}

func TestTasksForCheckedInstallUsesCachedFile(t *testing.T) {
	scan := ScanInputResult{OK: true, Safe: true, Targets: []ScanTarget{{Kind: "url", Item: "https://example.com/a.AppImage", Path: "/tmp/cache/a.AppImage"}}}
	tasks := TasksForCheckedInstall(scan, ParseBatchText("https://example.com/a.AppImage"))
	if len(tasks) != 1 || tasks[0].Kind != "local" || tasks[0].Items[0] != "/tmp/cache/a.AppImage" {
		t.Fatalf("expected cached local install task, got %#v", tasks)
	}
}

func TestGitHubPrefixedURLNormalizesToRelease(t *testing.T) {
	tasks := ParseBatchText("github: https://github.com/cli/cli/releases/latest")
	if len(tasks) != 1 || tasks[0].Kind != "github" || len(tasks[0].Items) != 1 || tasks[0].Items[0] != "cli/cli" {
		t.Fatalf("github URL was not normalized: %#v", tasks)
	}
	p := BuildPlan(tasks, Options{Yes: true, DryRun: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[0]), "--install-github-release cli/cli") {
		t.Fatalf("bad github URL plan: %#v", p.Commands)
	}
}

func TestURLPlanUsesGoSafeInstaller(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "url", Items: []string{"https://example.com/app.AppImage"}}}, Options{Yes: true, DryRun: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[0]), "--install-url-safe") {
		t.Fatalf("URL should use built-in safe installer: %#v", p.Commands)
	}
}

func TestLinuxRunInstallerHandledAfterScan(t *testing.T) {
	os.Setenv("INSTALLY_FORCE_PM", "apt")
	defer os.Unsetenv("INSTALLY_FORCE_PM")
	cache := cacheDir()
	os.MkdirAll(cache, 0o700)
	p := BuildPlan([]Task{{Kind: "local", Items: []string{filepath.Join(cache, "setup.run")}}}, Options{Yes: true, DryRun: true, NoSecurity: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[0]), "chmod +x") {
		t.Fatalf("run installer should be executable after scan: %#v", p.Commands)
	}
}

func TestMoreKnownApps(t *testing.T) {
	for _, name := range []string{"vlc", "gimp", "lazygit", "onlyoffice", "localsend"} {
		if task := AutoTask(name); task.Kind != "app" {
			t.Fatalf("%s should be known app: %#v", name, task)
		}
	}
}

func TestHumanizedGUIStyleEssentials(t *testing.T) {
	html := HTMLForTests()
	for _, want := range []string{"--sky", "sourceHint", "Проверить и установить", "Серьёзных угроз не найдено", "Всё выглядит нормально"} {
		if !strings.Contains(html, want) {
			t.Fatalf("GUI missing %s", want)
		}
	}
}

func TestSafeRunTextDryRunVirtualPackage(t *testing.T) {
	var b strings.Builder
	res := SafeRunText("vscode", Options{Yes: true, DryRun: true}, &b)
	if !res.OK {
		t.Fatalf("safe run dry-run should be ok: %#v\n%s", res, b.String())
	}
	if !strings.Contains(b.String(), "сначала проверяем") || !strings.Contains(b.String(), "проверка пройдена") {
		t.Fatalf("safe run output is not human readable: %s", b.String())
	}
}

func TestURLDryRunKeepsInstallerExtension(t *testing.T) {
	res := InstallURLSafe("https://example.com/app.AppImage", Options{Yes: true, DryRun: true})
	if !res.OK || !strings.Contains(res.Output, "app.AppImage") || !strings.Contains(res.Output, "would scan cached file") {
		t.Fatalf("URL dry-run should preserve file name and explain scan: %#v", res)
	}
}

func TestArchive7zHandledAfterScan(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "local", Items: []string{"/tmp/source.7z"}}}, Options{Yes: true, DryRun: true, NoSecurity: true})
	if len(p.Commands) == 0 || !strings.Contains(commandLine(p.Commands[len(p.Commands)-1]), "7z") {
		t.Fatalf("7z archive should have extraction plan: %#v", p.Commands)
	}
}

func TestSupportSummaryPresent(t *testing.T) {
	s := SupportSummary()
	for _, want := range []string{"Support matrix", "Пакеты системы", "Скачивание URL"} {
		if !strings.Contains(s, want) {
			t.Fatalf("support summary missing %s: %s", want, s)
		}
	}
}

func TestPrivateURLBlockedByDefault(t *testing.T) {
	if _, err := PreviewURLCachePath("http://127.0.0.1/app.AppImage"); err == nil {
		t.Fatal("private localhost URL must be blocked by default")
	}
}

func TestForceArchAffectsReleaseAssetScoring(t *testing.T) {
	linuxArm := SystemInfo{Family: Linux, Arch: "arm64", Manager: Manager{ID: "apt"}}
	if scoreAsset("tool-linux-x86_64.AppImage", "", linuxArm) != 0 {
		t.Fatal("x86_64 asset must not score for forced arm64")
	}
	if scoreAsset("tool-linux-aarch64.AppImage", "", linuxArm) <= 0 {
		t.Fatal("aarch64 asset should score for forced arm64")
	}
}

func TestInstallLocalSafeDryRunShowsPlan(t *testing.T) {
	f, err := os.CreateTemp("", "instally-local-*.AppImage")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString("#!/bin/sh\necho ok\n")
	_ = f.Close()
	res := InstallLocalSafe(f.Name(), Options{Yes: true, DryRun: true, AllowUnknown: true})
	if !res.OK || !strings.Contains(res.Output, "План установки") || !strings.Contains(res.Output, "AppImage") {
		t.Fatalf("dry-run should include install plan, got: %s", res.Output)
	}
}

func TestVirusTotalConfigSaveStatusAndEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("INSTALLY_DATA_DIR", dir)
	t.Setenv("INSTALLY_VT_API_KEY", "")
	if err := SaveVirusTotalKey("test-key-123"); err != nil {
		t.Fatal(err)
	}
	sec := SecurityOptionsFromEnv()
	if sec.VirusTotalKey != "test-key-123" {
		t.Fatalf("saved VT key not loaded: %#v", sec)
	}
	st := VirusTotalStatus()
	if !strings.Contains(st, "VirusTotal") || strings.Contains(st, "test-key-123") {
		t.Fatalf("status should mention VT and not leak key: %s", st)
	}
	if err := ClearVirusTotalKey(); err != nil {
		t.Fatal(err)
	}
	if SecurityOptionsFromEnv().VirusTotalKey != "" {
		t.Fatal("VT key should be cleared")
	}
}

func TestLanguageConfigEnglish(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("INSTALLY_DATA_DIR", dir)
	t.Setenv("INSTALLY_LANG", "")
	if err := SaveLanguage("en"); err != nil {
		t.Fatal(err)
	}
	if AppLanguage() != "en" || T("install.safe") != "Check and install" {
		t.Fatalf("english localization did not apply: lang=%s text=%s", AppLanguage(), T("install.safe"))
	}
}

func TestVirusTotalLargeUploadLimitParsing(t *testing.T) {
	t.Setenv("INSTALLY_VT_MAX_UPLOAD_MB", "128")
	if got := vtMaxUploadSize(); got != 128*1024*1024 {
		t.Fatalf("bad max upload size: %d", got)
	}
}

func TestParseMultiItemsCommaAndSemicolon(t *testing.T) {
	tasks := ParseMultiItems("vscode, discord; github:cli/cli\nhttps://example.com/app.AppImage")
	seen := map[string]bool{}
	for _, task := range tasks {
		seen[task.Kind] = true
	}
	if !seen["app"] || !seen["github"] || !seen["url"] {
		t.Fatalf("multi parse missed expected kinds: %#v", tasks)
	}
}

func TestEmbeddedEICARDetectionBlocks(t *testing.T) {
	f, err := os.CreateTemp("", "instally-eicar-*.com")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString(eicarTestString)
	_ = f.Close()
	rep := ScanFile(f.Name(), SecurityOptions{})
	if rep.Status != "unsafe" || !rep.Blocked {
		t.Fatalf("EICAR must be blocked: %#v", rep)
	}
}

func TestSecuritySelfTestMentionsEICAR(t *testing.T) {
	out := SecuritySelfTest()
	if !strings.Contains(out, "EICAR") || !strings.Contains(out, "OK:") {
		t.Fatalf("bad self-test output: %s", out)
	}
}

func TestVirusTotalSkippedIsOptionalText(t *testing.T) {
	f, err := os.CreateTemp("", "instally-vt-optional-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString("hello")
	_ = f.Close()
	rep := ScanFile(f.Name(), SecurityOptions{})
	found := false
	for _, c := range rep.Checks {
		if c.Name == "VirusTotal" && c.Status == "skipped" && strings.Contains(c.Detail, "локальные проверки") {
			found = true
		}
	}
	if !found {
		t.Fatalf("VirusTotal optional skip missing: %#v", rep.Checks)
	}
}

func TestMultiFlagPlanEquivalent(t *testing.T) {
	tasks := ParseMultiItems("vscode", "discord", "github:cli/cli")
	p := BuildPlan(tasks, Options{Yes: true, DryRun: true})
	if len(p.Commands) < 2 {
		t.Fatalf("multi plan should produce several commands: %#v", p.Commands)
	}
}

func TestWhyOutput(t *testing.T) {
	out := Why("firefox")
	if !strings.Contains(out, "Reason") && !strings.Contains(out, "Not in known-apps") {
		t.Fatalf("Why output missing reason: %s", out)
	}
}

func TestWhichOutput(t *testing.T) {
	out := Which("git")
	if !strings.Contains(out, "App:") {
		t.Fatalf("Which output missing App: %s", out)
	}
}

func TestEnvReport(t *testing.T) {
	t.Setenv("INSTALLY_LANG", "en")
	out := EnvReport()
	if !strings.Contains(out, "INSTALLY_LANG") || !strings.Contains(out, "source:") {
		t.Fatalf("EnvReport missing set var: %s", out)
	}
}

func TestVerifyInstalledSelf(t *testing.T) {
	out := VerifyInstalled([]string{"git"})
	if !strings.Contains(out, "✓") && !strings.Contains(out, "✗") {
		t.Fatalf("VerifyInstalled missing check marks: %s", out)
	}
}

func TestExportPlan(t *testing.T) {
	f, _ := os.CreateTemp("", "instally-plan-*.json")
	defer os.Remove(f.Name())
	_ = f.Close()
	tasks := []Task{{Kind: "pkg", Items: []string{"git"}}}
	err := ExportPlan(tasks, Options{DryRun: true}, f.Name())
	if err != nil {
		t.Fatalf("ExportPlan failed: %s", err)
	}
	data, _ := os.ReadFile(f.Name())
	if !strings.Contains(string(data), `"tasks"`) {
		t.Fatalf("ExportPlan missing tasks in JSON: %s", string(data))
	}
}

func TestPurgeCache(t *testing.T) {
	// Just ensure no panic
	n := PurgeCache()
	if n < 0 {
		t.Fatalf("PurgeCache returned negative: %d", n)
	}
}

func TestAutoComplete(t *testing.T) {
	bash := AutoComplete("bash")
	if !strings.Contains(bash, "complete -F") {
		t.Fatalf("bash completion wrong: %s", bash)
	}
	zsh := AutoComplete("zsh")
	if !strings.Contains(zsh, "compdef") {
		t.Fatalf("zsh completion wrong: %s", zsh)
	}
}

func TestBuildInfo(t *testing.T) {
	info := BuildInfo()
	if !strings.Contains(info, "Instally") || !strings.Contains(info, "Go version") {
		t.Fatalf("BuildInfo wrong: %s", info)
	}
}

func TestAppStats(t *testing.T) {
	s := AppStats()
	if !strings.Contains(s, "Known apps:") {
		t.Fatalf("AppStats missing count: %s", s)
	}
}
