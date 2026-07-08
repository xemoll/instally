# Instally

Multi-platform package installer — Linux, Windows, macOS.  
Detects your system, picks the right package manager, and installs apps with a single command.

```bash
instally firefox discord vlc blender
```

## Quick start

```bash
# Install apps — auto-detects system manager or known app profile
instally git curl wget

# List what's available
instally --list-apps
instally --list-presets

# Check your system
instally --detect
instally --doctor

# Batch install
instally --preset dev
instally --preset gaming,media

# Multi-install
instally --multi "firefox, vscode, discord"
instally --multi "git" --multi "curl" --multi "wget"
```

## Install methods

| Flag | What it does |
|------|-------------|
| `--pkg <name>` | Native package (apt, pacman, dnf, winget, brew, ...) |
| `--flatpak <id>` | Flatpak from Flathub |
| `--snap <name>` | Snap package |
| `--pipx <name>` | pipx tool |
| `--npm <name>` | npm global tool |
| `--cargo <crate>` | cargo install |
| `--go <pkg>` | go install |
| `--git <url>` | Clone + build from git |
| `-remote <owner/repo>` | Download latest GitHub release |
| `--url <url>` | Download and run installer |
| `--local <path>` | Install local file |
| `--aur <name>` | AUR package (Arch) |

### Presets

| Preset | Apps |
|--------|------|
| `base` | git, curl, wget, fastfetch, btop, htop |
| `dev` | vscode, neovim, git, docker, node, python, go, lazygit |
| `gaming` | steam, heroic, lutris, prismlauncher, mangohud, protonup-qt |
| `media` | vlc, obs, blender, gimp, krita, kdenlive, audacity, handbrake |
| `work` | libreoffice, thunderbird, bitwarden, keepassxc, onlyoffice, joplin |
| `security` | wireshark, veracrypt, tailscale, bleachbit, protonvpn |
| `terminals` | alacritty, wezterm, kitty, fastfetch |

## Advanced

### Security scanning
```bash
instally --scan ./installer.sh                          # scan file
instally --security-test                                 # test detection pipeline
instally --vt-key "<key>" --vt-upload --pkg firefox      # with VirusTotal
instally --vt-save-key-stdin < <keyfile>                 # save VT key securely
```

### Download cache management
```bash
instally --purge-cache                                   # clear cached downloads
```

### Update & upgrade
```bash
instally --update firefox git                            # update specific apps
instally --upgrade-all                                   # upgrade all system packages
```

### Diagnostics
```bash
instally --detect                                        # print system info (JSON)
instally --doctor                                        # full diagnostics
instally --support                                      # support matrix
instally --version                                      # version
instally --build-info                                   # version + Go runtime
instally --stats                                        # known-apps statistics
instally --which git                                    # locate binary + version
instally --why firefox                                  # explain install method
instally --verify-installed firefox git vscode          # check installed status
instally --search keyword                               # search package repos
instally --env                                         # show all INSTALLY_* vars
instally --fix-broken                                  # repair broken manager
instally --compat-matrix                                # 30-system dry-run matrix
```

### Plan & log
```bash
instally --export-plan plan.json firefox git vscode     # save plan as JSON
instally --log install.log firefox git                   # run + write log
instally --dry-run firefox git vscode                   # preview without running
```

### Batch files
```bash
instally --batch list.txt
instally --text "firefox, git, vscode"
```

### Language
```bash
instally --lang ru       # Russian UI
instally --lang en       # English (default)
```

## Supported package managers

Linux: apt, pacman, dnf, yum, zypper, apk, xbps, emerge, eopkg, snap, flatpak  
Windows: winget, chocolate, scoop  
macOS: brew, port

## Build from source

```bash
git clone https://github.com/xemoll/instally
cd instally
go build -o instally ./cmd/instally
./instally --detect
```

## License

MIT