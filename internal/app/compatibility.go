package app

import (
	"fmt"
	"os"
	"strings"
)

type CompatProfile struct {
	Name    string
	Family  string
	OSID    string
	OSLike  string
	Manager string
	Arch    string
}

func CompatibilityProfiles30() []CompatProfile {
	return []CompatProfile{
		{"CachyOS", "linux", "cachyos", "arch", "pacman", "amd64"},
		{"Arch Linux", "linux", "arch", "", "pacman", "amd64"},
		{"Manjaro", "linux", "manjaro", "arch", "pacman", "amd64"},
		{"EndeavourOS", "linux", "endeavouros", "arch", "pacman", "amd64"},
		{"Garuda", "linux", "garuda", "arch", "pacman", "amd64"},
		{"Ubuntu", "linux", "ubuntu", "debian", "apt", "amd64"},
		{"Debian", "linux", "debian", "", "apt", "amd64"},
		{"Linux Mint", "linux", "linuxmint", "ubuntu debian", "apt", "amd64"},
		{"Pop!_OS", "linux", "pop", "ubuntu debian", "apt", "amd64"},
		{"Kali", "linux", "kali", "debian", "apt", "amd64"},
		{"Zorin OS", "linux", "zorin", "ubuntu debian", "apt", "amd64"},
		{"elementary OS", "linux", "elementary", "ubuntu debian", "apt", "amd64"},
		{"Fedora", "linux", "fedora", "", "dnf", "amd64"},
		{"Nobara", "linux", "nobara", "fedora", "dnf", "amd64"},
		{"Rocky Linux", "linux", "rocky", "rhel fedora", "dnf", "amd64"},
		{"AlmaLinux", "linux", "almalinux", "rhel fedora", "dnf", "amd64"},
		{"openSUSE Tumbleweed", "linux", "opensuse-tumbleweed", "suse opensuse", "zypper", "amd64"},
		{"openSUSE Leap", "linux", "opensuse-leap", "suse opensuse", "zypper", "amd64"},
		{"Alpine", "linux", "alpine", "", "apk", "amd64"},
		{"Void Linux", "linux", "void", "", "xbps", "amd64"},
		{"Solus", "linux", "solus", "", "eopkg", "amd64"},
		{"Gentoo", "linux", "gentoo", "", "emerge", "amd64"},
		{"NixOS", "linux", "nixos", "", "nix", "amd64"},
		{"Linuxbrew", "linux", "linuxbrew", "", "brew", "amd64"},
		{"PackageKit fallback", "linux", "unknown", "", "packagekit", "amd64"},
		{"Windows winget", "windows", "windows", "", "winget", "amd64"},
		{"Windows scoop", "windows", "windows", "", "scoop", "amd64"},
		{"Windows Chocolatey", "windows", "windows", "", "choco", "amd64"},
		{"macOS Homebrew Intel", "darwin", "darwin", "", "brew", "amd64"},
		{"macOS Homebrew Apple Silicon", "darwin", "darwin", "", "brew", "arm64"},
	}
}

func CompatibilityMatrixReport() string {
	apps := []string{"vscode", "firefox", "discord", "telegram", "git", "curl", "node", "go", "rust", "python", "java", "docker", "ollama", "opencode", "claude-code"}
	var b strings.Builder
	fmt.Fprintf(&b, "Instally 30-system compatibility matrix (dry-run)\n")
	fmt.Fprintf(&b, "Apps: %s\n\n", strings.Join(apps, ", "))
	passed := 0
	for i, profile := range CompatibilityProfiles30() {
		withCompatEnv(profile, func() {
			plan := BuildPlan(ParseMultiItems(strings.Join(apps, ",")), Options{Yes: true, DryRun: true, AllowUnknown: true, ContinueOnError: true})
			status := "ok"
			if len(plan.Commands) == 0 {
				status = "failed"
			} else {
				passed++
			}
			fmt.Fprintf(&b, "%02d. %-28s %-7s %-10s arch=%-6s commands=%-3d warnings=%-2d %s\n", i+1, profile.Name, profile.Family, profile.Manager, profile.Arch, len(plan.Commands), len(plan.Warnings), status)
		})
	}
	fmt.Fprintf(&b, "\nsummary: passed=%d failed=%d total=%d\n", passed, len(CompatibilityProfiles30())-passed, len(CompatibilityProfiles30()))
	return b.String()
}

func withCompatEnv(p CompatProfile, fn func()) {
	keys := []string{"INSTALLY_FORCE_OS", "INSTALLY_FORCE_OS_ID", "INSTALLY_FORCE_OS_LIKE", "INSTALLY_FORCE_PM", "INSTALLY_FORCE_ARCH", "INSTALLY_SKIP_DNS_PRIVATE_CHECK"}
	old := map[string]string{}
	set := map[string]bool{}
	for _, k := range keys {
		old[k], set[k] = os.LookupEnv(k)
	}
	_ = os.Setenv("INSTALLY_FORCE_OS", p.Family)
	_ = os.Setenv("INSTALLY_FORCE_OS_ID", p.OSID)
	_ = os.Setenv("INSTALLY_FORCE_OS_LIKE", p.OSLike)
	_ = os.Setenv("INSTALLY_FORCE_PM", p.Manager)
	_ = os.Setenv("INSTALLY_FORCE_ARCH", p.Arch)
	_ = os.Setenv("INSTALLY_SKIP_DNS_PRIVATE_CHECK", "1")
	defer func() {
		for _, k := range keys {
			if set[k] {
				_ = os.Setenv(k, old[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()
	fn()
}
