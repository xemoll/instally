package app

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

var installPresets = map[string][]string{
	"base":      {"git", "curl", "wget", "fastfetch", "btop", "htop"},
	"dev":       {"git", "vscode", "node", "go", "rust", "python", "docker", "postman", "dbeaver"},
	"gaming":    {"steam", "heroic", "lutris", "bottles", "prismlauncher", "mangohud", "protonup-qt", "discord"},
	"media":     {"vlc", "mpv", "obs", "kdenlive", "gimp", "krita", "audacity", "handbrake", "blender"},
	"work":      {"firefox", "brave", "libreoffice", "thunderbird", "telegram", "signal", "bitwarden", "nextcloud"},
	"cli":       {"ripgrep", "fd", "bat", "eza", "zoxide", "tealdeer", "delta"},
	"security":  {"keepassxc", "bitwarden", "tailscale", "wireguard", "protonvpn", "tor-browser"},
	"terminals": {"alacritty", "wezterm", "kitty", "neovim", "lazygit", "lazydocker", "yt-dlp"},
}

func presetTasks(items []string) []Task {
	var tasks []Task
	for _, raw := range items {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		list, ok := installPresets[key]
		if !ok {
			tasks = append(tasks, Task{Kind: "pkg", Items: []string{raw}})
			continue
		}
		for _, item := range list {
			tasks = append(tasks, AutoTask(item))
		}
	}
	return mergeTasks(tasks)
}

func PresetApps(name string) []string {
	return installPresets[name]
}

func TasksForPreset(name string) []Task {
	list, ok := installPresets[name]
	if !ok {
		return nil
	}
	var tasks []Task
	for _, item := range list {
		tasks = append(tasks, AutoTask(item))
	}
	return mergeTasks(tasks)
}

func PresetList() []string {
	out := make([]string, 0, len(installPresets))
	for k := range installPresets {
		out = append(out, k)
	}
	return out
}

func PresetListFormatted() string {
	var b strings.Builder
	b.WriteString("Instally presets:\n\n")
	for _, k := range PresetList() {
		list := installPresets[k]
		b.WriteString(fmt.Sprintf("  %-12s %s\n", k, strings.Join(list, ", ")))
	}
	b.WriteString("\nUsage: instally --preset <name>\n")
	return b.String()
}

func KnownAppsList() string {
	var b strings.Builder
	b.WriteString("Instally known apps:\n\n")
	var keys []string
	for k := range knownApps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		app := knownApps[k]
		aliases := ""
		if len(app.Aliases) > 0 {
			aliases = " (" + strings.Join(app.Aliases, ", ") + ")"
		}
		linux := "—"
		if app.Linux.Kind != "" {
			linux = app.Linux.Kind + ": " + strings.Join(app.Linux.Items, ", ")
		}
		b.WriteString(fmt.Sprintf("  %-20s%s\n    linux:%s\n    win:  %s\n    mac:  %s\n\n", k, aliases, linux, app.Windows, app.Mac))
	}
	return b.String()
}

var (
	appVersion = "1.2.2"
	buildDate  = "2026-07-09"
)

func IsVerbose() bool {
	return os.Getenv("INSTALLY_VERBOSE") == "1"
}

func IsQuiet() bool {
	return os.Getenv("INSTALLY_QUIET") == "1"
}

func LatestReleaseNotes() string {
	releases, err := fetchGitHubReleases("xemoll/instally")
	if err != nil {
		return fmt.Sprintf("Error fetching release notes: %v\n", err)
	}
	if len(releases) == 0 {
		return "No releases found.\n"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("== %s (%s) ==\n", releases[0].TagName, releases[0].Name))
	if releases[0].Body != "" {
		b.WriteString(releases[0].Body)
		b.WriteString("\n")
	} else {
		b.WriteString("(no description)\n")
	}
	return b.String()
}

func Changelog() string {
	releases, err := fetchGitHubReleases("xemoll/instally")
	if err != nil {
		return fmt.Sprintf("Error fetching changelog: %v\n", err)
	}
	if len(releases) == 0 {
		return "No releases found.\n"
	}
	var b strings.Builder
	for i, rel := range releases {
		if i >= 5 {
			b.WriteString(fmt.Sprintf("... and %d more releases\n", len(releases)-5))
			break
		}
		b.WriteString(fmt.Sprintf("\n== %s", rel.TagName))
		if rel.Name != "" && rel.Name != rel.TagName {
			b.WriteString(fmt.Sprintf(" — %s", rel.Name))
		}
		b.WriteString(" ==\n")
		if rel.Body != "" {
			body := strings.TrimSpace(rel.Body)
			lines := strings.Split(body, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					b.WriteString(fmt.Sprintf("  %s\n", trimmed))
				}
			}
		}
	}
	return b.String()
}

func VersionInfo() string {
	s := fmt.Sprintf("Instally v%s\n", appVersion)
	ui := SelfUpdateCheck()
	if ui.Available {
		s += fmt.Sprintf("Update available: v%s (run: instally --update-self)\n", ui.Latest)
	}
	return s
}

func BuildInfo() string {
	s := fmt.Sprintf("Instally v%s\nBuild date: %s\nGo version: %s\n", appVersion, buildDate, runtime.Version())
	ui := SelfUpdateCheck()
	if ui.Available {
		s += fmt.Sprintf("Update available: v%s\n", ui.Latest)
	}
	return s
}

func AppStats() string {
	total := len(knownApps)
	byLinux := map[string]int{}
	for _, app := range knownApps {
		switch app.Linux.Kind {
		case "flatpak":
			byLinux["flatpak"]++
		case "pkg":
			byLinux["native"]++
		case "official-firefox":
			byLinux["official"]++
		case "official-discord":
			byLinux["official"]++
		case "github":
			byLinux["github"]++
		default:
			byLinux["other"]++
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Known apps: %d\n\n", total))
	b.WriteString("By Linux install method:\n")
	for _, k := range []string{"flatpak", "native", "github", "official", "other"} {
		if v := byLinux[k]; v > 0 {
			b.WriteString(fmt.Sprintf("  %-10s %d\n", k+":", v))
		}
	}
	b.WriteString(fmt.Sprintf("\nWindows (winget): %d\n", sumWithWin(knownApps)))
	b.WriteString(fmt.Sprintf("macOS (brew):   %d\n", sumWithMac(knownApps)))
	b.WriteString(fmt.Sprintf("GitHub repos:   %d\n", sumWithGH(knownApps)))
	return b.String()
}

func sumWithWin(m map[string]KnownApp) int {
	n := 0
	for _, a := range m {
		if a.Windows != "" {
			n++
		}
	}
	return n
}

func sumWithMac(m map[string]KnownApp) int {
	n := 0
	for _, a := range m {
		if a.Mac != "" {
			n++
		}
	}
	return n
}

func sumWithGH(m map[string]KnownApp) int {
	n := 0
	for _, a := range m {
		if a.GitHub != "" {
			n++
		}
	}
	return n
}
