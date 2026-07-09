package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Options struct {
	Yes                   bool
	DryRun                bool
	NoSecurity            bool
	AllowUnknown          bool
	VirusTotalKey         string
	VirusTotalUpload      bool
	ContinueOnError       bool
	TrustedOfficialScript bool
	Verbose               bool
	Quiet                 bool
}

func BuildPlan(tasks []Task, opts Options) Plan {
	_ = ensureDirs()
	sys := Detect()
	p := Plan{System: sys, Tasks: mergeTasks(tasks), ContinueOnError: opts.ContinueOnError}
	for _, task := range p.Tasks {
		switch task.Kind {
		case "app":
			p.addKnownApps(task.Items, opts)
		case "preset":
			for _, t := range presetTasks(task.Items) {
				p.addTaskInline(t, opts)
			}
		case "pkg":
			p.addNative(task.Items, opts)
		case "aur":
			p.addAUR(task.Items, opts)
		case "flatpak":
			p.addFlatpak(task.Items, opts)
		case "snap":
			p.addSnap(task.Items, opts)
		case "pip":
			p.addToolInstall("pip", []string{"python3", "-m", "pip", "install", "--user"}, task.Items, false)
		case "pipx":
			p.ensureTool("pipx")
			for _, item := range task.Items {
				p.Commands = append(p.Commands, CommandSpec{Title: "pipx install " + item, Cmd: []string{"pipx", "install", item}})
			}
		case "npm":
			p.addNPMGlobal(task.Items, opts)
		case "cargo":
			p.ensureTool("cargo")
			p.addToolInstall("cargo", []string{"cargo", "install"}, task.Items, false)
		case "go":
			p.ensureTool("go")
			for _, item := range task.Items {
				p.Commands = append(p.Commands, CommandSpec{Title: "go install " + item, Cmd: []string{"go", "install", item}})
			}
		case "official-firefox":
			p.addFirefoxOfficial(opts)
		case "official-discord":
			p.addDiscordOfficial(opts)
		case "github":
			for _, item := range task.Items {
				p.addGitHub(item, opts)
			}
		case "git":
			p.ensureTool("git")
			for _, item := range task.Items {
				p.addGit(item)
			}
		case "release":
			for _, item := range task.Items {
				p.addRelease(item, opts)
			}
		case "url":
			for _, item := range task.Items {
				p.addURL(item, opts)
			}
		case "local":
			for _, item := range task.Items {
				p.addLocal(item, opts)
			}
		case "winget", "scoop", "choco", "brew", "mas":
			p.addExplicitManager(task.Kind, task.Items, opts)
		default:
			p.Warnings = append(p.Warnings, "Неизвестный тип задачи: "+task.Kind)
		}
	}
	p.Commands = dedupeCommands(p.Commands)
	return p
}

func dedupeCommands(in []CommandSpec) []CommandSpec {
	seen := map[string]bool{}
	out := make([]CommandSpec, 0, len(in))
	for _, c := range in {
		key := c.Title + "\x00" + commandLine(c)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func (p *Plan) addKnownApps(items []string, opts Options) {
	for _, item := range items {
		key, ok := knownAppKey(item)
		if !ok {
			p.addNative([]string{item}, opts)
			continue
		}
		app := knownApps[key]
		switch p.System.Family {
		case Linux:
			if app.Linux.Kind != "" {
				p.addTaskInline(app.Linux, opts)
				continue
			}
		case Windows:
			if app.Windows != "" {
				p.addNative([]string{app.Windows}, opts)
				continue
			}
		case Darwin:
			if app.Mac != "" && p.System.Manager.ID == "brew" {
				cmd := []string{"brew", "install"}
				if app.MacCask {
					cmd = append(cmd, "--cask")
				}
				cmd = append(cmd, app.Mac)
				p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Homebrew install " + app.Name, Cmd: cmd}, profiles()["brew"]))
				continue
			}
		}
		if app.GitHub != "" {
			p.addGitHub(app.GitHub, opts)
			continue
		}
		p.addNative([]string{item}, opts)
	}
}

func (p *Plan) addTaskInline(task Task, opts Options) {
	switch task.Kind {
	case "app":
		p.addKnownApps(task.Items, opts)
	case "pkg":
		p.addNative(task.Items, opts)
	case "flatpak":
		p.addFlatpak(task.Items, opts)
	case "snap":
		p.addSnap(task.Items, opts)
	case "github":
		for _, item := range task.Items {
			p.addGitHub(item, opts)
		}
	case "release":
		for _, item := range task.Items {
			p.addRelease(item, opts)
		}
	case "pipx":
		p.addToolInstall("pipx", []string{"pipx", "install"}, task.Items, false)
	case "npm":
		p.addNPMGlobal(task.Items, opts)
	case "cargo":
		p.addToolInstall("cargo", []string{"cargo", "install"}, task.Items, false)
	case "go":
		items := append([]string{}, task.Items...)
		for i := range items {
			if !strings.Contains(items[i], "@") {
				items[i] += "@latest"
			}
		}
		p.addToolInstall("go", []string{"go", "install"}, items, false)
	case "official-firefox":
		p.addFirefoxOfficial(opts)
	case "official-discord":
		p.addDiscordOfficial(opts)
	default:
		p.Warnings = append(p.Warnings, "Неизвестный профиль приложения: "+task.Kind)
	}
}

func (p *Plan) addNative(items []string, opts Options) {
	items, warnings := validateInstallItems("pkg", items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	sys := p.System
	m := sys.Manager
	items = normalizeNativePackagesForManager(m.ID, items)
	if m.ID == "none" || len(m.Install) == 0 {
		p.Warnings = append(p.Warnings, "Системный менеджер пакетов не найден")
		return
	}
	if sys.Family == Windows && m.ID == "winget" {
		for _, item := range items {
			cmd := append([]string{}, m.Install...)
			cmd = append(cmd, item)
			cmd = append(cmd, m.Yes...)
			p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "winget install " + item, Cmd: cmd, Admin: false}, m))
		}
		return
	}
	if m.ID == "nix" {
		for _, item := range items {
			cmd := []string{"nix", "profile", "install", "nixpkgs#" + item}
			p.Commands = append(p.Commands, CommandSpec{Title: "nix profile install " + item, Cmd: cmd})
		}
		return
	}
	cmd := append([]string{}, m.Install...)
	cmd = append(cmd, items...)
	if opts.Yes {
		cmd = append(cmd, m.Yes...)
	}
	p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Установка системных пакетов: " + strings.Join(items, ", "), Cmd: cmd, Admin: m.NeedsElev}, m))
}

func (p *Plan) addExplicitManager(kind string, items []string, opts Options) {
	items, warnings := validateInstallItems(kind, items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	m := profiles()[kind]
	if m.ID == "" {
		p.Warnings = append(p.Warnings, "Профиль не найден: "+kind)
		return
	}
	if commandExists(firstTool(m)) == "" && !opts.DryRun {
		p.Warnings = append(p.Warnings, kind+" не найден в PATH")
	}
	for _, item := range items {
		cmd := append([]string{}, m.Install...)
		cmd = append(cmd, item)
		if opts.Yes {
			cmd = append(cmd, m.Yes...)
		}
		p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: kind + " install " + item, Cmd: cmd, Admin: m.NeedsElev}, m))
	}
}

func (p *Plan) addAUR(items []string, opts Options) {
	items, warnings := validateInstallItems("aur", items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	if p.System.Family != Linux || p.System.Manager.ID != "pacman" {
		p.Warnings = append(p.Warnings, "AUR поддерживается только на Arch/CachyOS/Manjaro. Для других систем используй flatpak/snap/github/release.")
		return
	}
	p.ensureTool("git")
	if commandExists("paru") == "" && commandExists("yay") == "" {
		p.Commands = append(p.Commands, CommandSpec{Title: "Установка yay", Shell: "tmp=$(mktemp -d) && git clone https://aur.archlinux.org/yay-bin.git \"$tmp/yay-bin\" && cd \"$tmp/yay-bin\" && makepkg -si --noconfirm"})
	}
	helper := "yay"
	if commandExists("paru") != "" {
		helper = "paru"
	}
	cmd := []string{helper, "-S"}
	cmd = append(cmd, items...)
	if opts.Yes {
		cmd = append(cmd, "--noconfirm")
	}
	p.Commands = append(p.Commands, CommandSpec{Title: "AUR: " + strings.Join(items, ", "), Cmd: cmd})
}

func (p *Plan) addFlatpak(items []string, opts Options) {
	items, warnings := validateInstallItems("flatpak", items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	p.ensureTool("flatpak")
	scope := "--user"
	admin := false
	if boolEnv("INSTALLY_FLATPAK_SYSTEM") {
		scope = "--system"
		admin = true
	}
	p.Commands = append(p.Commands, CommandSpec{Title: "Ensure Flathub remote", Shell: fmt.Sprintf("flatpak remote-list %s 2>/dev/null | grep -q '^flathub' || flatpak remote-add %s --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo", scope, scope), Admin: admin})
	for _, item := range items {
		cmd := []string{"flatpak", "install", scope, "flathub", item}
		if opts.Yes {
			cmd = append(cmd, "-y")
		}
		p.Commands = append(p.Commands, CommandSpec{Title: "Flatpak install " + item, Cmd: cmd, Admin: admin})
	}
}

func (p *Plan) addSnap(items []string, opts Options) {
	items, warnings := validateInstallItems("snap", items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	p.ensureTool("snap")
	for _, item := range items {
		cmd := []string{"snap", "install", item}
		p.Commands = append(p.Commands, CommandSpec{Title: "Snap install " + item, Cmd: cmd, Admin: true})
	}
}

func (p *Plan) addNPMGlobal(items []string, opts Options) {
	items, warnings := validateInstallItems("npm", items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	p.ensureTool("npm")
	sys := p.System
	if sys.Family == Linux || sys.Family == Darwin {
		prefix := filepath.Join(dataDir(), "npm-global")
		bin := filepath.Join(prefix, "bin")
		cmd := fmt.Sprintf("mkdir -p %s && npm install -g --prefix %s %s", shellQuote(prefix), shellQuote(prefix), shellJoin(items))
		p.Commands = append(p.Commands, CommandSpec{Title: "npm user install " + strings.Join(items, ", "), Shell: cmd})
		p.Warnings = append(p.Warnings, "npm global packages are installed to "+prefix+". Add "+bin+" to PATH if the command is not found.")
		return
	}
	cmd := []string{"npm", "install", "-g"}
	cmd = append(cmd, items...)
	p.Commands = append(p.Commands, CommandSpec{Title: "npm install " + strings.Join(items, ", "), Cmd: cmd, Admin: false})
}

func (p *Plan) addToolInstall(name string, base, items []string, admin bool) {
	items, warnings := validateInstallItems(name, items)
	p.Warnings = append(p.Warnings, warnings...)
	if len(items) == 0 {
		return
	}
	p.ensureTool(name)
	cmd := append([]string{}, base...)
	cmd = append(cmd, items...)
	p.Commands = append(p.Commands, CommandSpec{Title: name + " install " + strings.Join(items, ", "), Cmd: cmd, Admin: admin})
}

func (p *Plan) ensureTool(tool string) {
	if _, ok := p.System.Tools[tool]; ok {
		return
	}
	pkg := toolPackage(tool, p.System.Manager.ID)
	if pkg == "" {
		p.Warnings = append(p.Warnings, "Не найден инструмент "+tool+" и неизвестно как поставить его для "+p.System.Manager.ID)
		return
	}
	m := p.System.Manager
	cmd := append([]string{}, m.Install...)
	cmd = append(cmd, pkg)
	cmd = append(cmd, m.Yes...)
	p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Автоустановка зависимости: " + tool, Cmd: cmd, Admin: m.NeedsElev}, m))
	p.System.Tools[tool] = "planned"
}

func toolPackage(tool, manager string) string {
	m := map[string]map[string]string{
		"git":     {"pacman": "git", "apt": "git", "dnf": "git", "zypper": "git", "apk": "git", "xbps": "git", "eopkg": "git", "brew": "git", "winget": "Git.Git", "scoop": "git", "choco": "git"},
		"flatpak": {"pacman": "flatpak", "apt": "flatpak", "dnf": "flatpak", "zypper": "flatpak", "apk": "flatpak", "xbps": "flatpak"},
		"snap":    {"pacman": "snapd", "apt": "snapd", "dnf": "snapd", "zypper": "snapd"},
		"pipx":    {"pacman": "python-pipx", "apt": "pipx", "dnf": "pipx", "zypper": "python3-pipx", "apk": "pipx", "xbps": "python3-pipx", "brew": "pipx", "winget": "Pypa.Pipx"},
		"npm":     {"pacman": "npm", "apt": "npm", "dnf": "npm", "zypper": "npm", "apk": "npm", "xbps": "nodejs", "brew": "node", "winget": "OpenJS.NodeJS"},
		"cargo":   {"pacman": "rust", "apt": "cargo", "dnf": "cargo", "zypper": "cargo", "apk": "cargo", "xbps": "cargo", "brew": "rust", "winget": "Rustlang.Rustup"},
		"go":      {"pacman": "go", "apt": "golang-go", "dnf": "golang", "zypper": "go", "apk": "go", "xbps": "go", "brew": "go", "winget": "GoLang.Go"},
		"curl":    {"pacman": "curl", "apt": "curl", "dnf": "curl", "zypper": "curl", "apk": "curl", "xbps": "curl", "eopkg": "curl", "brew": "curl", "winget": "cURL.cURL"},
		"gpg":     {"pacman": "gnupg", "apt": "gnupg", "dnf": "gnupg2", "zypper": "gpg2", "apk": "gnupg", "xbps": "gnupg", "eopkg": "gnupg", "brew": "gnupg"},
		"wget":    {"pacman": "wget", "apt": "wget", "dnf": "wget", "zypper": "wget", "apk": "wget", "xbps": "wget", "eopkg": "wget", "brew": "wget"},
		"7z":      {"pacman": "p7zip", "apt": "p7zip-full", "dnf": "p7zip", "zypper": "p7zip", "apk": "p7zip", "xbps": "p7zip", "eopkg": "p7zip", "brew": "p7zip", "winget": "7zip.7zip", "scoop": "7zip", "choco": "7zip"},
	}
	return m[tool][manager]
}

func (p *Plan) addFirefoxOfficial(opts Options) {
	m := p.System.Manager.ID
	switch m {
	case "apt":
		// Mozilla's official Linux instructions recommend the Mozilla APT repo for Debian/Ubuntu-based systems.
		p.ensureTool("curl")
		p.ensureTool("gpg")
		sh := "set -e; install -d -m 0755 /etc/apt/keyrings; tmp=$(mktemp); curl -fsSL https://packages.mozilla.org/apt/repo-signing-key.gpg -o \"$tmp\"; gpg -n -q --import --import-options import-show \"$tmp\" | awk '/pub/{getline; gsub(/^ +| +$/,\"\"); if($0 != \"35BAA0B33E9EB396F59CA838C0BA5CE6DC6315A3\") {print \"Mozilla signing key fingerprint mismatch: \" $0; exit 1}}'; cat \"$tmp\" > /etc/apt/keyrings/packages.mozilla.org.asc; cat > /etc/apt/sources.list.d/mozilla.sources <<'EOF'\nTypes: deb\nURIs: https://packages.mozilla.org/apt\nSuites: mozilla\nComponents: main\nSigned-By: /etc/apt/keyrings/packages.mozilla.org.asc\nEOF\ncat > /etc/apt/preferences.d/mozilla <<'EOF'\nPackage: *\nPin: origin packages.mozilla.org\nPin-Priority: 1000\nEOF\napt-get update; apt-get install firefox" + yesFlagApt(opts)
		p.Commands = append(p.Commands, CommandSpec{Title: "Install Firefox from Mozilla APT repo", Shell: sh, Admin: true})
	case "dnf":
		cmd := []string{"dnf", "install", "firefox"}
		if opts.Yes {
			cmd = append(cmd, "-y")
		}
		p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Install Firefox", Cmd: cmd, Admin: true}, p.System.Manager))
	case "zypper":
		cmd := []string{"zypper", "install"}
		if opts.Yes {
			cmd = append(cmd, "-y")
		}
		cmd = append(cmd, "firefox")
		p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Install Firefox", Cmd: cmd, Admin: true}, p.System.Manager))
	case "pacman", "apk", "xbps", "eopkg", "emerge", "nix":
		p.addNative([]string{"firefox"}, opts)
	default:
		p.addFlatpak([]string{"org.mozilla.firefox"}, opts)
	}
}

func yesFlagApt(opts Options) string {
	if opts.Yes {
		return " -y"
	}
	return ""
}

func (p *Plan) addDiscordOfficial(opts Options) {
	switch p.System.Manager.ID {
	case "apt":
		p.addURL("https://discord.com/api/download?platform=linux&format=deb", opts)
	case "dnf", "zypper":
		p.addURL("https://discord.com/api/download?platform=linux&format=rpm", opts)
	case "pacman":
		p.addURL("https://discord.com/api/download?platform=linux&format=pkg.tar.zst", opts)
	default:
		p.addFlatpak([]string{"com.discordapp.Discord"}, opts)
	}
}



func (p *Plan) addGitHub(item string, opts Options) {
	item = normalizeGitHubTarget(item)
	if !ownerRepoRE.MatchString(item) {
		p.addGit(item)
		return
	}
	p.addRelease(item, opts)
}

func (p *Plan) addGit(item string) {
	if err := validateGitTarget(item); err != nil {
		p.Warnings = append(p.Warnings, "Git target rejected: "+err.Error())
		return
	}
	url := item
	if ownerRepoRE.MatchString(item) {
		url = "https://github.com/" + item + ".git"
	}
	url = normalizeGitURL(url)
	name := repoName(url)
	dir := filepath.Join(buildDir(), name)
	sh := fmt.Sprintf("if [ -d %s/.git ]; then cd %s && git pull --ff-only; else git clone %s %s; fi && cd %s && %s", shellQuote(dir), shellQuote(dir), shellQuote(url), shellQuote(dir), shellQuote(dir), detectBuildShell())
	if runtime.GOOS == "windows" {
		sh = fmt.Sprintf("if exist %s\\.git (cd /d %s && git pull --ff-only) else git clone %s %s", winQuote(dir), winQuote(dir), winQuote(url), winQuote(dir))
	}
	p.Commands = append(p.Commands, CommandSpec{Title: "Git source: " + item, Shell: sh})
}

func detectBuildShell() string {
	return "instally_elevate(){ if [ \"$(id -u 2>/dev/null || echo 1)\" = 0 ]; then \"$@\"; elif command -v pkexec >/dev/null 2>&1 && [ -n \"${DISPLAY:-}\" ]; then pkexec \"$@\"; elif command -v sudo >/dev/null 2>&1; then sudo \"$@\"; else echo 'Instally: admin rights required for install step'; return 1; fi; }; if [ -f PKGBUILD ]; then makepkg -si --noconfirm; elif [ -f Cargo.toml ]; then cargo install --path .; elif [ -f go.mod ]; then go install ./...; elif [ -f package.json ]; then npm install && npm run build --if-present && npm link 2>/dev/null || true; elif [ -f CMakeLists.txt ]; then cmake -S . -B build && cmake --build build --parallel && instally_elevate cmake --install build; elif [ -f meson.build ]; then meson setup build && meson compile -C build && instally_elevate meson install -C build; elif [ -x ./configure ]; then ./configure && make -j$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2) && instally_elevate make install; elif [ -f Makefile ] || [ -f makefile ]; then make -j$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2) && instally_elevate make install; elif [ -f pyproject.toml ]; then python3 -m pip install --user .; elif [ -f setup.py ]; then python3 -m pip install --user .; else echo 'Instally: source cloned, build recipe not detected'; fi"
}

func (p *Plan) addRelease(item string, opts Options) {
	item = normalizeGitHubTarget(item)
	if !ownerRepoRE.MatchString(item) {
		p.Warnings = append(p.Warnings, "GitHub Release ожидает owner/repo: "+item)
		return
	}
	cmd := []string{SelfPath(), "--install-github-release", item}
	if opts.Yes {
		cmd = append(cmd, "--yes")
	}
	if opts.AllowUnknown {
		cmd = append(cmd, "--allow-unknown")
	}
	if opts.VirusTotalUpload {
		cmd = append(cmd, "--vt-upload")
	}
	p.Commands = append(p.Commands, CommandSpec{Title: "GitHub smart install: " + item, Cmd: cmd, Env: envForChildSecurity(opts)})
}

func (p *Plan) addURL(raw string, opts Options) {
	if _, err := PreviewURLCachePath(raw); err != nil {
		p.Warnings = append(p.Warnings, "URL rejected before install: "+err.Error())
		return
	}
	cmd := []string{SelfPath(), "--install-url-safe", raw}
	if opts.Yes {
		cmd = append(cmd, "--yes")
	}
	if opts.AllowUnknown {
		cmd = append(cmd, "--allow-unknown")
	}
	if opts.VirusTotalUpload {
		cmd = append(cmd, "--vt-upload")
	}
	p.Commands = append(p.Commands, CommandSpec{Title: "Download, scan and install: " + raw, Cmd: cmd, Env: envForChildSecurity(opts)})
}

func (p *Plan) addLocal(path string, opts Options) {
	path = expandPath(path)
	if !opts.NoSecurity {
		cmd := []string{SelfPath(), "--install-local-safe", path}
		if opts.Yes {
			cmd = append(cmd, "--yes")
		}
		if opts.AllowUnknown {
			cmd = append(cmd, "--allow-unknown")
		}
		if opts.VirusTotalUpload {
			cmd = append(cmd, "--vt-upload")
		}
		p.Commands = append(p.Commands, CommandSpec{Title: "Проверка безопасности и установка: " + filepath.Base(path), Cmd: cmd, Env: envForChildSecurity(opts)})
		return
	}
	ext := normalizeExt(path)
	sys := p.System
	m := sys.Manager
	if ext == ".appimage" {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		dst := filepath.Join(dataDir(), "appimages", filepath.Base(path))
		link := filepath.Join(localBin(), sanitizeName(name))
		p.Commands = append(p.Commands,
			CommandSpec{Title: "Create AppImage dirs", Cmd: []string{"mkdir", "-p", filepath.Dir(dst), filepath.Dir(link)}},
			CommandSpec{Title: "Copy AppImage", Cmd: []string{"cp", path, dst}},
			CommandSpec{Title: "Set executable", Cmd: []string{"chmod", "+x", dst}},
			CommandSpec{Title: "Symlink to bin", Cmd: []string{"ln", "-sf", dst, link}},
		)
		return
	}
	if ext == ".flatpakref" || ext == ".flatpakrepo" {
		p.ensureTool("flatpak")
		p.Commands = append(p.Commands, CommandSpec{Title: "Flatpak file " + filepath.Base(path), Cmd: []string{"flatpak", "install", "-y", path}})
		return
	}
	if isArchiveExt(ext) {
		p.addArchive(path, ext)
		return
	}
	if sys.Family == Linux && (ext == ".run" || ext == ".bin" || ext == ".sh") {
		if !strings.HasPrefix(path, cacheDir()) && !strings.HasPrefix(path, dataDir()) && !strings.HasPrefix(path, buildDir()) {
			p.Warnings = append(p.Warnings, "файл вне доверенной директории (cache/data/build), установка заблокирована: "+path)
			return
		}
		admin := ext == ".run" || ext == ".bin"
		p.Commands = append(p.Commands,
			CommandSpec{Title: "Set executable " + filepath.Base(path), Cmd: []string{"chmod", "+x", path}},
			CommandSpec{Title: "Run installer " + filepath.Base(path), Cmd: []string{path}, Admin: admin},
		)
		return
	}
	if sys.Family == Linux {
		if cmdBase, ok := m.Local[ext]; ok {
			cmd := append([]string{}, cmdBase...)
			cmd = append(cmd, path)
			cmd = append(cmd, m.Yes...)
			p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Install local " + filepath.Base(path), Cmd: cmd, Admin: m.NeedsElev}, m))
			return
		}
	}
	if sys.Family == Windows {
		switch ext {
		case ".msi":
			p.Commands = append(p.Commands, CommandSpec{Title: "Install MSI " + filepath.Base(path), Cmd: []string{"msiexec", "/i", path, "/passive"}, Admin: true})
		case ".appx", ".msix", ".appxbundle", ".msixbundle":
			p.Commands = append(p.Commands, CommandSpec{Title: "Install AppX/MSIX " + filepath.Base(path), Shell: "powershell -NoProfile -ExecutionPolicy Bypass -Command Add-AppxPackage -Path " + winPSQuote(path)})
		case ".exe":
			p.Commands = append(p.Commands, CommandSpec{Title: "Run installer " + filepath.Base(path), Cmd: []string{path}, Admin: true, Shell: ""})
		default:
			p.Warnings = append(p.Warnings, "Windows local format пока не имеет безопасной авто-команды: "+ext)
		}
		return
	}
	if sys.Family == Darwin {
		switch ext {
		case ".pkg":
			p.Commands = append(p.Commands, CommandSpec{Title: "Install macOS pkg " + filepath.Base(path), Cmd: []string{"installer", "-pkg", path, "-target", "/"}, Admin: true})
		case ".dmg":
			sh := fmt.Sprintf("set -e; mount=$(hdiutil attach -nobrowse -readonly %s | awk '/\\/Volumes\\// {for(i=1;i<=NF;i++) if ($i ~ /^\\/Volumes\\//) {print $i; exit}}'); if [ -z \"$mount\" ]; then open %s; exit 0; fi; app=$(find \"$mount\" -maxdepth 2 -name '*.app' -type d | head -n 1); mkdir -p \"$HOME/Applications\"; if [ -n \"$app\" ]; then cp -R \"$app\" \"$HOME/Applications/\"; else open %s; fi; hdiutil detach \"$mount\" >/dev/null || true", shellQuote(path), shellQuote(path), shellQuote(path))
			p.Commands = append(p.Commands, CommandSpec{Title: "Install macOS DMG " + filepath.Base(path), Shell: sh})
		default:
			p.Warnings = append(p.Warnings, "macOS local format пока не имеет безопасной авто-команды: "+ext)
		}
		return
	}
	p.Warnings = append(p.Warnings, "Нет обработчика локального файла: "+path)
}

func (p *Plan) addArchive(path, ext string) {
	dir := filepath.Join(buildDir(), sanitizeName(strings.TrimSuffix(filepath.Base(path), ext)))
	if runtime.GOOS == "windows" {
		if ext == ".zip" {
			ps := fmt.Sprintf("New-Item -ItemType Directory -Force -Path %s | Out-Null; Expand-Archive -Force -Path %s -DestinationPath %s", winPSQuote(dir), winPSQuote(path), winPSQuote(dir))
			p.Commands = append(p.Commands, CommandSpec{Title: "Extract ZIP " + filepath.Base(path), Shell: "powershell -NoProfile -ExecutionPolicy Bypass -Command " + winQuote(ps)})
			return
		}
		if ext == ".7z" {
			p.ensureTool("7z")
			p.Commands = append(p.Commands, CommandSpec{Title: "Extract 7z " + filepath.Base(path), Shell: "7z x -y " + winQuote(path) + " -o" + winQuote(dir)})
			return
		}
		p.Warnings = append(p.Warnings, "Windows archive format требует tar/unzip в PATH: "+ext)
		return
	}
	if ext == ".7z" {
		p.ensureTool("7z")
		sh := fmt.Sprintf("rm -rf %s && mkdir -p %s && 7z x -y %s -o%s && root=$(find %s -mindepth 1 -maxdepth 1 -type d | head -n 1); if [ -n \"$root\" ]; then cd \"$root\"; else cd %s; fi; %s", shellQuote(dir), shellQuote(dir), shellQuote(path), shellQuote(dir), shellQuote(dir), shellQuote(dir), detectBuildShell())
		p.Commands = append(p.Commands, CommandSpec{Title: "Extract and build 7z archive " + filepath.Base(path), Shell: sh})
		return
	}
	sh := fmt.Sprintf("rm -rf %s && mkdir -p %s && case %s in *.zip) unzip -q %s -d %s ;; *) tar -xf %s -C %s ;; esac && root=$(find %s -mindepth 1 -maxdepth 1 -type d | head -n 1); if [ -n \"$root\" ]; then cd \"$root\"; else cd %s; fi; %s", shellQuote(dir), shellQuote(dir), shellQuote(path), shellQuote(path), shellQuote(dir), shellQuote(path), shellQuote(dir), shellQuote(dir), shellQuote(dir), detectBuildShell())
	p.Commands = append(p.Commands, CommandSpec{Title: "Extract and build source archive " + filepath.Base(path), Shell: sh})
}

func isArchiveExt(ext string) bool {
	return ext == ".zip" || ext == ".7z" || strings.HasPrefix(ext, ".tar")
}

func repoName(raw string) string {
	raw = strings.TrimSuffix(raw, ".git")
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	if len(parts) == 0 {
		return "repo"
	}
	return sanitizeName(parts[len(parts)-1])
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "item"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._+@-", r)
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" || out == "." || out == ".." {
		return "item"
	}
	if len(out) > 120 {
		out = strings.Trim(out[:120], ".-")
		if out == "" {
			out = "item"
		}
	}
	return out
}

func winQuote(s string) string {
	s = strings.ReplaceAll(s, "%", "%%")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
func winPSQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func BuildUpdatePlan(items []string, opts Options) Plan {
	sys := Detect()
	p := Plan{System: sys, ContinueOnError: true}
	m := sys.Manager
	flatpakTool := sys.Tools["flatpak"]
	snapTool := sys.Tools["snap"]

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "flatpak:") || strings.HasPrefix(item, "flathub:") {
			id := strings.TrimSpace(item[strings.IndexByte(item, ':')+1:])
			if flatpakTool != "" {
				p.Commands = append(p.Commands, CommandSpec{
					Title: "Update flatpak " + id,
					Cmd:   []string{"flatpak", "update", "-y", id},
				})
			} else {
				p.Warnings = append(p.Warnings, "flatpak not installed, can't update: "+id)
			}
			continue
		}
		if strings.HasPrefix(item, "snap:") {
			id := strings.TrimSpace(item[len("snap:"):])
			if snapTool != "" {
				p.Commands = append(p.Commands, CommandSpec{
					Title: "Update snap " + id,
					Cmd:   []string{"snap", "refresh", id},
					Admin: true,
				})
			} else {
				p.Warnings = append(p.Warnings, "snap not installed, can't update: "+id)
			}
			continue
		}
		if item == "instally" || strings.EqualFold(item, SelfPath()) {
			info := SelfUpdateCheck()
			if info.Available {
				p.Commands = append(p.Commands, CommandSpec{
					Title: "Update Instally",
					Shell: shellQuote(SelfPath()) + " --update-self --yes",
				})
			} else {
				p.Warnings = append(p.Warnings, "Instally already up to date")
			}
			continue
		}
		if m.ID != "none" && len(m.Update) > 0 {
			cmd := append([]string{}, m.Update...)
			cmd = append(cmd, item)
			p.Commands = append(p.Commands, CommandSpec{
				Title:          "Update " + item,
				Cmd:            cmd,
				Admin:          m.NeedsElev,
				TimeoutSeconds: 300,
			})
		} else {
			p.Warnings = append(p.Warnings, "Can't update "+item+": no package manager with update support")
		}
	}
	return p
}

func BuildRemovePlan(items []string, opts Options) Plan {
	sys := Detect()
	p := Plan{System: sys}
	m := sys.Manager
	flatpakTool := sys.Tools["flatpak"]
	snapTool := sys.Tools["snap"]

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "flatpak:") || strings.HasPrefix(item, "flathub:") {
			id := strings.TrimSpace(item[strings.IndexByte(item, ':')+1:])
			if flatpakTool != "" {
				p.Commands = append(p.Commands, CommandSpec{
					Title: "Remove flatpak " + id,
					Cmd:   []string{"flatpak", "uninstall", "-y", id},
				})
			} else {
				p.Warnings = append(p.Warnings, "flatpak not installed, can't remove: "+id)
			}
			continue
		}
		if strings.HasPrefix(item, "snap:") {
			id := strings.TrimSpace(item[len("snap:"):])
			if snapTool != "" {
				p.Commands = append(p.Commands, CommandSpec{
					Title: "Remove snap " + id,
					Cmd:   []string{"snap", "remove", id},
					Admin: true,
				})
			} else {
				p.Warnings = append(p.Warnings, "snap not installed, can't remove: "+id)
			}
			continue
		}
		if m.ID != "none" && len(m.Remove) > 0 {
			cmd := append([]string{}, m.Remove...)
			cmd = append(cmd, item)
			p.Commands = append(p.Commands, CommandSpec{
				Title:          "Remove " + item,
				Cmd:            cmd,
				Admin:          m.NeedsElev,
				TimeoutSeconds: 120,
			})
		} else {
			p.Warnings = append(p.Warnings, "Can't remove "+item+": no package manager with remove support")
		}
	}
	return p
}

func BuildUpgradePlan(opts Options) Plan {
	sys := Detect()
	p := Plan{System: sys}
	m := sys.Manager

	if m.ID == "none" || len(m.Update) == 0 {
		p.Warnings = append(p.Warnings, "Package manager not found or does not support upgrade")
	}
	if m.ID != "none" && len(m.Update) > 0 {
		var cmd []string
		switch m.ID {
		case "apt-get", "apt":
			cmd = []string{m.ID, "upgrade", "-y"}
		case "pacman":
			cmd = []string{"pacman", "-Syu", "--noconfirm"}
		case "dnf", "yum":
			cmd = []string{m.ID, "upgrade", "-y"}
		case "zypper":
			cmd = []string{"zypper", "update", "-y"}
		case "apk":
			cmd = []string{"apk", "upgrade"}
		case "winget":
			cmd = []string{"winget", "upgrade", "--all"}
		case "brew":
			cmd = []string{"brew", "upgrade"}
		case "choco":
			cmd = []string{"choco", "upgrade", "all", "-y"}
		default:
			cmd = append([]string{}, m.Update...)
		}
		if len(cmd) > 0 {
			p.Commands = append(p.Commands, CommandSpec{
				Title:         "Upgrade system packages",
				Cmd:           cmd,
				Admin:         m.NeedsElev,
				TimeoutSeconds: 600,
			})
		}
	}

	if sys.Tools["flatpak"] != "" {
		p.Commands = append(p.Commands, CommandSpec{
			Title: "Upgrade flatpak apps",
			Cmd:   []string{"flatpak", "update", "-y"},
		})
	}

	if sys.Tools["snap"] != "" {
		p.Commands = append(p.Commands, CommandSpec{
			Title: "Refresh snap packages",
			Cmd:   []string{"snap", "refresh"},
			Admin: true,
		})
	}

		info := SelfUpdateCheck()
	if info.Available {
		p.Warnings = append(p.Warnings, "Instally update available: v"+info.Latest+" — run 'instally --update-self'")
		p.Commands = append(p.Commands, CommandSpec{
			Title: "Update Instally",
			Shell: shellQuote(SelfPath()) + " --update-self --yes",
		})
	}

	return p
}

func PurgeCache() int {
	cache := cacheDir()
	cacheReal, err := filepath.EvalSymlinks(cache)
	if err != nil {
		cacheReal = cache
	}
	count := 0
	_ = filepath.Walk(cache, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == cache {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil
		}
		if !strings.HasPrefix(realPath, cacheReal) {
			return nil
		}
		if err := os.Remove(path); err == nil {
			count++
		}
		return nil
	})
	return count
}
