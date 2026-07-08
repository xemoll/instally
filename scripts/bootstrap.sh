#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass=0
fail=0
check() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo -e "  ${GREEN}ok${NC}  $name"
    pass=$((pass+1))
  else
    echo -e "  ${RED}FAIL${NC} $name"
    fail=$((fail+1))
  fi
}

echo -e "${CYAN}== Instally Bootstrap ==${NC}"
echo ""

echo -e "${YELLOW}Dependencies:${NC}"
check "go installed"    go version
check "git installed"   git version
check "gcc installed"   gcc --version

echo ""
echo -e "${YELLOW}Go module:${NC}"
check "go mod tidy"     go mod tidy
check "go vet"          go vet ./...

echo ""
echo -e "${YELLOW}Tests:${NC}"
check "go test"         go test ./... -count=1 -timeout=180s

echo ""
echo -e "${YELLOW}Build:${NC}"
check "build"           go build -trimpath -o /tmp/instally-bootstrap ./cmd/instally

echo ""
echo -e "${YELLOW}Security self-test:${NC}"
check "security test"   /tmp/instally-bootstrap --security-test

echo ""
echo -e "${YELLOW}Shell scripts syntax:${NC}"
check "install.sh"      bash -n install.sh
check "install-full.sh" bash -n install-full.sh
check "uninstall.sh"    bash -n uninstall.sh
check "install-macos.sh" bash -n install-macos.sh
check "build-native.sh" bash -n build-native.sh
for s in scripts/*.sh; do
  check "$(basename "$s")" bash -n "$s"
done

echo ""
echo -e "${YELLOW}Doctor:${NC}"
/tmp/instally-bootstrap --doctor || true

echo ""
echo -e "${CYAN}==============================${NC}"
echo -e "  ${GREEN}passed:${NC} $pass  ${RED}failed:${NC} $fail  ${NC}total: $((pass+fail))"

if [ "$fail" -eq 0 ]; then
  echo -e "  ${GREEN}All checks passed${NC}"
else
  echo -e "  ${RED}Some checks failed${NC}"
  exit 1
fi
