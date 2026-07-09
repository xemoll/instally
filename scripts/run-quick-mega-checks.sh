#!/usr/bin/env bash
set -u
PASS=0; FAIL=0; LOG=${1:-quick-mega-checks.log}; : > "$LOG"; BIN=${BIN:-./instally}
run(){ local name="$1"; shift; echo "### $name" >> "$LOG"; if timeout 20s "$@" >> "$LOG" 2>&1; then echo "PASS $name" >> "$LOG"; PASS=$((PASS+1)); else echo "FAIL $name" >> "$LOG"; FAIL=$((FAIL+1)); fi; }
run_sh(){ local name="$1"; shift; echo "### $name" >> "$LOG"; if timeout 20s bash -lc "$*" >> "$LOG" 2>&1; then echo "PASS $name" >> "$LOG"; PASS=$((PASS+1)); else echo "FAIL $name" >> "$LOG"; FAIL=$((FAIL+1)); fi; }
run go-test go test ./...
run build go build -o instally ./cmd/instally
for pm in pacman apt dnf zypper apk xbps eopkg brew winget scoop choco; do
  run "pkg-$pm" env INSTALLY_FORCE_PM=$pm "$BIN" --pkg git curl --dry-run --yes
  run "multi-$pm" env INSTALLY_FORCE_PM=$pm "$BIN" --multi "vscode,discord,telegram" --dry-run --yes --allow-unknown
 done
for os in linux windows darwin; do
 case "$os" in linux) pms="pacman apt dnf apk";; windows) pms="winget scoop choco";; darwin) pms="brew port";; esac
 for pm in $pms; do
  run "force-$os-$pm-apps" env INSTALLY_FORCE_OS=$os INSTALLY_FORCE_PM=$pm "$BIN" --multi "vscode,obs,vlc,brave" --dry-run --yes --allow-unknown
 done
done
for item in vscode discord telegram obs vlc blender gimp krita firefox brave chrome godot neovim docker node go rust github:cli/cli gh:sharkdp/fd https://example.com/app.AppImage; do
 run "auto-$item" "$BIN" --text "$item" --dry-run --yes --allow-unknown
 done
for f in /tmp/a.AppImage /tmp/a.deb /tmp/a.rpm /tmp/a.pkg.tar.zst /tmp/a.msi /tmp/a.exe /tmp/a.dmg /tmp/a.pkg /tmp/a.zip /tmp/a.7z /tmp/a.run /tmp/a.sh; do
 run "local-$f" "$BIN" --local "$f" --dry-run --yes --allow-unknown
 done
TMPD=$(mktemp -d)
trap 'rm -rf "$TMPD"' EXIT INT TERM
cat > "$TMPD/eicar.com" <<'EICAR'
X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*
EICAR
run_sh scan-eicar-block "! $BIN --scan '$TMPD/eicar.com'"
run security-test "$BIN" --security-test
run vt-status "$BIN" --vt-status
run vt-test-no-key "$BIN" --vt-test
run support "$BIN" --support
run doctor "$BIN" --doctor
run_sh private-url-blocked "! $BIN --install-url-safe http://127.0.0.1/a.AppImage --dry-run --yes"
cat > "$TMPD/apps.txt" <<'APPS'
vscode
discord
telegram
github: cli/cli
https://example.com/app.AppImage
APPS
run batch-apps "$BIN" --batch "$TMPD/apps.txt" --dry-run --yes --allow-unknown --continue-on-error
run_sh no-leaked-key 'if [ -n "${INSTALLY_FORBIDDEN_TEST_SECRET:-}" ]; then ! grep -R "$INSTALLY_FORBIDDEN_TEST_SECRET" . --exclude-dir=.git --exclude=instally --exclude="*.exe" --exclude="*.log"; else true; fi'
echo "passed=$PASS failed=$FAIL total=$((PASS+FAIL))" | tee -a "$LOG"
[ "$FAIL" -eq 0 ]
