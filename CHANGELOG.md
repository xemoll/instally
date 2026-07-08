# Changelog

## v1.1.0 (2026-07-08)

### New features
- Added `--update <apps>` — update specific packages via system manager
- Added `--upgrade-all` — upgrade all system packages
- Added `--purge-cache` — clear Instally download cache
- Added `--build-info` — show version, build date, Go runtime
- Added `--stats` — show known-apps statistics by package manager
- Added 15 new known apps: joplin, wireshark, flameshot, peek, freetube, transmission, veracrypt, gparted, bleachbit, ventoy, rclone, restic
- Known apps now count: 85+ applications

### Cleanup
- Removed AI-related flags, docs, tests, presets, and app references
- Removed 13 obsolete markdown documentation files (COMPATIBILITY_30, ADVANCED_SECURITY, QUALITY_REPORT, etc.)
- `preset "ai"` renamed to `"cli"` — includes ripgrep, fd, bat, delta, eza, zoxide, tealdeer

### Fixes
- Release archives now output to `dist/archives/` directory (tar.gz was producing broken paths)
- SHA256SUMS.txt correctly generated from single source folder

## Earlier versions (combined)

### Deep security hardening
- Blocked plain HTTP downloads by default
- Blocked insecure Git schemes
- Archive traversal/link/bomb findings promoted from warning to unsafe
- VirusTotal API keys no longer forwarded through child process environments
- Added stricter path/name sanitization for cache/build outputs

### Native audit + VirusTotal hardening
- Persistent VirusTotal config: `--vt-save-key`, `--vt-clear-key`, `--vt-status`
- Hash-lookup-first VT flow; upload only with `--vt-upload`
- Large-file VT upload via `/files/upload_url`
- URL reputation checks before downloads
- Russian/English language support: `--lang ru|en`
- Native Fyne GUI with language selector

### UI implementation pass
- Clean minimal Fyne GUI with source input, scan, install flow
- Drag-and-drop local files
- Safe install: scan first, stop on unsafe, install from cache
- `--install-url-safe` for URL → cache → scan → install
- `--support` quick support matrix

### Multi-install & security expansion
- No-key security mode (VirusTotal optional)
- `--security-test` with EICAR test
- Embedded EICAR signature blocking
- Optional YARA scan integration
- `--multi` for batch installs
- `--continue-on-error` for multi workflows
- Extended local security checks (PE/ELF/Mach-O, double-extensions, archive bombs, script heuristics)
- `--vt-save-key-stdin` for secure key entry
- `--vt-test` key self-test

### Universal install expansion
- 30+ system compatibility matrix (`--compat-matrix`)
- `--preset` groups: base, dev, gaming, media, work, security, terminals
- Flatpak Flathub auto-ensure
- Filename policy checks (bidi, control chars, masked extensions)

### Package manager recovery
- Refresh-and-retry recovery for 14 package managers
- Dry-run shows recovery command
- Real runner tests for first-install failure → refresh → retry

### Quality & fixes
- Package/app ID validation before command generation
- URL validation at plan time
- GitHub Release dry-run shows local install command
- Admin elevation check (pkexec/sudo)
- Flatpak uses `--user` by default
- Secure ephemeral VT key-file propagation