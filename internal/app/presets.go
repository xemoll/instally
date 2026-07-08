package app

import "strings"

var installPresets = map[string][]string{
	"base":      {"git", "curl", "wget", "fastfetch", "btop", "htop"},
	"dev":       {"git", "vscode", "node", "go", "rust", "python", "docker", "postman", "dbeaver"},
	"gaming":    {"steam", "heroic", "lutris", "bottles", "prismlauncher", "mangohud", "protonup-qt", "discord"},
	"media":     {"vlc", "mpv", "obs", "kdenlive", "gimp", "krita", "audacity", "handbrake", "blender"},
	"work":      {"firefox", "brave", "libreoffice", "thunderbird", "telegram", "signal", "bitwarden", "nextcloud"},
	"ai":        {"opencode", "ollama", "claude-code", "zed", "cursor"},
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
