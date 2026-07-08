package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)



func Which(appName string) string {
	sys := Detect()
	key, ok := knownAppKey(appName)
	if !ok {
		return fmt.Sprintf("App %q not in known apps list", appName)
	}
	ka := knownApps[key]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("App: %s (%s)\n", ka.Name, key))
	if len(ka.Aliases) > 0 {
		b.WriteString(fmt.Sprintf("Aliases: %s\n", strings.Join(ka.Aliases, ", ")))
	}
	b.WriteString("\n")

	p := commandExists(appName)
	if p == "" {
		p = commandExists(key)
	}
	if p != "" {
		b.WriteString(fmt.Sprintf("Binary: %s\n", p))
	}

	switch sys.Family {
	case Linux:
		if ka.Linux.Kind != "" {
			b.WriteString(fmt.Sprintf("Install method: %s\n", ka.Linux.Kind))
			b.WriteString(fmt.Sprintf("Target: %s\n", strings.Join(ka.Linux.Items, ", ")))
		}
		for _, check := range ka.Linux.Items {
			if bp := commandExists(check); bp != "" {
				b.WriteString(fmt.Sprintf("Installed: yes (%s)\n", bp))
				version := getVersionShort(bp)
				if version != "" {
					b.WriteString(fmt.Sprintf("Version: %s\n", version))
				}
			}
		}
		if ka.GitHub != "" {
			b.WriteString(fmt.Sprintf("GitHub fallback: %s\n", ka.GitHub))
		}
	case Windows:
		b.WriteString(fmt.Sprintf("Winget ID: %s\n", ka.Windows))
		b.WriteString("Installed: winget show needed for exact check\n")
	case Darwin:
		if ka.Mac != "" {
			b.WriteString(fmt.Sprintf("Brew formula: %s (cask: %v)\n", ka.Mac, ka.MacCask))
		}
	}

	return b.String()
}

func Why(app string) string {
	sys := Detect()
	key, ok := knownAppKey(app)
	if !ok {
		return fmt.Sprintf("Why %q:\n  Not in known-apps → will try native package manager with same name\n", app)
	}
	ka := knownApps[key]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Why %q (%s):\n", app, ka.Name))

	switch sys.Family {
	case Linux:
		if ka.Linux.Kind != "" {
			switch ka.Linux.Kind {
			case "official-firefox":
				b.WriteString("  Reason: official Mozilla APT repo or Flatpak\n")
				b.WriteString("  This method was chosen because Firefox has native APT repo with automatic updates\n")
			case "official-discord":
				b.WriteString("  Reason: direct .deb/.rpm download from discord.com\n")
				b.WriteString("  Native packages are used because Flatpak Discord has known input issues\n")
			case "flatpak":
				b.WriteString(fmt.Sprintf("  Reason: Flatpak from Flathub (%s)\n", strings.Join(ka.Linux.Items, ", ")))
				b.WriteString("  Flatpak provides sandboxed, up-to-date packages across all distros\n")
			case "pkg":
				b.WriteString(fmt.Sprintf("  Reason: native package (%s via %s)\n", strings.Join(ka.Linux.Items, ", "), sys.Manager.ID))
				b.WriteString(fmt.Sprintf("  %s package is used for best performance and integration\n", sys.Manager.Label))
			case "github":
				b.WriteString(fmt.Sprintf("  Reason: GitHub releases (%s)\n", ka.GitHub))
				b.WriteString("  GitHub release assets are used when no package exists for this distro\n")
			case "pipx":
				b.WriteString("  Reason: pipx (Python tool)\n")
				b.WriteString("  pipx keeps the tool in its own isolated environment\n")
			}
		} else if ka.GitHub != "" {
			b.WriteString(fmt.Sprintf("  Reason: GitHub release (%s) — no Linux package registered\n", ka.GitHub))
		} else {
			b.WriteString("  Reason: no registered install method — will pass name directly to package manager\n")
		}
	case Windows:
		b.WriteString(fmt.Sprintf("  Reason: Winget (%s)\n", ka.Windows))
	case Darwin:
		b.WriteString(fmt.Sprintf("  Reason: Homebrew (%s, cask: %v)\n", ka.Mac, ka.MacCask))
	}

	return b.String()
}

func SearchPackages(query string) string {
	sys := Detect()
	m := sys.Manager
	if m.ID == "none" || len(m.Search) == 0 {
		return fmt.Sprintf("No search command available for %s\n", m.Label)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, m.Search[0], append(m.Search[1:], query)...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("Search failed: %s\n", err)
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) > 30 {
		lines = lines[:30]
	}
	return fmt.Sprintf("Search results for %q (top %d):\n\n%s\n", query, len(lines), strings.Join(lines, "\n"))
}

func VerifyInstalled(apps []string) string {
	sys := Detect()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Verifying %d app(s) on %s (%s):\n\n", len(apps), sys.OSID, sys.Manager.ID))

	for _, app := range apps {
		key, ok := knownAppKey(app)
		if !ok {
			b.WriteString(fmt.Sprintf("? %-20s — not in known apps, checking as binary...\n", app))
			if p := commandExists(app); p != "" {
				b.WriteString(fmt.Sprintf("  ✓ found at %s\n", p))
			} else {
				b.WriteString(fmt.Sprintf("  ✗ not found\n"))
			}
			continue
		}

		ka := knownApps[key]
		expected := ""
		switch sys.Family {
		case Linux:
			if ka.Linux.Kind != "" {
				expected = strings.Join(ka.Linux.Items, ", ")
			}
		case Windows:
			expected = ka.Windows
		case Darwin:
			expected = ka.Mac
		}

		found := false
		for _, check := range strings.FieldsFunc(expected, func(r rune) bool { return r == ',' || r == ' ' }) {
			check = strings.TrimSpace(check)
			if check == "" {
				continue
			}
			if p := commandExists(check); p != "" {
				version := getVersionShort(p)
				b.WriteString(fmt.Sprintf("✓ %-20s %s (%s)\n", ka.Name, p, version))
				found = true
				break
			}
		}
		if !found {
			b.WriteString(fmt.Sprintf("✗ %-20s expected: %s\n", ka.Name, expected))
		}
	}
	return b.String()
}

func ExportPlan(tasks []Task, opts Options, file string) error {
	plan := BuildPlan(tasks, opts)
	data := JSON(plan)
	return os.WriteFile(file, []byte(data), 0o644)
}

func EnvReport() string {
	var b strings.Builder
	b.WriteString("Instally environment variables:\n\n")
	type envVar struct {
		name   string
		value  string
		source string
	}
	vars := []envVar{
		{"INSTALLY_CACHE_DIR", os.Getenv("INSTALLY_CACHE_DIR"), "cache directory override"},
		{"INSTALLY_DATA_DIR", os.Getenv("INSTALLY_DATA_DIR"), "data directory override"},
		{"INSTALLY_BUILD_DIR", os.Getenv("INSTALLY_BUILD_DIR"), "build directory override"},
		{"INSTALLY_KEY", maskEnv("INSTALLY_KEY"), "golden key for FunPay"},
		{"INSTALLY_FORCE_OS", os.Getenv("INSTALLY_FORCE_OS"), "force OS (linux/darwin/windows)"},
		{"INSTALLY_FORCE_OS_ID", os.Getenv("INSTALLY_FORCE_OS_ID"), "force distro ID"},
		{"INSTALLY_FORCE_OS_LIKE", os.Getenv("INSTALLY_FORCE_OS_LIKE"), "force distro family"},
		{"INSTALLY_FORCE_PM", os.Getenv("INSTALLY_FORCE_PM"), "force package manager"},
		{"INSTALLY_FORCE_ARCH", os.Getenv("INSTALLY_FORCE_ARCH"), "force architecture"},
		{"INSTALLY_LANG", os.Getenv("INSTALLY_LANG"), "language (ru/en)"},
		{"INSTALLY_COMMAND_TIMEOUT_SECONDS", os.Getenv("INSTALLY_COMMAND_TIMEOUT_SECONDS"), "per-command timeout"},
		{"INSTALLY_REFRESH_TIMEOUT_SECONDS", os.Getenv("INSTALLY_REFRESH_TIMEOUT_SECONDS"), "metadata refresh timeout"},
		{"INSTALLY_FLATPAK_SYSTEM", os.Getenv("INSTALLY_FLATPAK_SYSTEM"), "use system-wide flatpak"},
		{"INSTALLY_VT_KEY_FILE", maskEnv("INSTALLY_VT_KEY_FILE"), "VT key file path"},
		{"INSTALLY_ALLOW_PRIVATE_URLS", os.Getenv("INSTALLY_ALLOW_PRIVATE_URLS"), "allow private LAN URLs"},
	}

	for _, v := range vars {
		if v.value == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %-40s %s\n    source: %s\n\n", v.name, v.value, v.source))
	}
	if b.Len() == 0 {
		b.WriteString("  (none set)\n")
	}
	return b.String()
}

func FixBroken() string {
	sys := Detect()
	m := sys.Manager
	var b strings.Builder
	b.WriteString("Attempting to fix broken package manager state...\n\n")

	scripts := map[string]string{
		"apt":     "dpkg --configure -a && apt-get install -f -y",
		"apt-get":  "dpkg --configure -a && apt-get install -f -y",
		"pacman":  "pacman -Syu --noconfirm || pacman --sync --refresh --sysupgrade --noconfirm",
		"dnf":     "dnf check && dnf reinstall -y $(dnf repoquery --unsatisfied 2>/dev/null || echo '') || dnf distro-sync -y",
		"zypper":  "zypper --non-interactive verify",
		"apk":     "apk fix",
		"winget":   "winget source reset --force",
		"brew":    "brew doctor && brew upgrade",
	}

	script, ok := scripts[m.ID]
	if !ok {
		return fmt.Sprintf("No fix-known script for %s\n", m.ID)
	}

	cmd := exec.Command("/bin/sh", "-c", script)
	out, err := cmd.CombinedOutput()
	b.WriteString(fmt.Sprintf("Fix command: %s\n", script))
	if err != nil {
		b.WriteString(fmt.Sprintf("Fix completed with non-zero exit: %s\n", err))
	}
	b.WriteString(fmt.Sprintf("Output:\n%s\n", out))
	return b.String()
}

func AutoComplete(shell string) string {
	cmds := []string{
		"--help", "--version", "--build-info", "--stats", "--detect", "--doctor", "--support",
		"--list-apps", "--list-presets", "--gui",
		"--dry-run", "--yes", "--continue-on-error",
		"--update", "--upgrade-all", "--purge-cache", "--fix-broken",
		"--search", "--which", "--why", "--verify-installed", "--depends",
		"--env", "--export-plan",
		"--lang",
		"--pkg", "--flatpak", "--snap", "--pipx", "--npm", "--cargo", "--go",
		"--git", "--github", "--release", "--url", "--local", "--multi", "--preset",
		"--batch", "--text", "--scan",
	}

	switch shell {
	case "bash":
		return bashCompletion(cmds)
	case "zsh":
		return zshCompletion(cmds)
	default:
		return bashCompletion(cmds)
	}
}

func bashCompletion(cmds []string) string {
	var b strings.Builder
	b.WriteString("# instally bash completion\n")
	b.WriteString("_instally() {\n")
	b.WriteString("  local cur=${COMP_WORDS[COMP_CWORD]}\n")
	b.WriteString("  local prev=${COMP_WORDS[COMP_CWORD-1]}\n")
	b.WriteString("  local opts=\"")
	b.WriteString(strings.Join(cmds, " "))
	b.WriteString("\"\n")
	b.WriteString("  COMPREPLY=($(compgen -W \"$opts\" -- \"$cur\"))\n")
	b.WriteString("}\n")
	b.WriteString("complete -F _instally instally\n")
	return b.String()
}

func zshCompletion(cmds []string) string {
	var b strings.Builder
	b.WriteString("# compinstally zsh completion\n")
	b.WriteString("#compinst _instally\n\n")
	b.WriteString("_instally() {\n")
	b.WriteString("  local -a opts\n")
	for _, c := range cmds {
		b.WriteString(fmt.Sprintf("  opts+=('%s')\n", c))
	}
	b.WriteString("  _describe 'instally' opts\n")
	b.WriteString("}\n\n")
	b.WriteString("compdef _instally instally\n")
	return b.String()
}

func maskEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		return ""
	}
	if len(v) > 8 {
		return v[:4] + "****" + v[len(v)-4:]
	}
	return "****"
}

func getVersionShort(bin string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	out, _ := cmd.Output()
	line := strings.SplitN(string(out), "\n", 2)[0]
	line = strings.TrimSpace(line)
	if len(line) > 60 {
		line = line[:60] + "..."
	}
	return line
}