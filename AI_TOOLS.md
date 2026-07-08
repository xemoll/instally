# AI tools pack

Instally can install a coding-AI pack:

```bash
instally --ai-tools --dry-run --yes
instally --ai-tools --yes
```

The pack includes:

- OpenCode
- Ollama
- Claude Code

## CachyOS / Arch

```bash
instally --ai-tools --dry-run --yes
instally --ai-tools --yes
```

Current plan:

- OpenCode: `pacman -S opencode` when the system manager is pacman.
- Ollama: official installer downloaded to cache, scanned, then run.
- Claude Code: official installer downloaded to cache, scanned, then run.

## Debian / Ubuntu

- OpenCode: npm fallback (`opencode-ai`) if no native package is available.
- Ollama: official installer downloaded, scanned, then run.
- Claude Code: signed official apt repository with fingerprint check, then `apt install claude-code`.

## Fedora / RHEL-like

- OpenCode: npm fallback.
- Ollama: official installer downloaded, scanned, then run.
- Claude Code: official signed dnf repository, then `dnf install claude-code`.

## Alpine

- OpenCode: npm fallback.
- Ollama: official installer downloaded, scanned, then run.
- Claude Code: official apk repository with key hash check, then `apk add claude-code`.

## Windows

Instally prefers the detected manager:

- winget: `Ollama.Ollama`, `Anthropic.ClaudeCode`; OpenCode via npm fallback unless scoop/choco is detected.
- scoop/choco: OpenCode through scoop/choco when available.

## macOS

With Homebrew:

- `brew install anomalyco/tap/opencode`
- `brew install --cask ollama`
- `brew install --cask claude-code`

## Safety

Official shell installers are never piped directly from curl into shell by Instally. Instally downloads them into cache, validates the URL, scans the file, and only then runs the checked cached script.

## Safer AI tools installation

`instally --ai-tools` installs OpenCode, Ollama, and Claude Code using the safest available method for the current OS:

- OpenCode: native package when available, otherwise npm in Instally's user npm prefix (`~/.local/share/instally/npm-global`) instead of sudo npm.
- Ollama: official install script downloaded with Instally's Go downloader, scanned, then executed from cache.
- Claude Code: signed apt/dnf/apk repositories where available, Homebrew/WinGet on supported systems, otherwise official install script downloaded, scanned, then executed from cache.

Use dry-run first:

```bash
instally --ai-tools --dry-run --yes
```

Terminal/agent mode:

```bash
printf 'opencode\nollama\nclaude-code\n' | instally --terminal-install --dry-run --yes
```
