
## 30-system compatibility check

Run a resolver-only compatibility matrix without installing anything:

```bash
instally --compat-matrix
```

It validates install plans for 30 common Linux/Windows/macOS profiles and catches resolver/package-name issues before real installation.


## New in this build

- `--multi` for installing many apps at once.
- No-key security mode: VirusTotal is optional; local/system scanners still work.
- `--security-test` creates a safe EICAR test file and verifies that Instally blocks it.
- `scripts/run-50-checks.sh` runs 50+ dry-run/security compatibility checks.

Examples:

```bash
instally --multi "vscode, discord, telegram, github:cli/cli" --dry-run --yes
instally --security-test
```

See `NO_KEY_SECURITY.md` and `MULTI_INSTALL.md`.

# Instally

## Быстрый запуск GUI

```bash
./dist/linux-amd64/instally --gui
```

Новый основной сценарий:

1. выбери файл или вставь `github: owner/repo`, URL или имя программы;
2. нажми **Проверить**;
3. если проверка разрешила установку — нажми **Установить**;
4. либо сразу нажми **Проверить и установить**: Instally сначала выполнит проверку, затем продолжит установку только при разрешённом результате.

Примеры:

```bash
./instally --text 'vscode' --dry-run --yes
./instally --text 'gh:sharkdp/fd' --dry-run --yes
./instally --text 'local: "./some file.AppImage"' --dry-run --yes
```

Скриншоты лежат в `assets/screenshots/`.

 Go

Instally Go — универсальный установщик приложений, написанный полностью на Go. Один бинарник содержит CLI и GUI, не требует Python/Tkinter/GTK/Qt как runtime-зависимости и работает через локальный встроенный web-интерфейс.

## Что поддерживается

### Linux

- pacman: Arch, CachyOS, Manjaro, EndeavourOS, Garuda
- apt: Debian, Ubuntu, Mint, Pop!_OS, Kali, Zorin
- dnf/dnf5: Fedora, Nobara, RHEL-like
- zypper: openSUSE
- apk: Alpine
- xbps: Void Linux
- eopkg: Solus
- emerge: Gentoo
- nix: Nix/NixOS
- PackageKit/pkcon fallback
- AUR через paru/yay на Arch-like системах
- Flatpak, Snap, AppImage
- локальные `.deb`, `.rpm`, `.pkg.tar.*`, `.apk`, `.AppImage`, `.flatpakref`, `.flatpakrepo`, архивы исходников

### Windows

- winget
- scoop
- choco
- локальные `.msi`, `.exe`, `.appx`, `.msix`, `.appxbundle`, `.msixbundle`
- GitHub Releases и source workflows

### macOS

- Homebrew, включая cask-приложения
- MacPorts
- локальные `.pkg`, `.dmg`
- GitHub Releases и source workflows

## GUI

```bash
./instally --gui
```

GUI теперь сделан как простой мастер установки:

- одна крупная зона выбора файла/ссылки/GitHub/программы;
- drag-and-drop и кнопка выбора локального файла;
- локальный файл загружается во временный cache и сразу проверяется через `/api/upload-scan`;
- понятная карточка результата: что проверено, что ограничено, почему установка заблокирована;
- журнал процесса свернут по умолчанию, чтобы интерфейс не выглядел как терминал;
- кнопка установки активируется только после разрешающей проверки или ручного `allow unknown`;
- VirusTotal спрятан в блок “Дополнительная проверка”, чтобы не захламлять главный экран.

Для запуска без открытия браузера:

```bash
./instally --gui --no-open --port 39119
```

## CLI

```bash
./instally --detect
./instally --doctor
./instally --pkg git htop --dry-run --yes
./instally --flatpak com.visualstudio.code --dry-run --yes
./instally --github sharkdp/fd --dry-run --yes
./instally --release cli/cli --dry-run --yes
./instally --local ~/Downloads/app.AppImage --dry-run --yes
./instally --batch example-list.txt --dry-run --yes
```

Можно вставлять смешанный список:

```bash
./instally vscode discord telegram obs github:sharkdp/fd ~/Downloads/app.AppImage --dry-run --yes
```

## Умные имена приложений

Instally умеет распознавать популярные имена и выбирать нормальный источник под систему:

```text
vscode discord telegram spotify steam obs vlc blender gimp krita firefox brave chrome godot neovim docker node go rust alacritty wezterm kitty obsidian bitwarden keepassxc signal slack zoom postman insomnia github-desktop qbittorrent inkscape kdenlive audacity libreoffice thunderbird mpv yt-dlp
```

Пример:

- Linux: `vscode` → Flatpak `com.visualstudio.code`
- Windows: `vscode` → winget `Microsoft.VisualStudioCode`
- macOS: `vscode` → `brew install --cask visual-studio-code`

## GitHub

`github:` теперь работает как smart install:

1. пробует найти последний совместимый GitHub Release asset;
2. выбирает подходящий файл по OS/архитектуре/формату;
3. скачивает его во внутренний cache;
4. передаёт в локальный установщик;
5. если релиза или подходящего asset нет — откатывается к `git clone` + автоопределение сборки.

Поддерживаемые варианты:

```text
github: owner/repo
release: owner/repo
https://github.com/owner/repo
https://github.com/owner/repo/releases/latest/download/app.AppImage
```

Source build автоматически проверяет:

- `PKGBUILD`
- `Cargo.toml`
- `go.mod`
- `package.json`
- `CMakeLists.txt`
- `meson.build`
- `configure`
- `Makefile`
- `pyproject.toml`
- `setup.py`

## Batch format

```text
vscode discord telegram obs
pkg: git htop curl
flatpak: com.visualstudio.code org.telegram.desktop
snap: discord
github: sharkdp/fd cli/cli
release: AppImageCrafters/appimage-builder
cargo: ripgrep
npm: pnpm
pipx: black
local: ~/Downloads/app.AppImage
url: https://example.com/app.AppImage
```

## Установщик по умолчанию

Linux:

```bash
./instally --install-self --yes
./instally --set-default-installer --yes
```

или полностью:

```bash
./install-full.sh
```

Это устанавливает бинарник, создаёт `.desktop`, регистрирует MIME-типы и подключает файлы пакетов к Instally там, где это разрешает окружение рабочего стола.

Windows и macOS ограничивают тихий захват default-приложений. Instally регистрирует приложение/context/open-with, а финальный выбор default может требовать системные настройки.

## Сборка

```bash
go build -o instally ./cmd/instally
```

Cross-build:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/instally.exe ./cmd/instally
GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/instally ./cmd/instally
GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/instally ./cmd/instally
GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/instally ./cmd/instally
```

## Безопасность

Сначала запускай `--dry-run`. Instally показывает точные команды до реальной установки. GUI тоже сначала строит план и показывает шаги.

## Безопасная установка

Новая логика Instally работает по схеме: источник → скачивание во временный cache → проверка → установка. Для локальных файлов, прямых URL и GitHub Release assets установка по умолчанию теперь проходит через `--install-local-safe`, поэтому файл сначала сканируется и только потом передаётся обработчику `.AppImage`, `.deb`, `.rpm`, `.msi`, `.dmg`, архивов и других форматов.

Уровни проверки:

- SHA-256 и определение типа файла.
- Локальный антивирус: `clamdscan`/`clamscan` на Linux/macOS, Microsoft Defender через `MpCmdRun.exe` на Windows.
- Проверка подписи там, где это возможно: Gatekeeper/spctl на macOS, Authenticode через PowerShell на Windows, `.sig` + gpg на Linux.
- Лёгкая статическая эвристика для небольших текстовых установщиков/скриптов.
- VirusTotal API v3: сначала hash lookup, а upload выполняется только если пользователь явно включил `--vt-upload` или галку в GUI.

Важно: 100% гарантии безопасности не бывает. Instally не пишет “абсолютно безопасно”; он показывает `clean`, `limited`, `warning`, `unsafe` или `error` и объясняет, что именно не удалось проверить. Если проверка `limited`, установка по умолчанию блокируется. Её можно разрешить вручную через `--allow-unknown` или галку “разрешить неполную проверку”.

Примеры:

```bash
instally --scan ./app.AppImage
INSTALLY_VT_API_KEY=... instally --scan ./app.AppImage
INSTALLY_VT_API_KEY=... instally --scan ./app.AppImage --vt-upload
instally --install-local-safe ./app.AppImage --yes
instally --github sharkdp/fd --yes
```

GUI стал проще: одно поле источника, VirusTotal API key, две явные галки privacy/strictness, кнопки `Проверить`, `Dry-run`, `Установить безопасно`, карточка результата и живой лог.


## GitHub smart install

`github: owner/repo` и `--github owner/repo` сначала пытаются найти подходящий бинарный asset в последних GitHub Releases. Если `latest` не подходит, Instally просматривает несколько последних релизов. Выбор учитывает ОС, архитектуру, формат пакета и старается предпочесть нормальный установочный файл: AppImage/deb/rpm/pkg.tar для Linux, msi/exe/msix/zip для Windows, dmg/pkg/zip для macOS.

Если готовый asset не найден, остаётся fallback на `git clone` и автоопределение сборки: `PKGBUILD`, `Cargo.toml`, `go.mod`, `package.json`, `CMakeLists.txt`, `meson.build`, `configure`, `Makefile`, `pyproject.toml`, `setup.py`.

Для приватных репозиториев или большого количества запросов можно передать `GITHUB_TOKEN` в окружении.


## Cleaner link-first GUI

The current GUI focuses on one flow: choose a file, paste a URL/GitHub repo/app name, scan it, then install. URL and GitHub inputs now show recognition and a small four-step flow: source, download/cache, security scan, install. Safe installation reuses the already checked cache file whenever possible, so links and GitHub release assets are not downloaded twice.

Useful examples:

```bash
instally --gui
instally --text 'vscode' --dry-run --yes
instally --text 'gh:sharkdp/fd' --dry-run --yes
instally --url 'https://example.com/app.AppImage' --dry-run --yes
```

## Последнее обновление UI

Интерфейс переписан под чистый стиль: один основной экран, большая зона выбора файла, поле ссылки/GitHub/имени программы, две основные кнопки и свернутые блоки «Дополнительно» и «Журнал». Результат проверки появляется только после действия, поэтому стартовый экран не перегружен.

### Улучшения установщика

- `github: https://github.com/owner/repo/...` теперь нормализуется в `owner/repo` и идет через GitHub Release smart-install.
- URL-установка больше не зависит от `curl`/PowerShell-скрипта в плане: используется встроенный Go-режим `--install-url-safe`.
- `--install-url-safe` скачивает файл во временный cache, проверяет его и только потом передает в локальный установщик.
- Проверенный cache-файл используется повторно в safe-run, чтобы не скачивать один и тот же файл заново.
- В интерфейсе добавлены защиты от вылезания текста: перенос длинных ссылок/хешей, `min-width:0` для grid/flex-блоков, аккуратные отступы и скрытие вторичных элементов.


## Native GUI без HTML/CSS

Добавлен нативный desktop-интерфейс на Go + Fyne: `native/fyne`. Он не использует HTML/CSS, webview, localhost-сервер или браузер. Основной CLI остаётся без внешних зависимостей, а нативная оболочка вынесена отдельным модулем. Подробнее: `NATIVE_GUI.md`.

Запуск:

```bash
cd native/fyne
go mod tidy
go run .
```


## VirusTotal quick setup

```bash
instally --vt-save-key YOUR_API_KEY
instally --vt-status
instally --scan ./app.AppImage --vt-upload
```

Instally checks hash reports first. Unknown files are uploaded only when `--vt-upload` or `INSTALLY_VT_UPLOAD=1` is enabled. Large files use VirusTotal upload URLs automatically.

## Language

```bash
instally --lang en --support
instally --lang ru --support
```

Native GUI also has a language selector in Advanced.

## Expanded security and multi-install

### Save a VirusTotal key safely

Do not paste the key into scripts or commit it into the project. Use stdin so it does not remain in shell history:

```bash
printf '%s' 'PASTE_NEW_KEY_HERE' | instally --vt-save-key-stdin
instally --vt-status
instally --vt-test
```

### Install several programs

```bash
instally --multi "vscode, discord, telegram, github:cli/cli" --dry-run --yes
instally --multi "vscode, discord, telegram, github:cli/cli" --yes
```

Continue after a failed item:

```bash
instally --multi "vscode, discord, telegram" --yes --continue-on-error
```

### Security self-test

```bash
instally --security-test
```

The self-test uses the standard EICAR test string and must be blocked by Instally before installation.

## Universal 15+ app install test

This build expands universal installation handling for common desktop, developer, gaming, media, security and terminal apps. The main multi-install path was tested with 15+ popular apps across Linux pacman, Linux apt, Windows winget and macOS Homebrew dry-run profiles.

Examples:

```bash
instally --multi "vscode, discord, telegram, firefox, brave, obs, vlc, blender, gimp, krita, steam, docker, node, go, rust" --dry-run --yes
instally --preset dev --dry-run --yes
instally --preset gaming --dry-run --yes
instally --batch apps.txt --yes --continue-on-error
```

## Terminal / agent-friendly installer

For SSH sessions, terminal-only desktops, or AI agents that need a simple text workflow:

```bash
instally --terminal-install --yes
```

Paste a comma-separated list or one item per line, then press an empty line:

```text
vscode, discord, telegram
github: cli/cli
https://example.com/app.AppImage
```

Instally prints the exact plan first. If a command needs admin rights on Linux, it tries `pkexec` in a desktop session and otherwise uses `sudo`, so the password prompt appears in the terminal. For scripts such as Ollama and Claude Code installers, Instally downloads to cache, scans the file, then runs the checked copy instead of using raw `curl | sh`.

### Package-manager recovery

If a system package install fails, Instally now refreshes the package manager metadata/source list and retries once. Supported recovery covers apt, pacman, dnf, zypper, apk, xbps, eopkg, emerge, PackageKit, Homebrew, MacPorts, WinGet, Scoop and Chocolatey. Dry-run output shows the recovery command, for example `on failure: apt-get update && retry once`.

