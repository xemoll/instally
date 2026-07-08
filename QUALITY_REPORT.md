# Instally quality pass

This build focuses on reliability of the install plan before any real system changes.

## Fixed / hardened

- Package-like items are validated before being passed to package managers.
- Values starting with `-` are rejected so user input cannot become package-manager flags.
- Control characters, bidi controls, and shell-like metacharacters are rejected for package/app IDs.
- Bad URL schemes and private/local URLs are rejected at plan-building time.
- Git source targets reject `file://`, credentials, control characters and shell-like metacharacters.
- GitHub Release dry-run now shows the real post-scan local install plan instead of a second scan wrapper.
- Admin-required commands now fail early with a clear error if neither `pkexec` nor `sudo` is available on Unix-like systems.
- Multi-install and batch inputs were tested across Linux, Windows and macOS manager profiles.

## Smoke/QA run

Latest smoke run:

```text
passed=39 failed=0 total=39
```

Unit test pass events from `go test -json ./...`:

```text
pass_tests=182
```

Covered areas:

- Linux profiles: pacman, apt, dnf, zypper, apk, xbps, eopkg, emerge, nix, PackageKit.
- Windows profiles: winget, scoop, choco.
- macOS profiles: Homebrew, MacPorts.
- Large multi-install list: VS Code, Discord, Telegram, Firefox, Brave, OBS, VLC, Blender, GIMP, Krita, Steam, Docker, Node, Go, Rust, Git, curl, fastfetch, btop, qBittorrent, lazygit, yt-dlp.
- Mixed batch text: app names, GitHub repos, URL installers, local files.
- Security checks: EICAR blocking, fake AppImage, suspicious shell script, zip/tar traversal, private URL blocking, bad package names, bad git schemes.
- Key leakage scan: the previously pasted key is not present in the project tree.
