package app

import "strings"

type KnownApp struct {
	Name    string
	Aliases []string
	Linux   Task
	Windows string
	Mac     string
	MacCask bool
	GitHub  string
}

var knownApps = map[string]KnownApp{
	"vscode":         {Name: "Visual Studio Code", Aliases: []string{"code", "visual-studio-code"}, Linux: Task{Kind: "flatpak", Items: []string{"com.visualstudio.code"}}, Windows: "Microsoft.VisualStudioCode", Mac: "visual-studio-code", MacCask: true},
	"vscodium":       {Name: "VSCodium", Aliases: []string{"codium"}, Linux: Task{Kind: "flatpak", Items: []string{"com.vscodium.codium"}}, Windows: "VSCodium.VSCodium", Mac: "vscodium", MacCask: true},
	"discord":        {Name: "Discord", Linux: Task{Kind: "official-discord", Items: []string{"discord"}}, Windows: "Discord.Discord", Mac: "discord", MacCask: true},
	"telegram":       {Name: "Telegram", Aliases: []string{"telegram-desktop"}, Linux: Task{Kind: "flatpak", Items: []string{"org.telegram.desktop"}}, Windows: "Telegram.TelegramDesktop", Mac: "telegram", MacCask: true},
	"spotify":        {Name: "Spotify", Linux: Task{Kind: "flatpak", Items: []string{"com.spotify.Client"}}, Windows: "Spotify.Spotify", Mac: "spotify", MacCask: true},
	"steam":          {Name: "Steam", Linux: Task{Kind: "flatpak", Items: []string{"com.valvesoftware.Steam"}}, Windows: "Valve.Steam", Mac: "steam", MacCask: true},
	"heroic":         {Name: "Heroic Games Launcher", Linux: Task{Kind: "flatpak", Items: []string{"com.heroicgameslauncher.hgl"}}, Windows: "HeroicGamesLauncher.HeroicGamesLauncher", Mac: "heroic", MacCask: true, GitHub: "Heroic-Games-Launcher/HeroicGamesLauncher"},
	"bottles":        {Name: "Bottles", Linux: Task{Kind: "flatpak", Items: []string{"com.usebottles.bottles"}}, GitHub: "bottlesdevs/Bottles"},
	"prismlauncher":  {Name: "Prism Launcher", Aliases: []string{"prism"}, Linux: Task{Kind: "flatpak", Items: []string{"org.prismlauncher.PrismLauncher"}}, Windows: "PrismLauncher.PrismLauncher", Mac: "prismlauncher", MacCask: true, GitHub: "PrismLauncher/PrismLauncher"},
	"obs":            {Name: "OBS Studio", Aliases: []string{"obs-studio"}, Linux: Task{Kind: "flatpak", Items: []string{"com.obsproject.Studio"}}, Windows: "OBSProject.OBSStudio", Mac: "obs", MacCask: true},
	"vlc":            {Name: "VLC", Linux: Task{Kind: "flatpak", Items: []string{"org.videolan.VLC"}}, Windows: "VideoLAN.VLC", Mac: "vlc", MacCask: true},
	"blender":        {Name: "Blender", Linux: Task{Kind: "flatpak", Items: []string{"org.blender.Blender"}}, Windows: "BlenderFoundation.Blender", Mac: "blender", MacCask: true},
	"gimp":           {Name: "GIMP", Linux: Task{Kind: "flatpak", Items: []string{"org.gimp.GIMP"}}, Windows: "GIMP.GIMP", Mac: "gimp", MacCask: true},
	"krita":          {Name: "Krita", Linux: Task{Kind: "flatpak", Items: []string{"org.kde.krita"}}, Windows: "KDE.Krita", Mac: "krita", MacCask: true},
	"firefox":        {Name: "Firefox", Linux: Task{Kind: "official-firefox", Items: []string{"firefox"}}, Windows: "Mozilla.Firefox", Mac: "firefox", MacCask: true},
	"brave":          {Name: "Brave Browser", Linux: Task{Kind: "flatpak", Items: []string{"com.brave.Browser"}}, Windows: "Brave.Brave", Mac: "brave-browser", MacCask: true},
	"chrome":         {Name: "Google Chrome", Aliases: []string{"google-chrome"}, Linux: Task{Kind: "flatpak", Items: []string{"com.google.Chrome"}}, Windows: "Google.Chrome", Mac: "google-chrome", MacCask: true},
	"godot":          {Name: "Godot", Linux: Task{Kind: "flatpak", Items: []string{"org.godotengine.Godot"}}, Windows: "GodotEngine.GodotEngine", Mac: "godot", MacCask: true, GitHub: "godotengine/godot"},
	"neovim":         {Name: "Neovim", Aliases: []string{"nvim"}, Linux: Task{Kind: "pkg", Items: []string{"neovim"}}, Windows: "Neovim.Neovim", Mac: "neovim"},
	"docker":         {Name: "Docker", Linux: Task{Kind: "pkg", Items: []string{"docker"}}, Windows: "Docker.DockerDesktop", Mac: "docker", MacCask: true},
	"node":           {Name: "Node.js", Aliases: []string{"nodejs"}, Linux: Task{Kind: "pkg", Items: []string{"nodejs", "npm"}}, Windows: "OpenJS.NodeJS", Mac: "node"},
	"go":             {Name: "Go", Aliases: []string{"golang"}, Linux: Task{Kind: "pkg", Items: []string{"go"}}, Windows: "GoLang.Go", Mac: "go"},
	"rust":           {Name: "Rust", Aliases: []string{"cargo"}, Linux: Task{Kind: "pkg", Items: []string{"rust"}}, Windows: "Rustlang.Rustup", Mac: "rust"},
	"alacritty":      {Name: "Alacritty", Linux: Task{Kind: "pkg", Items: []string{"alacritty"}}, Windows: "Alacritty.Alacritty", Mac: "alacritty", MacCask: true, GitHub: "alacritty/alacritty"},
	"wezterm":        {Name: "WezTerm", Linux: Task{Kind: "flatpak", Items: []string{"org.wezfurlong.wezterm"}}, Windows: "wez.wezterm", Mac: "wezterm", MacCask: true, GitHub: "wez/wezterm"},
	"kitty":          {Name: "Kitty", Linux: Task{Kind: "pkg", Items: []string{"kitty"}}, Mac: "kitty", MacCask: true, GitHub: "kovidgoyal/kitty"},
	"obsidian":       {Name: "Obsidian", Linux: Task{Kind: "flatpak", Items: []string{"md.obsidian.Obsidian"}}, Windows: "Obsidian.Obsidian", Mac: "obsidian", MacCask: true},
	"bitwarden":      {Name: "Bitwarden", Linux: Task{Kind: "flatpak", Items: []string{"com.bitwarden.desktop"}}, Windows: "Bitwarden.Bitwarden", Mac: "bitwarden", MacCask: true},
	"keepassxc":      {Name: "KeePassXC", Linux: Task{Kind: "flatpak", Items: []string{"org.keepassxc.KeePassXC"}}, Windows: "KeePassXCTeam.KeePassXC", Mac: "keepassxc", MacCask: true},
	"signal":         {Name: "Signal", Linux: Task{Kind: "flatpak", Items: []string{"org.signal.Signal"}}, Windows: "OpenWhisperSystems.Signal", Mac: "signal", MacCask: true},
	"slack":          {Name: "Slack", Linux: Task{Kind: "flatpak", Items: []string{"com.slack.Slack"}}, Windows: "SlackTechnologies.Slack", Mac: "slack", MacCask: true},
	"zoom":           {Name: "Zoom", Linux: Task{Kind: "flatpak", Items: []string{"us.zoom.Zoom"}}, Windows: "Zoom.Zoom", Mac: "zoom", MacCask: true},
	"postman":        {Name: "Postman", Linux: Task{Kind: "flatpak", Items: []string{"com.getpostman.Postman"}}, Windows: "Postman.Postman", Mac: "postman", MacCask: true},
	"insomnia":       {Name: "Insomnia", Linux: Task{Kind: "flatpak", Items: []string{"rest.insomnia.Insomnia"}}, Windows: "Kong.Insomnia", Mac: "insomnia", MacCask: true},
	"github-desktop": {Name: "GitHub Desktop", Aliases: []string{"githubdesktop"}, Linux: Task{Kind: "github", Items: []string{"shiftkey/desktop"}}, Windows: "GitHub.GitHubDesktop", Mac: "github", MacCask: true, GitHub: "shiftkey/desktop"},
	"qbittorrent":    {Name: "qBittorrent", Aliases: []string{"qb"}, Linux: Task{Kind: "flatpak", Items: []string{"org.qbittorrent.qBittorrent"}}, Windows: "qBittorrent.qBittorrent", Mac: "qbittorrent", MacCask: true},
	"inkscape":       {Name: "Inkscape", Linux: Task{Kind: "flatpak", Items: []string{"org.inkscape.Inkscape"}}, Windows: "Inkscape.Inkscape", Mac: "inkscape", MacCask: true},
	"kdenlive":       {Name: "Kdenlive", Linux: Task{Kind: "flatpak", Items: []string{"org.kde.kdenlive"}}, Windows: "KDE.Kdenlive", Mac: "kdenlive", MacCask: true},
	"audacity":       {Name: "Audacity", Linux: Task{Kind: "flatpak", Items: []string{"org.audacityteam.Audacity"}}, Windows: "Audacity.Audacity", Mac: "audacity", MacCask: true},
	"libreoffice":    {Name: "LibreOffice", Linux: Task{Kind: "flatpak", Items: []string{"org.libreoffice.LibreOffice"}}, Windows: "TheDocumentFoundation.LibreOffice", Mac: "libreoffice", MacCask: true},
	"thunderbird":    {Name: "Thunderbird", Linux: Task{Kind: "flatpak", Items: []string{"org.mozilla.Thunderbird"}}, Windows: "Mozilla.Thunderbird", Mac: "thunderbird", MacCask: true},
	"mpv":            {Name: "mpv", Linux: Task{Kind: "pkg", Items: []string{"mpv"}}, Windows: "shinchiro.mpv", Mac: "mpv", GitHub: "mpv-player/mpv"},
	"yt-dlp":         {Name: "yt-dlp", Linux: Task{Kind: "pipx", Items: []string{"yt-dlp"}}, Windows: "yt-dlp.yt-dlp", Mac: "yt-dlp", GitHub: "yt-dlp/yt-dlp"},
	"ripgrep": {Name: "ripgrep", Aliases: []string{"rg"}, Linux: Task{Kind: "pkg", Items: []string{"ripgrep"}}, Windows: "BurntSushi.ripgrep.MSVC", Mac: "ripgrep", GitHub: "BurntSushi/ripgrep"},
	"fd": {Name: "fd", Aliases: []string{"fd-find"}, Linux: Task{Kind: "pkg", Items: []string{"fd"}}, Windows: "sharkdp.fd", Mac: "fd", GitHub: "sharkdp/fd"},
	"bat": {Name: "bat", Aliases: []string{"batcat"}, Linux: Task{Kind: "pkg", Items: []string{"bat"}}, Windows: "sharkdp.bat", Mac: "bat", GitHub: "sharkdp/bat"},
	"delta": {Name: "delta", Aliases: []string{"git-delta"}, Linux: Task{Kind: "pkg", Items: []string{"git-delta"}}, GitHub: "dandavison/delta"},
	"duf": {Name: "duf", Linux: Task{Kind: "pkg", Items: []string{"duf"}}, GitHub: "muesli/duf"},
	"dust": {Name: "dust", Aliases: []string{"du-dust"}, Linux: Task{Kind: "pkg", Items: []string{"dust"}}, GitHub: "bootandy/dust"},
	"procs": {Name: "procs", Linux: Task{Kind: "pkg", Items: []string{"procs"}}, GitHub: "dalance/procs"},
	"bottom": {Name: "bottom", Aliases: []string{"btm"}, Linux: Task{Kind: "pkg", Items: []string{"bottom"}}, Windows: "Clement.bottom", Mac: "bottom", GitHub: "ClementTsang/bottom"},
	"zoxide": {Name: "zoxide", Aliases: []string{"z"}, Linux: Task{Kind: "pkg", Items: []string{"zoxide"}}, Windows: "ajeetdsouza.zoxide", Mac: "zoxide", GitHub: "ajeetdsouza/zoxide"},
	"eza": {Name: "eza", Aliases: []string{"exa"}, Linux: Task{Kind: "pkg", Items: []string{"eza"}}, Mac: "eza", GitHub: "eza-community/eza"},
	"tealdeer": {Name: "tealdeer", Aliases: []string{"tldr"}, Linux: Task{Kind: "pkg", Items: []string{"tealdeer"}}, GitHub: "dbrgn/tealdeer"},
	"sublime-text": {Name: "Sublime Text", Aliases: []string{"sublime", "subl"}, Linux: Task{Kind: "flatpak", Items: []string{"com.sublimetext.three"}}, Windows: "SublimeHQ.SublimeText.4", Mac: "sublime-text", MacCask: true},
	"notepadplusplus": {Name: "Notepad++", Aliases: []string{"npp", "notepad"}, Linux: Task{Kind: "pkg", Items: []string{"notepadplusplus"}}, Windows: "Notepad++.Notepad++"},
	"doublecmd": {Name: "Double Commander", Aliases: []string{"double-commander"}, Linux: Task{Kind: "flatpak", Items: []string{"net.doublecmd.DoubleCommander"}}, Windows: "doublecmd.doublecmd", Mac: "double-commander", MacCask: true},
	"lazygit":        {Name: "lazygit", Linux: Task{Kind: "pkg", Items: []string{"lazygit"}}, Windows: "JesseDuffield.lazygit", Mac: "lazygit", GitHub: "jesseduffield/lazygit"},
	"lazydocker":     {Name: "lazydocker", Linux: Task{Kind: "github", Items: []string{"jesseduffield/lazydocker"}}, Mac: "lazydocker", GitHub: "jesseduffield/lazydocker"},
	"onlyoffice":     {Name: "ONLYOFFICE", Aliases: []string{"onlyoffice-desktopeditors"}, Linux: Task{Kind: "flatpak", Items: []string{"org.onlyoffice.desktopeditors"}}, Windows: "ONLYOFFICE.DesktopEditors", Mac: "onlyoffice", MacCask: true},
	"localsend":      {Name: "LocalSend", Linux: Task{Kind: "flatpak", Items: []string{"org.localsend.localsend_app"}}, Windows: "LocalSend.LocalSend", Mac: "localsend", MacCask: true, GitHub: "localsend/localsend"},
	"stremio":        {Name: "Stremio", Linux: Task{Kind: "flatpak", Items: []string{"com.stremio.Stremio"}}, Windows: "Stremio.Stremio", Mac: "stremio", MacCask: true},
	"fastfetch":      {Name: "Fastfetch", Aliases: []string{"neofetch"}, Linux: Task{Kind: "pkg", Items: []string{"fastfetch"}}, Windows: "Fastfetch-cli.Fastfetch", Mac: "fastfetch", GitHub: "fastfetch-cli/fastfetch"},
	"btop":           {Name: "btop", Aliases: []string{"bpytop"}, Linux: Task{Kind: "pkg", Items: []string{"btop"}}, Windows: "aristocratos.btop4win", Mac: "btop", GitHub: "aristocratos/btop"},
	"htop":           {Name: "htop", Linux: Task{Kind: "pkg", Items: []string{"htop"}}, Windows: "htop.htop", Mac: "htop"},
	"git":            {Name: "Git", Linux: Task{Kind: "pkg", Items: []string{"git"}}, Windows: "Git.Git", Mac: "git"},
	"curl":           {Name: "curl", Linux: Task{Kind: "pkg", Items: []string{"curl"}}, Windows: "cURL.cURL", Mac: "curl"},
	"wget":           {Name: "wget", Linux: Task{Kind: "pkg", Items: []string{"wget"}}, Windows: "GNU.Wget2", Mac: "wget"},
	"python":         {Name: "Python", Aliases: []string{"python3"}, Linux: Task{Kind: "pkg", Items: []string{"python", "python-pip"}}, Windows: "Python.Python.3.12", Mac: "python"},
	"java":           {Name: "OpenJDK", Aliases: []string{"jdk", "openjdk"}, Linux: Task{Kind: "pkg", Items: []string{"jdk-openjdk"}}, Windows: "EclipseAdoptium.Temurin.21.JDK", Mac: "temurin", MacCask: true},
	"intellij":       {Name: "IntelliJ IDEA Community", Aliases: []string{"idea", "intellij-idea"}, Linux: Task{Kind: "flatpak", Items: []string{"com.jetbrains.IntelliJ-IDEA-Community"}}, Windows: "JetBrains.IntelliJIDEA.Community", Mac: "intellij-idea-ce", MacCask: true},
	"pycharm":        {Name: "PyCharm Community", Linux: Task{Kind: "flatpak", Items: []string{"com.jetbrains.PyCharm-Community"}}, Windows: "JetBrains.PyCharm.Community", Mac: "pycharm-ce", MacCask: true},
	"webstorm":       {Name: "WebStorm", Linux: Task{Kind: "flatpak", Items: []string{"com.jetbrains.WebStorm"}}, Windows: "JetBrains.WebStorm", Mac: "webstorm", MacCask: true},
	"android-studio": {Name: "Android Studio", Aliases: []string{"androidstudio"}, Linux: Task{Kind: "flatpak", Items: []string{"com.google.AndroidStudio"}}, Windows: "Google.AndroidStudio", Mac: "android-studio", MacCask: true},
	"dbeaver":        {Name: "DBeaver Community", Linux: Task{Kind: "flatpak", Items: []string{"io.dbeaver.DBeaverCommunity"}}, Windows: "dbeaver.dbeaver", Mac: "dbeaver-community", MacCask: true},
	"filezilla":      {Name: "FileZilla", Linux: Task{Kind: "flatpak", Items: []string{"org.filezillaproject.Filezilla"}}, Windows: "TimKosse.FileZilla.Client", Mac: "filezilla", MacCask: true},
	"virtualbox":     {Name: "VirtualBox", Linux: Task{Kind: "pkg", Items: []string{"virtualbox"}}, Windows: "Oracle.VirtualBox", Mac: "virtualbox", MacCask: true},
	"wine":           {Name: "Wine", Linux: Task{Kind: "pkg", Items: []string{"wine"}}, Mac: "wine-stable", MacCask: true},
	"lutris":         {Name: "Lutris", Linux: Task{Kind: "flatpak", Items: []string{"net.lutris.Lutris"}}, Mac: "lutris", GitHub: "lutris/lutris"},
	"retroarch":      {Name: "RetroArch", Linux: Task{Kind: "flatpak", Items: []string{"org.libretro.RetroArch"}}, Windows: "Libretro.RetroArch", Mac: "retroarch", MacCask: true},
	"mangohud":       {Name: "MangoHud", Linux: Task{Kind: "pkg", Items: []string{"mangohud"}}, GitHub: "flightlessmango/MangoHud"},
	"protonup-qt":    {Name: "ProtonUp-Qt", Aliases: []string{"protonup"}, Linux: Task{Kind: "flatpak", Items: []string{"net.davidotek.pupgui2"}}, Windows: "DavidoTek.ProtonUp-Qt", Mac: "protonup-qt", GitHub: "DavidoTek/ProtonUp-Qt"},
	"handbrake":      {Name: "HandBrake", Linux: Task{Kind: "flatpak", Items: []string{"fr.handbrake.ghb"}}, Windows: "HandBrake.HandBrake", Mac: "handbrake", MacCask: true},
	"calibre":        {Name: "Calibre", Linux: Task{Kind: "flatpak", Items: []string{"com.calibre_ebook.calibre"}}, Windows: "calibre.calibre", Mac: "calibre", MacCask: true},
	"musescore":      {Name: "MuseScore", Linux: Task{Kind: "flatpak", Items: []string{"org.musescore.MuseScore"}}, Windows: "MuseScore.MuseScore", Mac: "musescore", MacCask: true},
	"remmina":        {Name: "Remmina", Linux: Task{Kind: "flatpak", Items: []string{"org.remmina.Remmina"}}, GitHub: "FreeRDP/Remmina"},
	"tailscale":      {Name: "Tailscale", Linux: Task{Kind: "pkg", Items: []string{"tailscale"}}, Windows: "tailscale.tailscale", Mac: "tailscale", MacCask: true},
	"syncthing":      {Name: "Syncthing", Linux: Task{Kind: "pkg", Items: []string{"syncthing"}}, Windows: "Syncthing.Syncthing", Mac: "syncthing", GitHub: "syncthing/syncthing"},
	"nextcloud":      {Name: "Nextcloud Desktop", Linux: Task{Kind: "flatpak", Items: []string{"com.nextcloud.desktopclient.nextcloud"}}, Windows: "Nextcloud.NextcloudDesktop", Mac: "nextcloud", MacCask: true},
	"tor-browser":    {Name: "Tor Browser", Aliases: []string{"torbrowser"}, Linux: Task{Kind: "flatpak", Items: []string{"org.torproject.torbrowser-launcher"}}, Windows: "TorProject.TorBrowser", Mac: "tor-browser", MacCask: true},
	"wireguard":      {Name: "WireGuard", Linux: Task{Kind: "pkg", Items: []string{"wireguard-tools"}}, Windows: "WireGuard.WireGuard", Mac: "wireguard-tools"},
	"protonvpn":      {Name: "Proton VPN", Linux: Task{Kind: "flatpak", Items: []string{"com.protonvpn.www"}}, Windows: "Proton.ProtonVPN", Mac: "protonvpn", MacCask: true},
}

func knownAppKey(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := knownApps[v]; ok {
		return v, true
	}
	for k, app := range knownApps {
		for _, a := range app.Aliases {
			if v == a {
				return k, true
			}
		}
	}
	return "", false
}
