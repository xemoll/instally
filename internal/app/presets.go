package app

import (
	"fmt"
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

var appVersion = "1.0.0"

func VersionInfo() string {
	return fmt.Sprintf("Instally v%s\n", appVersion)
}
