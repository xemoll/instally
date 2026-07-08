#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 1

LOG="${1:-advanced-checks.log}"
: > "$LOG"

BOLD='\033[1m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
use_color() { [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; }

say() {
  local msg="$1"
  echo "$msg" >> "$LOG"
  if use_color; then echo -e "$msg"; else echo "$msg" | sed 's/\x1b\[[0-9;]*m//g'; fi
}

PASS=0; FAIL=0
run() {
  local name="$1"; shift
  local start elapsed
  start=$(date +%s%N)
  echo "### $name" >> "$LOG"
  if timeout 60s "$@" >> "$LOG" 2>&1; then
    elapsed=$(( ($(date +%s%N) - start) / 1000000 ))
    say "  ${GREEN}ok${NC}   ${name} (${elapsed}ms)"
    PASS=$((PASS+1))
  else
    elapsed=$(( ($(date +%s%N) - start) / 1000000 ))
    say "  ${RED}FAIL${NC} ${name} (${elapsed}ms)"
    FAIL=$((FAIL+1))
  fi
}
run_sh() {
  local name="$1"; shift
  local start elapsed
  start=$(date +%s%N)
  echo "### $name" >> "$LOG"
  if timeout 60s bash -lc "$*" >> "$LOG" 2>&1; then
    elapsed=$(( ($(date +%s%N) - start) / 1000000 ))
    say "  ${GREEN}ok${NC}   ${name} (${elapsed}ms)"
    PASS=$((PASS+1))
  else
    elapsed=$(( ($(date +%s%N) - start) / 1000000 ))
    say "  ${RED}FAIL${NC} ${name} (${elapsed}ms)"
    FAIL=$((FAIL+1))
  fi
}

BIN="${BIN:-./instally}"

say "${CYAN}== Build & Test ==${NC}"
run go-test       go test ./... -count=1 -timeout=180s
run build-linux   go build -o instally ./cmd/instally

say "${CYAN}== Cross-compilation ==${NC}"
for os in windows darwin linux; do
  for arch in amd64 arm64; do
    [ "$os-$arch" = "windows-arm64" ] && continue
    out="/tmp/instally-$os-$arch"
    [ "$os" = windows ] && out="$out.exe"
    run "cross-$os-$arch" env GOOS="$os" GOARCH="$arch" go build -trimpath -o "$out" ./cmd/instally
  done
done

say "${CYAN}== Shell syntax ==${NC}"
run shell-syntax bash -n install.sh install-full.sh uninstall.sh install-macos.sh build-native.sh native/fyne/scripts/install-linux-build-deps.sh
for s in scripts/*.sh; do
  run "syntax-$(basename "$s")" bash -n "$s"
done

say "${CYAN}== Core diagnostics ==${NC}"
run doctor        "$BIN" --doctor
run support       "$BIN" --support
run vt-status     "$BIN" --vt-status
run vt-test       "$BIN" --vt-test
run security-test "$BIN" --security-test

say "${CYAN}== Language support ==${NC}"
for lang in ru en; do
  run "support-lang-$lang" env INSTALLY_LANG="$lang" "$BIN" --support
done

say "${CYAN}== Package managers (all 10) ==${NC}"
for pm in pacman apt dnf zypper apk xbps eopkg brew winget scoop choco; do
  run "pkg-$pm"       env INSTALLY_FORCE_PM="$pm" "$BIN" --pkg git htop --dry-run --yes
  run "multi-$pm"     env INSTALLY_FORCE_PM="$pm" "$BIN" --multi "vscode,discord,telegram" --dry-run --yes
done

say "${CYAN}== OS-forced tests ==${NC}"
for os in linux windows darwin; do
  case "$os" in
    linux)   pms="pacman apt dnf apk";;
    windows) pms="winget scoop choco";;
    darwin)  pms="brew port";;
  esac
  for pm in $pms; do
    run "force-$os-$pm-url"  env INSTALLY_FORCE_OS="$os" INSTALLY_FORCE_PM="$pm" "$BIN" --install-url-safe https://example.com/app.AppImage --dry-run --yes
  done
done

say "${CYAN}== Item resolution ==${NC}"
for item in vscode discord telegram github:cli/cli gh:sharkdp/fd https://example.com/app.AppImage "local: /tmp/test.AppImage"; do
  run "text-$item" "$BIN" --text "$item" --dry-run --yes --allow-unknown
done

say "${CYAN}== Local file plans ==${NC}"
for f in /tmp/a.AppImage /tmp/a.deb /tmp/a.rpm /tmp/a.pkg.tar.zst /tmp/a.msi /tmp/a.exe /tmp/a.dmg /tmp/a.pkg /tmp/a.zip /tmp/a.7z /tmp/a.run /tmp/a.sh; do
  run "local-$f" "$BIN" --local "$f" --dry-run --yes --allow-unknown
done

say "${CYAN}== Security checks ==${NC}"
TMPD=$(mktemp -d)
trap 'rm -rf "$TMPD"' EXIT
cat > "$TMPD/eicar.com" <<'EICAR'
X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*
EICAR
run_sh "scan-eicar-block" "! $BIN --scan '$TMPD/eicar.com'"
run_sh "private-url-blocked" "! $BIN --install-url-safe http://127.0.0.1/a.AppImage --dry-run --yes"

cat > "$TMPD/apps.txt" <<'APPS'
vscode
discord
telegram
github: cli/cli
https://example.com/app.AppImage
APPS
run "batch-apps" "$BIN" --batch "$TMPD/apps.txt" --dry-run --yes --allow-unknown --continue-on-error

say "${CYAN}== Long multi install ==${NC}"
run "multi-long" "$BIN" --multi "vscode,discord,telegram,obs,vlc,blender,gimp,krita,firefox,brave,chrome,godot,neovim,docker,node,go,rust" --dry-run --yes --allow-unknown --continue-on-error

# Secret leak check
run_sh "no-leaked-key" 'if [ -n "${INSTALLY_FORBIDDEN_TEST_SECRET:-}" ]; then ! grep -R "$INSTALLY_FORBIDDEN_TEST_SECRET" . --exclude-dir=.git --exclude=instally --exclude="*.exe"; else true; fi'

say ""
say "${CYAN}================================${NC}"
say "  ${GREEN}passed:${NC} $PASS  ${RED}failed:${NC} $FAIL  total: $((PASS+FAIL))"
[ "$FAIL" -eq 0 ]
