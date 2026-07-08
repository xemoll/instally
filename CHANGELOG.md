
## Deep security hardening

- Blocked plain HTTP downloads by default.
- Blocked insecure Git schemes by default.
- Promoted archive traversal/link/bomb findings from warning to unsafe.
- Stopped forwarding VirusTotal API keys through child process environments.
- Added stricter path/name sanitization for cache/build outputs.
- Kept URL dry-run non-networked and clearer.
- Added regression tests for the new security rules.


## Native audit + VirusTotal hardening

- Added persistent VirusTotal configuration: `--vt-save-key`, `--vt-clear-key`, `--vt-status`.
- VirusTotal now performs hash lookup first and only uploads files when `--vt-upload` / `INSTALLY_VT_UPLOAD=1` is explicitly enabled.
- Added large-file VirusTotal support via `/files/upload_url` with streaming multipart upload up to the configured max size.
- Added VirusTotal URL reputation checks before URL downloads when an API key is configured.
- Added Russian/English language support via `--lang ru|en` and `INSTALLY_LANG`.
- Native Fyne GUI now includes a language selector and localized core labels.
- Added tests for VT config, language selection, upload limits, URL dry-run, package managers and platform dry-runs.
- Hardened URL validation, private URL blocking, cache naming, release fallback, and dry-run plans.


## Audit hardening update

- Dark native UI toned down: less neon, shorter labels, safer layout for long text.
- URL downloads now block local/private hosts by default to reduce SSRF/local-network risk. Use INSTALLY_ALLOW_PRIVATE_URLS=1 only when you intentionally install from a trusted LAN/local source.
- GitHub source-build fallback is blocked unless --allow-unknown is passed.
- Release asset scoring now respects forced/target architecture via INSTALLY_FORCE_ARCH.
- Local safe dry-run now shows the real installation plan after scanning.

# Changelog

## Polished UI + safer workflow

- Упрощён GUI: одна главная карточка, две основные кнопки, скрытые дополнительные настройки и скрытый журнал.
- Добавлен режим `Проверить и установить` через `/api/safe-run-stream`.
- GUI теперь сначала показывает понятный результат проверки, затем разрешает установку.
- Добавлена более честная обработка системных менеджеров: для pkg/Flatpak/Snap/Winget/Brew показывается проверка источника, а не фейковый VirusTotal без файла.
- Улучшен парсер: поддержка `gh:owner/repo`, `github.com/owner/repo`, путей Windows, локальных установщиков без существующего файла, путей с пробелами в кавычках.
- Улучшены dry-run сценарии для Linux/Windows/macOS.
- Добавлены PNG-превью главного экрана и результата проверки.

## Polished link flow update

- Added a clearer link/GitHub loading flow in the GUI: source recognition, download/cache, security scan, installation.
- Added `/api/inspect` so the GUI can recognize URLs, GitHub, local files, package-manager apps and git repositories before running a scan.
- Safe install now reuses already checked cached files for URL/GitHub/local sources instead of downloading/scanning twice.
- Hardened downloads with scheme validation, per-host cache directories, unique filenames, temporary `.part` files, size checks and retry attempts.
- Added tests for source inspection and checked-cache installation planning.

## UI implementation pass

- Implemented the clean minimal UI style directly inside the Go GUI, not as a separate mockup.
- Fixed overflowing text in buttons, long URLs, SHA fields, result cards, status badges, and checklist rows.
- Reduced the default screen to the core installer flow.
- Added built-in `--install-url-safe` for URL download → cache → scan → install.
- Normalized GitHub URL inputs passed with `github:` / `release:` prefixes.
- Added tests for GitHub URL normalization and URL safe installer planning.

## Humanized UI polish

- Installation buttons are now light-blue and calmer instead of dark/technical.
- Link handling in the GUI now shows a human-readable source card: what was recognized, what will be downloaded, and why it is safe to continue.
- The main flow now uses simple wording: source, download, check, install.
- Result cards now use short user-facing text instead of technical scan language.
- Long URLs, hashes, and commands are wrapped safely and no longer overflow cards.
- Linux `.run`, `.bin`, and `.sh` installers can be handled after the safe scan path.
- Added more app aliases: Zed, lazygit, lazydocker, ONLYOFFICE, LocalSend, Stremio.

## Native Fyne UI polish

- Переработан `native/fyne/main.go`: интерфейс стал мастером установки вместо технической панели.
- Добавлен drag-and-drop локальных файлов в нативное окно.
- Добавлены состояния шагов: источник, загрузка, проверка, установка.
- Добавлена кнопка «План» для dry-run плана без запуска установки.
- Добавлена кнопка «Установить проверенное», которая использует результат последней проверки и уже скачанный cache-файл.
- `instally --gui` теперь пытается запускать `instally-native`; HTML/CSS GUI оставлен только как `--legacy-web-gui`.
- Добавлены UI helper-функции для человекочитаемых статусов, планов и короткого SHA.

## Native dark functional UI

- Reworked the Fyne interface into a dark, minimal installer screen.
- Removed the marketing-style headline from the start screen.
- Kept the start screen focused on source input, file selection, one primary install action, compact steps, advanced settings, and log.
- Moved explanatory text into scan/install states and result messages.
- Reduced visible actions: the default screen now has one primary button, while scan-only and plan actions are hidden under Advanced.
- Improved safe-run flow in the native GUI: scan first, stop on unsafe or incomplete checks, install only from the checked cache/source.
- Fixed VirusTotal/env option propagation for safe local/URL installs.

## Native dark UI stability pass

- Reworked native Fyne window to reduce text overflow and overlapping content.
- Long URLs, SHA hashes, warnings and command plans are shortened in the visible UI and kept fully available in the log.
- Switched the native layout to a safer vertical structure with wrapped labels and shorter action text.
- Added `--support` to print a quick support matrix for the current machine.
- Improved URL dry-run: it now preserves the real file name and extension in the cache preview.
- Added `.7z` archive extraction support through 7-Zip/p7zip.
- Improved macOS `.dmg` install handling by mounting and copying `.app` bundles to `~/Applications` when possible.

## No-key security + multi-install hardening

- Added no-key security mode: VirusTotal is optional, not required.
- Added `--security-test` with a safe EICAR test signature file.
- Added embedded EICAR signature blocking, so the pipeline can be validated even without ClamAV/VirusTotal.
- Added optional YARA scan integration when `yara` is available.
- Added `--multi` for comma/semicolon/repeated multi-install input.
- Added `scripts/run-50-checks.sh` with 50+ dry-run and security checks across Linux, Windows and macOS profiles.
- Improved support summary to show VirusTotal as optional and YARA as an extra layer.

## Expanded security + multi-install hardening

- Added extended local security checks before installation:
  - file structure/magic checks for PE, ELF, Mach-O, AppImage, RPM, deb/ar and scripts;
  - suspicious double-extension detection such as `invoice.pdf.exe`;
  - world-writable file warning;
  - zip/tar path-traversal checks;
  - archive symlink/link checks;
  - archive-bomb size and entry-count heuristics;
  - expanded script heuristics for `curl|bash`, PowerShell encoded commands, Defender disable commands, persistence, startup tasks, sudoers/systemd changes, base64 blobs and SSH/private key strings.
- Added `--vt-save-key-stdin` so a VirusTotal key can be saved without putting it in shell history.
- Added `--vt-test` for a configured-key check using the EICAR hash lookup path.
- Added `--continue-on-error` for multi/batch installation workflows.
- Fixed known app profiles whose Linux installer is `pipx`, `npm`, `cargo`, or `go`.
- Added 130-check regression run covering CLI, security, multi-install, managers, formats, languages, URL safety, archives and cross-builds.

## Universal install expansion

- Added many more app aliases across Linux/Windows/macOS.
- Added `--preset` install groups: base, dev, gaming, media, work, security, terminals.
- Improved Flatpak flow by ensuring Flathub exists before installing apps.
- Added filename policy checks: bidi/control characters, masked double extensions, overly long names.
- Added installer metadata checks for tiny packages/fake AppImages/script installers.
- Added broader multi-install dry-run coverage for 15+ popular applications.

## Quality hardening pass

- Added package/app ID validation before command generation.
- Rejected option-like install names, control characters, bidi controls and shell-like metacharacters.
- Added URL validation at plan time for unsafe schemes/private hosts.
- Hardened git source validation.
- Improved GitHub Release dry-run to show the actual local install command after a scan.
- Added early failure when admin elevation is required but neither pkexec nor sudo is available.
- Added `QUALITY_REPORT.md` and expanded smoke checks across Linux/Windows/macOS profiles.

## RC: 30-system quality pass

- Added `--compat-matrix` dry-run validation across 30 common OS/distro profiles.
- Added `INSTALLY_FORCE_OS_ID` and `INSTALLY_FORCE_OS_LIKE` for better distro simulation and tests.
- Added per-manager native package normalization for packages such as Go, Rust, Python, Java, Docker and Node.
- Fixed multi-install parsing: semicolon is no longer treated as a separator. Inputs like `bad;name` are rejected instead of being split into two installable package names.
- Added compatibility tests for 30 systems and package alias regression tests.
- Added compatibility documentation in `COMPATIBILITY_30.md`.


## Live verification fix
- Fixed warnings-only plans: unsafe/invalid input that produces no runnable commands now exits with non-zero status.
- Added regression test `TestRunPlanWarningsOnlyFails`.
- Rebuilt bundled Linux/Windows/macOS binaries after the fix.

## Package manager recovery update

- Added refresh-and-retry recovery for apt, pacman, dnf, zypper, apk, xbps, eopkg, emerge, PackageKit, Homebrew, MacPorts, WinGet, Scoop and Chocolatey.
- Dry-run now shows the exact recovery command with `on failure: ... && retry once`.
- Runtime diagnostics now explain stale repositories, missing package names, DNS/VPN issues, package-manager locks, permission problems and signature/mirror failures.
- Real runner tests now simulate first-install failure, metadata refresh and successful retry across 14 package managers.


## Final polish quality pass

- Fixed duplicated/out-of-order output during real installs.
- Flatpak now uses `--user` by default; set `INSTALLY_FLATPAK_SYSTEM=1` for system-wide Flatpak installs.
- Added secure ephemeral VirusTotal key-file propagation for child safe-install commands without raw-key environment leakage.
- Added `INSTALLY_VT_KEY_FILE` support.
- Added more regression tests around Flatpak, VirusTotal key masking/cleanup, and install output reliability.
- Re-ran full build/test/compat/security/practical install checks.
