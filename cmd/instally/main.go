package main

import (
	"flag"
	"fmt"
	"instally/internal/app"
	"io"
	"os"
	"strings"
)

type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var pkgs, aur, flatpak, snap, git, release, local, urls, pipx, npm, cargo, golang, multi, presets listFlag
	var batch, text, installGitHubRelease, scanPath, installLocalSafe, installURLSafe, vtKey, vtSaveKey, lang, logPath, exportPlanPath string
	var dry, yes, detect, doctor, support, gui, legacyWebGUI, noOpen, prepare, fullSetup, setDefault, unsetDefault, installSelf, vtUpload, allowUnknown, trustedOfficialScript, vtStatus, vtClearKey, vtSaveKeyStdin, vtTest, securityTest, continueOnError, terminalMode, compatMatrix, listApps, listPresets, updateMode, upgradeMode, purgeCache, buildInfo, statsMode, fixBroken, envMode bool
	var version, verifyInstalled, search, which, why, depends bool
	var port int
	var completions string

	flag.Var(&pkgs, "pkg", "native package")
	flag.Var(&aur, "aur", "AUR package")
	flag.Var(&flatpak, "flatpak", "Flatpak app id")
	flag.Var(&snap, "snap", "Snap package")
	flag.Var(&git, "git", "Git repository or owner/repo")
	flag.Var(&release, "github", "GitHub owner/repo or latest release")
	flag.Var(&release, "release", "GitHub release owner/repo")
	flag.Var(&local, "local", "local file")
	flag.Var(&local, "open-file", "open file from desktop association")
	flag.Var(&urls, "url", "download URL")
	flag.Var(&pipx, "pipx", "pipx package")
	flag.Var(&npm, "npm", "npm package")
	flag.Var(&cargo, "cargo", "cargo crate")
	flag.Var(&golang, "go", "go package")
	flag.Var(&multi, "multi", "multi install items")
	flag.Var(&presets, "preset", "install preset")
	flag.StringVar(&batch, "batch", "", "batch file")
	flag.StringVar(&text, "text", "", "batch text")
	flag.StringVar(&installGitHubRelease, "install-github-release", "", "install latest compatible GitHub release asset")
	flag.StringVar(&scanPath, "scan", "", "scan local file and print security report")
	flag.StringVar(&installLocalSafe, "install-local-safe", "", "scan local file, then install")
	flag.StringVar(&installURLSafe, "install-url-safe", "", "download URL, scan, install")
	flag.StringVar(&vtKey, "vt-key", "", "VirusTotal API key")
	flag.StringVar(&vtSaveKey, "vt-save-key", "", "save VirusTotal API key")
	flag.BoolVar(&vtSaveKeyStdin, "vt-save-key-stdin", false, "read VirusTotal key from stdin")
	flag.BoolVar(&vtClearKey, "vt-clear-key", false, "remove saved VirusTotal key")
	flag.BoolVar(&vtStatus, "vt-status", false, "VirusTotal configuration status")
	flag.BoolVar(&vtTest, "vt-test", false, "test VirusTotal key with EICAR hash")
	flag.BoolVar(&securityTest, "security-test", false, "run EICAR self-test")
	flag.BoolVar(&continueOnError, "continue-on-error", false, "continue after command fails")
	flag.StringVar(&logPath, "log", "", "write log to file")
	flag.StringVar(&exportPlanPath, "export-plan", "", "export plan as JSON")
	flag.BoolVar(&terminalMode, "terminal", false, "terminal installer")
	flag.BoolVar(&terminalMode, "terminal-install", false, "alias for --terminal")
	flag.BoolVar(&updateMode, "update", false, "update specified apps")
	flag.BoolVar(&upgradeMode, "upgrade-all", false, "upgrade all packages")
	flag.BoolVar(&purgeCache, "purge-cache", false, "clear cache")
	flag.BoolVar(&fixBroken, "fix-broken", false, "repair package manager state")
	flag.BoolVar(&buildInfo, "build-info", false, "show build info")
	flag.BoolVar(&statsMode, "stats", false, "show app statistics")
	flag.BoolVar(&envMode, "env", false, "show environment variables")
	flag.BoolVar(&version, "version", false, "print version")
	flag.BoolVar(&verifyInstalled, "verify-installed", false, "check if installed")
	flag.BoolVar(&search, "search", false, "search packages")
	flag.BoolVar(&which, "which", false, "locate app binary")
	flag.BoolVar(&why, "why", false, "explain install method")
	flag.BoolVar(&depends, "depends", false, "show dependencies")
	flag.StringVar(&completions, "completions", "", "generate shell completions")
	flag.BoolVar(&compatMatrix, "compat-matrix", false, "run compatibility matrix")
	flag.BoolVar(&listApps, "list-apps", false, "list known apps")
	flag.BoolVar(&listPresets, "list-presets", false, "list presets")
	flag.StringVar(&lang, "lang", "", "language: ru or en")
	flag.BoolVar(&dry, "dry-run", false, "print commands without executing")
	flag.BoolVar(&yes, "yes", false, "assume yes")
	flag.BoolVar(&detect, "detect", false, "detect system")
	flag.BoolVar(&doctor, "doctor", false, "full diagnostics")
	flag.BoolVar(&support, "support", false, "support matrix")
	flag.BoolVar(&gui, "gui", false, "start TUI")
	flag.BoolVar(&legacyWebGUI, "legacy-web-gui", false, "start web GUI")
	flag.BoolVar(&noOpen, "no-open", false, "do not open browser for web GUI")
	flag.IntVar(&port, "port", 0, "GUI port")
	flag.BoolVar(&prepare, "prepare", false, "install build dependencies")
	flag.BoolVar(&fullSetup, "full-setup", false, "install deps, menu entry, defaults")
	flag.BoolVar(&setDefault, "set-default-installer", false, "register as default installer")
	flag.BoolVar(&unsetDefault, "unset-default-installer", false, "unregister defaults")
	flag.BoolVar(&installSelf, "install-self", false, "install Instally")
	flag.BoolVar(&vtUpload, "vt-upload", false, "allow upload to VirusTotal")
	flag.BoolVar(&allowUnknown, "allow-unknown", false, "allow install with limited scan")
	flag.BoolVar(&trustedOfficialScript, "trusted-official-script", false, "skip scan for allowlisted installers")

	flag.CommandLine.Parse(normalizeArgs(os.Args[1:]))

	args := flag.Args()

	if lang != "" {
		app.SetAppLanguage(lang)
		_ = app.SaveLanguage(lang)
	}
	if vtSaveKey != "" {
		if err := app.SaveVirusTotalKey(vtSaveKey); err != nil {
			fatal(err)
		}
		fmt.Println("VirusTotal key saved")
		return
	}
	if vtSaveKeyStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			fatal(err)
		}
		if err := app.SaveVirusTotalKey(strings.TrimSpace(string(b))); err != nil {
			fatal(err)
		}
		fmt.Println("VirusTotal key saved")
		return
	}
	if vtClearKey {
		if err := app.ClearVirusTotalKey(); err != nil {
			fatal(err)
		}
		fmt.Println("VirusTotal key removed")
		return
	}
	if vtStatus {
		fmt.Print(app.VirusTotalStatus())
		return
	}
	if vtTest {
		fmt.Print(app.VirusTotalSelfTestWithConfiguredKey())
		return
	}
	if version {
		fmt.Println(app.VersionInfo())
		return
	}
	if listApps {
		fmt.Print(app.KnownAppsList())
		return
	}
	if listPresets {
		fmt.Print(app.PresetListFormatted())
		return
	}
	if gui {
		if legacyWebGUI {
			fatal(app.ServeGUI(app.ServerOptions{Port: port, NoOpen: noOpen}))
			return
		}
		exitCode := app.RunTUI(app.Options{})
		if exitCode == 2 {
			os.Exit(app.RunTerminalInstaller(app.Options{}))
		}
		os.Exit(exitCode)
		return
	}
	if detect {
		fmt.Println(app.JSON(app.Detect()))
		return
	}
	if doctor {
		fmt.Print(app.Doctor())
		return
	}
	if support {
		fmt.Print(app.SupportSummary())
		return
	}
	if compatMatrix {
		fmt.Print(app.CompatibilityMatrixReport())
		return
	}
	if securityTest {
		fmt.Print(app.SecuritySelfTest())
		return
	}
	if purgeCache {
		count := app.PurgeCache()
		fmt.Printf("Cache purged: %d files removed\n", count)
		return
	}
	if buildInfo {
		fmt.Print(app.BuildInfo())
		return
	}
	if statsMode {
		fmt.Print(app.AppStats())
		return
	}
	if envMode {
		fmt.Print(app.EnvReport())
		return
	}
	if fixBroken {
		fmt.Print(app.FixBroken())
		return
	}
	if completions != "" {
		fmt.Print(app.AutoComplete(completions))
		return
	}
	if which {
		for _, a := range args {
			fmt.Print(app.Which(a))
		}
		return
	}
	if why {
		for _, a := range args {
			fmt.Print(app.Why(a))
		}
		return
	}
	if verifyInstalled {
		fmt.Print(app.VerifyInstalled(args))
		return
	}
	if search {
		query := strings.Join(args, " ")
		if query == "" {
			fmt.Println("Usage: instally --search <query>")
			return
		}
		fmt.Print(app.SearchPackages(query))
		return
	}
	if updateMode {
		items := args
		if len(items) == 0 {
			items = append(items, pkgs...)
			items = append(items, flatpak...)
		}
		if len(items) == 0 {
			fmt.Println("Usage: instally --update firefox discord git")
			return
		}
		plan := app.BuildUpdatePlan(items, app.Options{})
		runPlan(plan, dry)
		return
	}
	if upgradeMode {
		plan := app.BuildUpgradePlan(app.Options{})
		runPlan(plan, dry)
		return
	}

	securityOpts := app.SecurityOptionsFromEnv()
	if vtKey != "" {
		securityOpts.VirusTotalKey = vtKey
	}
	if vtUpload {
		securityOpts.VirusTotalUpload = true
	}
	if allowUnknown {
		securityOpts.AllowUnknown = true
	}
	baseOpts := app.Options{
		Yes:                   yes,
		DryRun:                dry,
		AllowUnknown:          securityOpts.AllowUnknown,
		VirusTotalKey:         securityOpts.VirusTotalKey,
		VirusTotalUpload:      securityOpts.VirusTotalUpload,
		ContinueOnError:       continueOnError,
		TrustedOfficialScript: trustedOfficialScript,
	}
	if terminalMode {
		os.Exit(app.RunTerminalInstaller(baseOpts))
	}
	if scanPath != "" {
		rep := app.ScanFile(scanPath, securityOpts)
		fmt.Println(app.JSON(rep))
		if rep.Status == "unsafe" || rep.Status == "error" {
			os.Exit(2)
		}
		return
	}
	if installLocalSafe != "" {
		res := app.InstallLocalSafe(installLocalSafe, baseOpts)
		fmt.Print(res.Output)
		if !res.OK {
			os.Exit(res.ExitCode)
		}
		return
	}
	if installURLSafe != "" {
		res := app.InstallURLSafe(installURLSafe, baseOpts)
		fmt.Print(res.Output)
		if !res.OK {
			os.Exit(res.ExitCode)
		}
		return
	}
	if installGitHubRelease != "" {
		res := app.InstallGitHubRelease(installGitHubRelease, baseOpts)
		fmt.Print(res.Output)
		if !res.OK {
			os.Exit(1)
		}
		return
	}
	if fullSetup {
		runPlan(app.FullSetupPlan(app.SelfPath()), dry || !yes)
		return
	}
	if installSelf {
		runPlan(app.Plan{System: app.Detect(), Tasks: []app.Task{{Kind: "install-self", Items: []string{app.SelfPath()}}}, Commands: app.InstallCommands(app.SelfPath(), false)}, dry || !yes)
		return
	}
	if setDefault {
		runPlan(app.Plan{System: app.Detect(), Tasks: []app.Task{{Kind: "set-default"}}, Commands: app.SetDefaultCommands()}, dry || !yes)
		return
	}
	if unsetDefault {
		runPlan(app.Plan{System: app.Detect(), Tasks: []app.Task{{Kind: "unset-default"}}, Commands: app.UnsetDefaultCommands()}, dry || !yes)
		return
	}

	var tasks []app.Task
	if prepare {
		tasks = append(tasks, app.Task{Kind: "pkg", Items: app.Detect().Manager.Prepare})
	}
	add := func(kind string, items []string) {
		if len(items) > 0 {
			tasks = append(tasks, app.Task{Kind: kind, Items: items})
		}
	}
	add("pkg", pkgs)
	add("aur", aur)
	add("flatpak", flatpak)
	add("snap", snap)
	add("git", git)
	add("release", release)
	add("local", local)
	add("url", urls)
	add("pipx", pipx)
	add("npm", npm)
	add("cargo", cargo)
	add("go", golang)
	add("preset", presets)
	if len(multi) > 0 {
		tasks = append(tasks, app.ParseMultiItems(multi...)...)
	}
	if batch != "" {
		bt, err := app.ParseBatchFile(batch)
		if err != nil {
			fatal(err)
		}
		tasks = append(tasks, bt...)
	}
	if text != "" {
		tasks = append(tasks, app.ParseBatchText(text)...)
	}
	for _, arg := range args {
		tasks = append(tasks, app.AutoTask(arg))
	}
	if len(tasks) == 0 {
		if app.StdinIsTerminal() {
			os.Exit(app.RunTUI(app.Options{}))
		}
		fmt.Println("Instally: use --gui, --doctor, --pkg, --batch or pass app names/URLs/files. Example: instally firefox")
		return
	}

	if exportPlanPath != "" {
		if err := app.ExportPlan(tasks, baseOpts, exportPlanPath); err != nil {
			fatal(err)
		}
		fmt.Printf("Plan exported to %s\n", exportPlanPath)
		if !dry && logPath == "" {
			return
		}
	}

	if logPath != "" {
		plan := app.BuildPlan(tasks, baseOpts)
		res := app.RunPlanStream(plan, dry, os.Stdout)
		_ = os.WriteFile(logPath, []byte(res.Output), 0o600)
		if !res.OK {
			os.Exit(1)
		}
		return
	}

	plan := app.BuildPlan(tasks, baseOpts)
	runPlan(plan, dry)
}

func normalizeArgs(args []string) []string {
	multi := map[string]bool{"--pkg": true, "--package": true, "--aur": true, "--flatpak": true, "--snap": true, "--git": true, "--github": true, "--release": true, "--local": true, "--open-file": true, "--url": true, "--pipx": true, "--npm": true, "--cargo": true, "--go": true, "--multi": true, "--preset": true}
	value := map[string]bool{"--batch": true, "--text": true, "--port": true, "--install-github-release": true, "--install-local-safe": true, "--install-url-safe": true, "--scan": true, "--vt-key": true, "--vt-save-key": true, "--lang": true, "--log": true, "--export-plan": true}
	var out []string
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			name := a
			if eq := strings.IndexByte(a, '='); eq >= 0 {
				name = a[:eq]
			}
			if multi[name] && !strings.Contains(a, "=") {
				used := false
				for i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					out = append(out, a, args[i+1])
					i++
					used = true
				}
				if !used {
					out = append(out, a)
				}
				continue
			}
			if value[name] && !strings.Contains(a, "=") && i+1 < len(args) {
				out = append(out, a, args[i+1])
				i++
				continue
			}
			out = append(out, a)
		} else {
			positional = append(positional, a)
		}
	}
	return append(out, positional...)
}

func runPlan(plan app.Plan, dry bool) {
	res := app.RunPlan(plan, dry)
	fmt.Print(res.Output)
	if !res.OK {
		os.Exit(1)
	}
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
