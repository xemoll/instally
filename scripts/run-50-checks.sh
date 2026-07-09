#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/instally"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

LOG="${1:-}"
REBUILD="${2:-}"
[ -n "$LOG" ] && : > "$LOG"

BOLD='\033[1m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
use_color() { [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; }

say() {
  local msg="$1"
  [ -n "$LOG" ] && echo "$msg" >> "$LOG"
  if use_color; then echo -e "$msg"; else echo "$msg" | sed 's/\x1b\[[0-9;]*m//g'; fi
}
ok()   { say "${GREEN}ok${NC}   $1"; }
fail() { say "${RED}FAIL${NC} $1"; }

CKOUT="$TMP/check.out"
CKERR="$TMP/check.err"
run_ok(){
  local name="$1"; shift
  echo "### $name" >> "$LOG"
  if "$@" >"$CKOUT" 2>"$CKERR"; then
    ok "$name"
    pass=$((pass+1))
  else
    fail "$name"
    cat "$CKOUT" "$CKERR" | head -20 >> "$LOG"
    fail=$((fail+1))
  fi
}
run_fail(){
  local name="$1"; shift
  echo "### $name" >> "$LOG"
  if "$@" >"$CKOUT" 2>"$CKERR"; then
    fail "${name} (expected failure)"
    cat "$CKOUT" "$CKERR" | head -10 >> "$LOG"
    fail=$((fail+1))
  else
    ok "$name"
    pass=$((pass+1))
  fi
}

pass=0
fail=0

# Build if needed
if [ -n "$REBUILD" ] || [ ! -x "$BIN" ]; then
  say "${YELLOW}rebuilding...${NC}"
  go build -o "$BIN" ./cmd/instally || { fail "build failed"; exit 1; }
fi

say "${CYAN}== Core checks ==${NC}"
run_ok detect            "$BIN" --detect
run_ok doctor            "$BIN" --doctor
run_ok support           "$BIN" --support
run_ok vt-status         "$BIN" --vt-status
run_ok lang-en           "$BIN" --lang en --support
run_ok lang-ru           "$BIN" --lang ru --support
run_ok security-self-test "$BIN" --security-test

say "${CYAN}== Security scans ==${NC}"
printf 'hello\n' > "$TMP/safe.txt"
printf '#!/bin/sh\necho safe\n' > "$TMP/setup.AppImage"
printf 'X5O!P%%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > "$TMP/eicar.com"
run_ok scan-safe          "$BIN" --scan "$TMP/safe.txt"
run_fail scan-eicar-blocks "$BIN" --scan "$TMP/eicar.com"
run_ok local-safe-dryrun  "$BIN" --install-local-safe "$TMP/setup.AppImage" --allow-unknown --dry-run --yes
run_ok url-safe-dryrun    "$BIN" --install-url-safe 'https://example.com/app.AppImage' --dry-run --yes
run_fail private-url-blocked "$BIN" --install-url-safe 'http://127.0.0.1/app.AppImage' --dry-run --yes

say "${CYAN}== Multi install ==${NC}"
run_ok multi-comma        "$BIN" --multi 'vscode,discord,github:cli/cli' --dry-run --yes
run_ok multi-repeat       "$BIN" --multi vscode --multi discord --multi 'https://example.com/app.AppImage' --dry-run --yes

say "${CYAN}== Input formats ==${NC}"
run_ok text-batch         "$BIN" --text $'pkg: git htop\nflatpak: com.visualstudio.code\ngithub: cli/cli' --dry-run --yes
cat > "$TMP/list.txt" <<LIST
vscode
discord
github: cli/cli
https://example.com/app.AppImage
LIST
run_ok batch-file         "$BIN" --batch "$TMP/list.txt" --dry-run --yes
run_ok positional         "$BIN" vscode discord --dry-run --yes
run_ok github-owner-repo  "$BIN" --text 'cli/cli' --dry-run --yes
run_ok github-url         "$BIN" --text 'github: https://github.com/cli/cli/releases/latest' --dry-run --yes
run_ok url-task           "$BIN" --url 'https://example.com/file.deb' --dry-run --yes
run_ok local-appimage     "$BIN" --local "$TMP/setup.AppImage" --dry-run --yes

say "${CYAN}== Package managers (dry-run) ==${NC}"
run_ok local-deb          "$BIN" --local '/tmp/app.deb' --dry-run --yes
run_ok local-rpm          "$BIN" --local '/tmp/app.rpm' --dry-run --yes
run_ok local-7z           "$BIN" --local '/tmp/source.7z' --dry-run --yes
run_ok local-dmg          "$BIN" --local '/tmp/App.dmg' --dry-run --yes
run_ok local-msi          "$BIN" --local 'C:\\Temp\\setup.msi' --dry-run --yes
run_ok pipx               "$BIN" --pipx black --dry-run --yes
run_ok npm                "$BIN" --npm pnpm --dry-run --yes
run_ok cargo              "$BIN" --cargo ripgrep --dry-run --yes
run_ok goinstall          "$BIN" --go golang.org/x/tools/cmd/stringer@latest --dry-run --yes
run_ok flatpak            "$BIN" --flatpak org.telegram.desktop --dry-run --yes
run_ok snap               "$BIN" --snap discord --dry-run --yes
run_ok aur-pacman         env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=pacman "$BIN" --aur yay-bin --dry-run --yes

say "${CYAN}== OS package managers ==${NC}"
run_ok pm-pacman          env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=pacman "$BIN" --pkg git --dry-run --yes
run_ok pm-apt             env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=apt "$BIN" --pkg git --dry-run --yes
run_ok pm-dnf             env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=dnf "$BIN" --pkg git --dry-run --yes
run_ok pm-zypper          env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=zypper "$BIN" --pkg git --dry-run --yes
run_ok pm-apk             env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=apk "$BIN" --pkg git --dry-run --yes
run_ok pm-xbps            env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=xbps "$BIN" --pkg git --dry-run --yes
run_ok pm-eopkg           env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=eopkg "$BIN" --pkg git --dry-run --yes
run_ok pm-emerge          env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=emerge "$BIN" --pkg git --dry-run --yes
run_ok pm-nix             env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=nix "$BIN" --pkg git --dry-run --yes
run_ok pm-packagekit      env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=packagekit "$BIN" --pkg git --dry-run --yes
run_ok pm-brew-linux      env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=brew "$BIN" --pkg git --dry-run --yes

say "${CYAN}== Cross-platform tests ==${NC}"
run_ok win-winget         env INSTALLY_FORCE_OS=windows INSTALLY_FORCE_PM=winget "$BIN" --text 'vscode discord vlc' --dry-run --yes
run_ok win-scoop          env INSTALLY_FORCE_OS=windows INSTALLY_FORCE_PM=scoop "$BIN" --pkg git ripgrep --dry-run --yes
run_ok win-choco          env INSTALLY_FORCE_OS=windows INSTALLY_FORCE_PM=choco "$BIN" --pkg git ripgrep --dry-run --yes
run_ok mac-brew           env INSTALLY_FORCE_OS=darwin INSTALLY_FORCE_PM=brew "$BIN" --text 'vscode discord' --dry-run --yes
run_ok mac-port           env INSTALLY_FORCE_OS=darwin INSTALLY_FORCE_PM=port "$BIN" --pkg wget --dry-run --yes
run_ok force-arm64        env INSTALLY_FORCE_ARCH=arm64 "$BIN" --text 'github: cli/cli' --dry-run --yes
run_ok unknown-manager    env INSTALLY_FORCE_OS=linux INSTALLY_FORCE_PM=unknownpm "$BIN" --pkg git --dry-run --yes

say ""
say "${CYAN}================================${NC}"
say "  ${GREEN}passed:${NC} $pass  ${RED}failed:${NC} $fail  total: $((pass+fail))"
[ "$pass" -ge 50 ] && [ "$fail" -eq 0 ]
