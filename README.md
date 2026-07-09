# Instally

Универсальный установщик пакетов для Linux, Windows и macOS.  
Определяет систему, подбирает менеджер и устанавливает приложения одной командой.

```bash
instally firefox discord vlc blender
```

## TUI-интерфейс

```bash
instally --gui
```

Текстовый интерфейс в терминале: поиск, проверка безопасности, план установки,
подтверждение, выполнение. Работает в любом терминале.

```bash
instally                        # открывает TUI если есть терминал
instally --gui                  # принудительно
```

## Установка

```bash
# Из релиза
curl -L https://github.com/xemoll/instally/releases/latest/download/linux-amd64 -o instally
chmod +x instally && ./instally --install-self

# Из исходников
git clone https://github.com/xemoll/instally
cd instally && go build -o instally ./cmd/instally
./instally --install-self --set-default-installer
```

## Быстрый старт

```bash
instally git curl wget                     # установка через менеджер
instally firefox discord vscode            # известные приложения
instally --preset dev,gaming               # пресеты
instally --multi "firefox, git, curl"      # мульти-установка
instally --batch list.txt                  # из файла
instally --text "firefox, git"             # из текста
```

## Поиск

```bash
instally --search python        # ищет в системном менеджере
instally --list-apps            # список известных приложений
instally --list-presets         # список пресетов
instally --which firefox        # где установлено и версия
instally --why firefox          # почему такой метод установки
```

## Методы установки

| Флаг | Что делает |
|------|-----------|
| `--pkg <name>` | Пакет системы (apt, pacman, dnf, winget, brew...) |
| `--aur <name>` | AUR (Arch) |
| `--flatpak <id>` | Flatpak из Flathub |
| `--snap <name>` | Snap |
| `--pipx <name>` | pipx |
| `--npm <name>` | npm global |
| `--cargo <crate>` | cargo install |
| `--go <pkg>` | go install |
| `--git <url>` | Клонировать и собрать из исходников |
| `--github <owner/repo>` | Скачать GitHub Release |
| `--url <url>` | Скачать и установить |
| `--local <path>` | Установить локальный файл |
| `--preset <name>` | Пресет приложений |

### Пресеты

| Пресет | Приложения |
|--------|-----------|
| `base` | git, curl, wget, fastfetch, btop, htop |
| `dev` | vscode, neovim, git, docker, node, python, go, lazygit |
| `gaming` | steam, heroic, lutris, prismlauncher, mangohud, protonup-qt |
| `media` | vlc, obs, blender, gimp, krita, kdenlive, audacity, handbrake |
| `work` | libreoffice, thunderbird, bitwarden, keepassxc, onlyoffice, joplin |
| `security` | wireshark, veracrypt, tailscale, bleachbit, protonvpn |
| `terminals` | alacritty, wezterm, kitty, fastfetch |

## Безопасность

```bash
instally --scan ./installer.sh               # проверить файл
instally --security-test                      # тест системы обнаружения
instally --vt-key "<key>" firefox            # с VirusTotal
instally --vt-save-key-stdin < keyfile       # сохранить ключ VT
instally --vt-status                         # статус VT
instally --vt-test                           # проверить ключ VT
instally --vt-clear-key                      # удалить ключ VT
```

Instally проверяет файлы перед установкой:
- SHA-256
- ClamAV и YARA (если установлены)
- VirusTotal (по ключу)
- Статический анализ скриптов
- Проверка архивов на path-traversal
- Проверка подписей (GPG, Authenticode, Gatekeeper)

## Диагностика

```bash
instally --detect             # информация о системе (JSON)
instally --doctor             # полная диагностика
instally --support            # матрица поддержки
instally --version            # версия
instally --build-info         # версия + Go
instally --stats              # статистика приложений
instally --env                # переменные окружения
instally --fix-broken         # починить менеджер пакетов
```

## План и логи

```bash
instally --export-plan plan.json firefox git    # сохранить план
instally --log install.log firefox git           # выполнить + лог
instally --dry-run firefox git vscode           # предпросмотр
```

## Обновление

```bash
instally --update firefox git     # обновить конкретные
instally --upgrade-all            # обновить все
instally --purge-cache            # очистить кэш
```

## Поддерживаемые менеджеры

Linux: apt, pacman, dnf, yum, zypper, apk, xbps, emerge, eopkg, snap, flatpak  
Windows: winget, chocolatey, scoop  
macOS: brew, port

## Сборка

```bash
git clone https://github.com/xemoll/instally
cd instally
go build -o instally ./cmd/instally
./instally --detect
```

## Лицензия

MIT
