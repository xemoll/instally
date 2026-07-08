# Package manager recovery

Instally now adds a recovery step for package-manager installs. The first install command is tried normally. If it fails, Instally refreshes the package manager metadata/source list and retries the original install once.

This is meant to fix common real-world failures such as stale apt indexes, stale pacman databases, outdated dnf/zypper/apk/xbps metadata, stale Homebrew taps, stale WinGet sources, or outdated Scoop buckets.

## Refresh commands

| Manager | Refresh before retry |
|---|---|
| apt | `apt-get update` |
| pacman | `pacman -Sy` |
| dnf | `dnf makecache --refresh` |
| zypper | `zypper refresh` |
| apk | `apk update` |
| xbps | `xbps-install -S` |
| eopkg | `eopkg ur` |
| emerge | `emerge --sync` |
| PackageKit | `pkcon refresh` |
| Homebrew | `brew update` |
| MacPorts | `port selfupdate` |
| WinGet | `winget source update` |
| Scoop | `scoop update` |
| Chocolatey | `choco source list`, then retry; if it still fails, check enabled sources |

Dry-run output shows the recovery action:

```bash
instally --pkg firefox --dry-run --yes
```

Example:

```text
apt-get install firefox -y
on failure: apt-get update && retry once
```

Instally does not silently loop forever. It attempts:

1. install;
2. refresh metadata if supported;
3. retry the same install once;
4. show diagnostics if it still fails.

Diagnostics detect common causes: missing package name, disabled repository/source, DNS or VPN problems, package-manager locks, permission issues, mirror failures, signature/GPG errors.
