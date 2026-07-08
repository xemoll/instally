package app

import (
	"fmt"
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
	appVersion = "1.1.0"
	buildDate  = "2026-07-08"
)

func VersionInfo() string {
	return fmt.Sprintf("Instally v%s\n", appVersion)
}

func BuildInfo() string {
	return fmt.Sprintf("Instally v%s\nBuild date: %s\nGo version: %s\n", appVersion, buildDate, runtime.Version())
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
