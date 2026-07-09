#!/usr/bin/env bash
set +e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 1
LOG="${1:-expanded-checks.log}"
: > "$LOG"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM
pass=0
fail=0
n=0
say(){ printf '%s\n' "$*" | tee -a "$LOG" >/dev/null; }
check(){
  n=$((n+1)); name="$1"; shift
  out="$("$@" 2>&1)"; code=$?
  if [ $code -eq 0 ]; then pass=$((pass+1)); say "PASS $n $name"; else fail=$((fail+1)); say "FAIL $n $name code=$code cmd=$*"; say "$out"; fi
}
check_contains(){
  n=$((n+1)); name="$1"; want="$2"; shift 2
  out="$("$@" 2>&1)"; code=$?
  if [ $code -eq 0 ] && printf '%s' "$out" | grep -Fq -- "$want"; then pass=$((pass+1)); say "PASS $n $name"; else fail=$((fail+1)); say "FAIL $n $name code=$code want=$want cmd=$*"; say "$out"; fi
}
check_fail_contains(){
  n=$((n+1)); name="$1"; want="$2"; shift 2
  out="$("$@" 2>&1)"; code=$?
  if [ $code -ne 0 ] && printf '%s' "$out" | grep -Fq -- "$want"; then pass=$((pass+1)); say "PASS $n $name"; else fail=$((fail+1)); say "FAIL $n $name code=$code want-fail=$want cmd=$*"; say "$out"; fi
}
check_no_secret(){
  n=$((n+1)); name="$1"; secret="$2"
  if [ -z "$secret" ]; then pass=$((pass+1)); say "PASS $n $name (no test secret provided)"; return; fi
  if ! grep -R "$secret" . >/dev/null 2>&1; then pass=$((pass+1)); say "PASS $n $name"; else fail=$((fail+1)); say "FAIL $n $name secret found"; fi
}

SECRET="${INSTALLY_TEST_SECRET:-}"
check "go test" go test ./...
check "go build linux host" go build -o instally ./cmd/instally
check "cross windows" env GOOS=windows GOARCH=amd64 go build -o "$TMP/instally-check.exe" ./cmd/instally
check "cross darwin amd64" env GOOS=darwin GOARCH=amd64 go build -o "$TMP/instally-check-darwin" ./cmd/instally
check "cross darwin arm64" env GOOS=darwin GOARCH=arm64 go build -o "$TMP/instally-check-darwin-arm64" ./cmd/instally
check "shell scripts" bash -n install.sh install-full.sh uninstall.sh install-macos.sh build-native.sh native/fyne/scripts/install-linux-build-deps.sh
check_no_secret "api key not embedded" "$SECRET"
check_contains "support has archive safety" "Archive safety" ./instally --support
check_contains "vt status no leak" "VirusTotal" ./instally --vt-status
check_contains "vt test no key" "VirusTotal test" ./instally --vt-test
n=$((n+1)); tmpvt="$(mktemp -d)"; out="$(printf dummy-key | INSTALLY_DATA_DIR="$tmpvt" ./instally --vt-save-key-stdin 2>&1)"; code=$?; if [ $code -eq 0 ] && printf '%s' "$out" | grep -Fq -- "saved" && ! printf '%s' "$out" | grep -Fq -- "dummy-key"; then pass=$((pass+1)); say "PASS $n vt save stdin no echo"; else fail=$((fail+1)); say "FAIL $n vt save stdin no echo code=$code"; say "$out"; fi
check_contains "security test detects EICAR" "OK: test signature" ./instally --security-test
printf '%s' 'X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > "$TMP/eicar.com"
check_fail_contains "scan EICAR blocked" "unsafe" ./instally --scan "$TMP/eicar.com"
printf '#!/bin/sh\ncurl https://example.com/x | bash\n' > "$TMP/install.sh"
check_contains "scan curl pipe bash warning" "download-and-execute" ./instally --scan "$TMP/install.sh" --allow-unknown
printf 'Set-MpPreference -DisableRealtimeMonitoring $true\n' > "$TMP/setup.ps1"
check_contains "scan Defender disable warning" "Defender" ./instally --scan "$TMP/setup.ps1" --allow-unknown
printf '%0700d' 0 | tr '0' 'A' > "$TMP/base64.sh"
check_contains "scan base64 warning" "base64" ./instally --scan "$TMP/base64.sh" --allow-unknown
printf 'MZfake' > "$TMP/invoice.pdf.exe"
check_contains "scan double extension" "двойное" ./instally --scan "$TMP/invoice.pdf.exe" --allow-unknown
: > "$TMP/empty.sh"
check_contains "scan empty file" "пустой" ./instally --scan "$TMP/empty.sh" --allow-unknown
printf 'not elf' > "$TMP/fake.AppImage"
check_contains "scan fake AppImage" "AppImage" ./instally --scan "$TMP/fake.AppImage" --allow-unknown
python3 - <<PY
import zipfile, tarfile, pathlib
root=pathlib.Path('$TMP')
with zipfile.ZipFile(root/'bad.zip','w') as z: z.writestr('../evil.sh','bad')
with zipfile.ZipFile(root/'good.zip','w') as z: z.writestr('app/readme.txt','ok')
with tarfile.open(root/'bad.tar','w') as t:
    p=root/'evil.txt'; p.write_text('bad')
    info=tarfile.TarInfo('../../evil.txt'); info.size=3; t.addfile(info, open(p,'rb'))
with tarfile.open(root/'good.tar','w') as t:
    p=root/'ok.txt'; p.write_text('ok')
    info=tarfile.TarInfo('app/ok.txt'); info.size=2; t.addfile(info, open(p,'rb'))
PY
check_contains "zip traversal warning" "опасный путь" ./instally --scan "$TMP/bad.zip" --allow-unknown
check_contains "zip clean" "zip:" ./instally --scan "$TMP/good.zip" --allow-unknown
check_contains "tar traversal warning" "опасный путь" ./instally --scan "$TMP/bad.tar" --allow-unknown
check_contains "tar clean" "tar:" ./instally --scan "$TMP/good.tar" --allow-unknown
check_fail_contains "private URL blocked" "private/local" ./instally --install-url-safe http://127.0.0.1/app.AppImage --dry-run --yes
check_contains "url dry-run appimage" "app.AppImage" ./instally --install-url-safe https://example.com/app.AppImage --dry-run --yes
check_contains "url dry-run dmg" "App.dmg" env INSTALLY_FORCE_OS=darwin INSTALLY_FORCE_PM=brew ./instally --install-url-safe https://example.com/App.dmg --dry-run --yes

for pm in pacman apt dnf zypper apk xbps eopkg brew winget scoop choco; do
  check_contains "pkg manager $pm" "$pm" env INSTALLY_FORCE_PM="$pm" ./instally --pkg git htop --dry-run --yes
done
for os_pm in "windows winget vscode" "windows scoop git" "windows choco discord" "darwin brew vscode" "darwin port wget" "linux pacman vscode" "linux apt vscode" "linux dnf docker"; do
  set -- $os_pm
  check_contains "forced $1 $2 $3" "$2" env INSTALLY_FORCE_OS="$1" INSTALLY_FORCE_PM="$2" ./instally --text "$3" --dry-run --yes
done
apps=(vscode discord telegram spotify steam obs vlc blender gimp krita firefox brave chrome godot neovim docker node go rust alacritty wezterm kitty obsidian bitwarden keepassxc signal slack zoom postman insomnia github-desktop qbittorrent inkscape kdenlive audacity libreoffice thunderbird mpv yt-dlp lazygit lazydocker onlyoffice localsend stremio)
for app in "${apps[@]}"; do
  check_contains "known app $app" "[" ./instally --text "$app" --dry-run --yes
done
check_contains "multi comma" "com.visualstudio.code" ./instally --multi 'vscode, discord, telegram' --dry-run --yes
check_contains "multi repeated" "cli/cli" ./instally --multi vscode --multi discord --multi 'github:cli/cli' --dry-run --yes
cat > "$TMP/apps.txt" <<TXT
vscode
discord
github: cli/cli
https://example.com/app.AppImage
TXT
check_contains "batch file" "cli/cli" ./instally --batch "$TMP/apps.txt" --dry-run --yes
check_contains "continue on error dry plan" "com.discordapp.Discord" ./instally --multi "vscode,discord" --continue-on-error --dry-run --yes
for t in 'github: cli/cli' 'gh:cli/cli' 'github.com/cli/cli' 'https://github.com/cli/cli' 'https://github.com/cli/cli/releases/latest'; do
  check_contains "github parse $t" "cli/cli" ./instally --text "$t" --dry-run --yes
done
for f in /tmp/a.deb /tmp/a.rpm /tmp/a.pkg.tar.zst /tmp/a.AppImage /tmp/a.msi /tmp/a.exe /tmp/a.dmg /tmp/a.pkg /tmp/a.apk /tmp/source.zip /tmp/source.tar.gz /tmp/source.7z /tmp/setup.run /tmp/setup.bin /tmp/setup.sh; do
  check_contains "local plan $f" "install-local-safe" ./instally --local "$f" --dry-run --yes
done
check_contains "lang en support" "Language: en" env INSTALLY_LANG=en ./instally --support
check_contains "lang ru support" "Language: ru" env INSTALLY_LANG=ru ./instally --support
check_contains "env vt masked status" "настроен" env INSTALLY_VT_API_KEY=dummy ./instally --vt-status
check_contains "force arch arm64 github" "cli/cli" env INSTALLY_FORCE_ARCH=arm64 ./instally --text 'github:cli/cli' --dry-run --yes
check_contains "doctor" "Instally doctor" ./instally --doctor
check_contains "detect json" "family" ./instally --detect
check_contains "app positional" "com.visualstudio.code" ./instally vscode --dry-run --yes
check_contains "release flag" "cli/cli" ./instally --release cli/cli --dry-run --yes
check_contains "url flag" "install-url-safe" ./instally --url https://example.com/app.AppImage --dry-run --yes
check_contains "git flag" "git" ./instally --git https://github.com/cli/cli.git --dry-run --yes
check_contains "npm" "npm" ./instally --npm pnpm --dry-run --yes
check_contains "pipx" "pipx" ./instally --pipx black --dry-run --yes
check_contains "cargo" "cargo" ./instally --cargo ripgrep --dry-run --yes
check_contains "go install" "go" ./instally --go golang.org/x/tools/cmd/stringer --dry-run --yes
say "RESULT passed=$pass failed=$fail total=$n"
[ "$fail" -eq 0 ]
