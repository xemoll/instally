# Instally

**Instally** is a safer app installer launcher for Linux, Windows and macOS.

It is built around one rule: **never download the first random search result**. Plain app names are resolved only through trusted profiles, official package managers, conservative GitHub Release matching, or official allowlisted URLs.

## Download

Download the build for your OS from the repository **Releases** page.

Expected release assets:

| System | File |
|---|---|
| Linux x86_64 | `instally-linux-amd64.tar.gz` |
| Windows x86_64 | `instally-windows-amd64.zip` |
| macOS Intel | `instally-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `instally-darwin-arm64.tar.gz` |

Before running, verify the archive with [`SHA256SUMS.txt`](downloads/SHA256SUMS.txt).

> Note: this repository is intentionally release-oriented. Full development source can be kept private while stable builds and documentation are published here.

## Quick start

### Linux

```bash
mkdir -p instally
tar -xzf instally-linux-amd64.tar.gz -C instally --strip-components=1
cd instally
chmod +x instally instally-native install.sh
./instally --doctor
./instally --security-test
./instally firefox discord lazygit --dry-run
```

### Windows PowerShell

```powershell
Expand-Archive .\instally-windows-amd64.zip -DestinationPath .\instally
cd .\instally\instally_pkg_windows-amd64
.\instally.exe --doctor
.\instally.exe --security-test
.\instally.exe firefox --dry-run
```

### macOS

```bash
mkdir -p instally
tar -xzf instally-darwin-arm64.tar.gz -C instally --strip-components=1
cd instally
chmod +x instally instally-native install-macos.sh
./instally --doctor
./instally --security-test
```

## Examples

```bash
# Check environment and scanners
instally --doctor

# Run built-in security regression checks
instally --security-test

# Resolve and preview installation without touching the system
instally firefox discord lazygit --dry-run

# Update selected apps through supported package managers
instally --update firefox discord lazygit --dry-run

# Preview supported global upgrades
instally --upgrade-all --dry-run
```

## Safety model

- No Google/Yandex/Bing/random SEO download fallback.
- Known apps use trusted package managers, official profiles, or conservative GitHub Release matching.
- Plain HTTP and private/local URL downloads are blocked by default.
- Unsafe scan results are always blocked.
- `--allow-unknown` only allows incomplete scans; it does not allow unsafe or warning results.
- URL, local and GitHub installers are copied to private verified cache and SHA-256 checked before install.
- Source builds are blocked unless `--allow-source-build` is explicitly provided.

## What is public here

This public repository contains:

- user-facing documentation;
- security policy;
- release notes;
- user-facing release notes;
- checksums for the prepared binary archives.

It does **not** publish the full private development source tree.
