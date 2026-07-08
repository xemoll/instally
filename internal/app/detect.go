package app

import (
	"bufio"
	"os"
	"runtime"
	"sort"
	"strings"
)

func profiles() map[string]Manager {
	return map[string]Manager{
		"pacman":     {ID: "pacman", Label: "Arch/CachyOS/Manjaro", Family: Linux, Tools: []string{"pacman"}, Install: []string{"pacman", "-S", "--needed"}, Yes: []string{"--noconfirm"}, Update: []string{"pacman", "-Syu"}, Search: []string{"pacman", "-Ss"}, Info: []string{"pacman", "-Si"}, Local: map[string][]string{".pkg.tar.zst": {"pacman", "-U"}, ".pkg.tar.xz": {"pacman", "-U"}, ".pkg.tar.gz": {"pacman", "-U"}}, Prepare: []string{"base-devel", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "go", "npm", "flatpak"}, Priority: 100, NeedsElev: true},
		"apt":        {ID: "apt", Label: "Debian/Ubuntu/Mint/Pop/Kali", Family: Linux, Tools: []string{"apt-get", "apt"}, Install: []string{"apt-get", "install"}, Yes: []string{"-y"}, Update: []string{"apt-get", "update"}, Search: []string{"apt-cache", "search"}, Info: []string{"apt-cache", "show"}, Local: map[string][]string{".deb": {"apt", "install"}}, Prepare: []string{"build-essential", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja-build", "cargo", "golang-go", "npm", "flatpak"}, Priority: 90, NeedsElev: true},
		"dnf":        {ID: "dnf", Label: "Fedora/Nobara/RHEL-like", Family: Linux, Tools: []string{"dnf5", "dnf"}, Install: []string{"dnf", "install"}, Yes: []string{"-y"}, Update: []string{"dnf", "upgrade"}, Search: []string{"dnf", "search"}, Info: []string{"dnf", "info"}, Local: map[string][]string{".rpm": {"dnf", "install"}}, Prepare: []string{"@development-tools", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja-build", "rust", "cargo", "golang", "npm", "flatpak"}, Priority: 80, NeedsElev: true},
		"zypper":     {ID: "zypper", Label: "openSUSE", Family: Linux, Tools: []string{"zypper"}, Install: []string{"zypper", "install"}, Yes: []string{"--non-interactive"}, Update: []string{"zypper", "refresh"}, Search: []string{"zypper", "search"}, Info: []string{"zypper", "info"}, Local: map[string][]string{".rpm": {"zypper", "install"}}, Prepare: []string{"patterns-devel-base-devel_basis", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "cargo", "go", "npm", "flatpak"}, Priority: 70, NeedsElev: true},
		"apk":        {ID: "apk", Label: "Alpine", Family: Linux, Tools: []string{"apk"}, Install: []string{"apk", "add", "--no-cache"}, Yes: nil, Update: []string{"apk", "update"}, Search: []string{"apk", "search"}, Info: []string{"apk", "info"}, Local: map[string][]string{".apk": {"apk", "add", "--allow-untrusted"}}, Prepare: []string{"build-base", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "cargo", "go", "npm", "flatpak"}, Priority: 60, NeedsElev: true},
		"xbps":       {ID: "xbps", Label: "Void Linux", Family: Linux, Tools: []string{"xbps-install"}, Install: []string{"xbps-install", "-S"}, Yes: []string{"-y"}, Update: []string{"xbps-install", "-Syu"}, Search: []string{"xbps-query", "-Rs"}, Info: []string{"xbps-query", "-R"}, Local: map[string][]string{}, Prepare: []string{"base-devel", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "cargo", "go", "nodejs", "flatpak"}, Priority: 50, NeedsElev: true},
		"eopkg":      {ID: "eopkg", Label: "Solus", Family: Linux, Tools: []string{"eopkg"}, Install: []string{"eopkg", "install"}, Yes: []string{"-y"}, Update: []string{"eopkg", "ur"}, Search: []string{"eopkg", "search"}, Info: []string{"eopkg", "info"}, Local: map[string][]string{}, Prepare: []string{"system.devel", "git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "golang", "nodejs", "flatpak"}, Priority: 45, NeedsElev: true},
		"emerge":     {ID: "emerge", Label: "Gentoo", Family: Linux, Tools: []string{"emerge"}, Install: []string{"emerge", "--ask=n"}, Yes: nil, Update: []string{"emerge", "--sync"}, Search: []string{"emerge", "--search"}, Info: []string{"emerge", "--info"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget", "tar", "unzip", "cmake", "meson", "ninja", "rust", "go", "npm"}, Priority: 40, NeedsElev: true},
		"nix":        {ID: "nix", Label: "Nix/NixOS", Family: Linux, Tools: []string{"nix"}, Install: []string{"nix", "profile", "install", "nixpkgs#"}, Yes: nil, Update: []string{"nix", "flake", "update"}, Search: []string{"nix", "search", "nixpkgs"}, Info: []string{"nix", "profile", "list"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget"}, Priority: 35, NeedsElev: false},
		"packagekit": {ID: "packagekit", Label: "PackageKit fallback", Family: Linux, Tools: []string{"pkcon"}, Install: []string{"pkcon", "install"}, Yes: []string{"-y"}, Update: []string{"pkcon", "refresh"}, Search: []string{"pkcon", "search", "name"}, Info: []string{"pkcon", "details"}, Local: map[string][]string{".deb": {"pkcon", "install-local"}, ".rpm": {"pkcon", "install-local"}}, Prepare: []string{"git", "curl", "wget", "tar", "unzip", "cmake"}, Priority: 10, NeedsElev: false},
		"brew":       {ID: "brew", Label: "Homebrew/Linuxbrew/macOS", Family: Darwin, Tools: []string{"brew"}, Install: []string{"brew", "install"}, Yes: nil, Update: []string{"brew", "update"}, Search: []string{"brew", "search"}, Info: []string{"brew", "info"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget", "cmake", "rust", "go", "node"}, Priority: 100, NeedsElev: false},
		"port":       {ID: "port", Label: "MacPorts", Family: Darwin, Tools: []string{"port"}, Install: []string{"port", "install"}, Yes: nil, Update: []string{"port", "selfupdate"}, Search: []string{"port", "search"}, Info: []string{"port", "info"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget"}, Priority: 60, NeedsElev: true},
		"winget":     {ID: "winget", Label: "Windows Package Manager", Family: Windows, Tools: []string{"winget"}, Install: []string{"winget", "install"}, Yes: []string{"--silent", "--accept-package-agreements", "--accept-source-agreements"}, Update: []string{"winget", "upgrade"}, Search: []string{"winget", "search"}, Info: []string{"winget", "show"}, Local: map[string][]string{}, Prepare: []string{"Git.Git", "Microsoft.PowerShell", "Python.Python.3.12", "OpenJS.NodeJS", "GoLang.Go", "Rustlang.Rustup"}, Priority: 100, NeedsElev: false},
		"scoop":      {ID: "scoop", Label: "Scoop", Family: Windows, Tools: []string{"scoop"}, Install: []string{"scoop", "install"}, Yes: nil, Update: []string{"scoop", "update"}, Search: []string{"scoop", "search"}, Info: []string{"scoop", "info"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget", "go", "rust", "nodejs"}, Priority: 80, NeedsElev: false},
		"choco":      {ID: "choco", Label: "Chocolatey", Family: Windows, Tools: []string{"choco"}, Install: []string{"choco", "install"}, Yes: []string{"-y"}, Update: []string{"choco", "upgrade", "all"}, Search: []string{"choco", "search"}, Info: []string{"choco", "info"}, Local: map[string][]string{}, Prepare: []string{"git", "curl", "wget", "golang", "rust", "nodejs"}, Priority: 70, NeedsElev: true},
	}
}

func Detect() SystemInfo {
	family := Family(runtime.GOOS)
	if family != Linux && family != Windows && family != Darwin {
		family = Unknown
	}
	osid, like := osRelease()
	if forcedID := strings.TrimSpace(os.Getenv("INSTALLY_FORCE_OS_ID")); forcedID != "" {
		osid = strings.ToLower(forcedID)
	}
	if forcedLike := strings.TrimSpace(os.Getenv("INSTALLY_FORCE_OS_LIKE")); forcedLike != "" {
		like = strings.ToLower(forcedLike)
	}
	forcedOS := strings.TrimSpace(os.Getenv("INSTALLY_FORCE_OS"))
	if forcedOS != "" {
		family = Family(forcedOS)
	}
	all := profiles()
	forcedPM := strings.TrimSpace(os.Getenv("INSTALLY_FORCE_PM"))
	manager, found, toolPath := chooseManager(family, osid, like, forcedPM, all)
	tools := map[string]string{}
	for _, t := range []string{"git", "curl", "wget", "tar", "unzip", "7z", "yara", "flatpak", "snap", "npm", "node", "cargo", "go", "pipx", "clamscan", "clamdscan", "xdg-mime", "update-desktop-database", "powershell", "pwsh", "hdiutil", "open"} {
		if p := commandExists(t); p != "" {
			tools[t] = p
		}
	}
	arch := runtime.GOARCH
	if forcedArch := strings.TrimSpace(os.Getenv("INSTALLY_FORCE_ARCH")); forcedArch != "" {
		arch = forcedArch
	}
	return SystemInfo{Family: family, GOOS: runtime.GOOS, Arch: arch, OSID: osid, OSLike: like, Manager: manager, ManagerFound: found, ToolPath: toolPath, Tools: tools, Home: homeDir(), BuildDir: buildDir(), DataDir: dataDir(), CacheDir: cacheDir()}
}

func osRelease() (string, string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return strings.ToLower(runtime.GOOS), ""
	}
	defer f.Close()
	m := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		v := strings.Trim(parts[1], "\"'")
		m[strings.ToUpper(parts[0])] = strings.ToLower(v)
	}
	return m["ID"], m["ID_LIKE"]
}

func chooseManager(family Family, osid, like, forced string, all map[string]Manager) (Manager, bool, string) {
	if forced != "" {
		m := all[forced]
		if m.ID == "" {
			m = Manager{ID: forced, Label: forced, Family: family, Tools: []string{forced}, Install: []string{forced, "install"}, Priority: 1}
		}
		return m, commandExists(firstTool(m)) != "", commandExists(firstTool(m))
	}
	preferred := preferredPM(family, osid, like)
	if preferred != "" {
		if m, ok := all[preferred]; ok {
			if p := commandExists(firstTool(m)); p != "" || family == Linux || family == Darwin || family == Windows {
				return m, p != "", p
			}
		}
	}
	list := make([]Manager, 0, len(all))
	for _, m := range all {
		if m.Family == family || (family == Linux && m.ID == "brew") {
			list = append(list, m)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Priority > list[j].Priority })
	for _, m := range list {
		for _, t := range m.Tools {
			if p := commandExists(t); p != "" {
				return m, true, p
			}
		}
	}
	return Manager{ID: "none", Label: "No package manager detected", Family: family, Local: map[string][]string{}}, false, ""
}

func preferredPM(family Family, osid, like string) string {
	if family == Windows {
		return "winget"
	}
	if family == Darwin {
		return "brew"
	}
	idMap := map[string]string{"arch": "pacman", "cachyos": "pacman", "manjaro": "pacman", "endeavouros": "pacman", "garuda": "pacman", "artix": "pacman", "debian": "apt", "ubuntu": "apt", "linuxmint": "apt", "pop": "apt", "kali": "apt", "zorin": "apt", "elementary": "apt", "fedora": "dnf", "nobara": "dnf", "rhel": "dnf", "centos": "dnf", "rocky": "dnf", "almalinux": "dnf", "opensuse": "zypper", "opensuse-tumbleweed": "zypper", "opensuse-leap": "zypper", "suse": "zypper", "alpine": "apk", "void": "xbps", "solus": "eopkg", "gentoo": "emerge", "nixos": "nix", "asahi": "pacman", "blackarch": "pacman", "parrot": "apt", "mx": "apt", "deepin": "apt", "tuxedo": "apt", "vanilla": "apt", "blendos": "pacman", "ultramarine": "dnf", "bazzite": "dnf", "bluefin": "dnf", "opensuse-microos": "zypper"}
	if v := idMap[osid]; v != "" {
		return v
	}
	likeMap := map[string]string{"arch": "pacman", "debian": "apt", "ubuntu": "apt", "fedora": "dnf", "rhel": "dnf", "suse": "zypper", "opensuse": "zypper", "alpine": "apk", "void": "xbps", "gentoo": "emerge"}
	for _, item := range strings.Fields(like) {
		if v := likeMap[item]; v != "" {
			return v
		}
	}
	return ""
}

func firstTool(m Manager) string {
	if len(m.Tools) > 0 {
		return m.Tools[0]
	}
	if len(m.Install) > 0 {
		return m.Install[0]
	}
	return m.ID
}
