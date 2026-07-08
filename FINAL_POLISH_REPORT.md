# Instally final polish / bugfix pass

This build focuses on installation reliability, safer defaults, clearer output, and secret handling.

## Fixed/improved

- Fixed duplicated/out-of-order command output in real installs. Buffered runner now captures output once; streaming runner remains available for live UI/terminal flows.
- Flatpak now defaults to user-scope installs (`--user`) to avoid unnecessary root prompts and broken system-scope remotes on desktop Linux.
- Added explicit opt-in for system-wide Flatpak: `INSTALLY_FLATPAK_SYSTEM=1`.
- Added one-shot VirusTotal key forwarding without exposing the raw key in child process args or logs: Instally writes an ephemeral `0600` key file, passes only its masked path, and removes it after the child process exits.
- Added `INSTALLY_VT_KEY_FILE` support for secure key-file based CI/agent usage.
- Added regression tests for Flatpak scope, secure VT key-file loading/cleanup, and command-line masking.
- Extended explicit batch kind support for `official-firefox` and `official-discord`.
- Kept HTTP downloads and insecure Git blocked by default.
- Kept package-manager refresh + retry for apt/pacman/dnf/zypper/apk/xbps/eopkg/emerge/PackageKit/brew/port/winget/scoop/choco.

## Checks run

- `go test ./...`
- `go vet ./...`
- Linux build
- Windows amd64 build
- macOS amd64 build
- macOS arm64 build
- shell syntax checks
- 30-system compatibility matrix
- EICAR security self-test
- live practical install of `jq, git, curl`
- multi-install dry-run for Firefox, Discord, Telegram, VSCode, Git, curl, jq, OpenCode, Ollama, Claude Code
- Flatpak user-scope dry-run
- bad input blocking
- HTTP blocking
- old VirusTotal key secret scan

See `instally_final_polished_checks.log` for the full run.
