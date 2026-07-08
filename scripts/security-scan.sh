#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

issues=0

check_secret() {
  local pattern="$1" desc="$2"
  if grep -rI "$pattern" . --exclude-dir=.git --exclude="*.log" --exclude="*.png" --exclude="*.exe" --exclude-dir=vendor 2>/dev/null | grep -v 'os\.Getenv\|os\.Setenv\|viper\|PasswordEntry\|env\b' | grep -v 'nolint'; then
    echo -e "  ${YELLOW}possible $desc found${NC}"
    issues=$((issues+1))
  else
    echo -e "  ${GREEN}clean${NC} $desc"
  fi
}

check_file() {
  local path="$1" desc="$2"
  if [ -f "$path" ]; then
    echo -e "  ${YELLOW}exists${NC} $desc: $path"
  fi
}

echo "== Instally Security Scan =="
echo ""

echo "Sensitive patterns in source:"
check_secret 'api.?key'             'API key'
check_secret 'ghp_|gho_|ghu_|ghs_' 'GitHub token'
check_secret 'sk-[a-zA-Z0-9]{20,}' 'OpenAI key'
check_secret 'INSTALLY_VT_API_KEY'  'VT key in code'
check_secret '-----BEGIN.*PRIVATE KEY-----' 'private key'
check_secret 'password\s*='         'password assignment'
check_secret 'token\s*='            'token assignment'

echo ""
echo "Sensitive files:"
check_file '.env'            '.env file'
check_file '*.key'           'key files'
check_file 'secrets/'        'secrets dir'
check_file 'credentials'     'credentials file'

echo ""
echo "Compiled binary check:"
if [ -f "$ROOT/instally" ] && file "$ROOT/instally" | grep -q "ELF\|Mach-O\|PE"; then
  echo -e "  ${YELLOW}warning${NC} compiled binary present: instally"
  echo "    Add to .gitignore or remove before commit"
fi

echo ""
echo "Unnecessary files:"
for f in *.log *.tmp *.part; do
  [ -f "$f" ] && echo -e "  ${YELLOW}extra file${NC} $f"
done

echo ""
if [ "$issues" -eq 0 ]; then
  echo -e "${GREEN}No security issues found${NC}"
else
  echo -e "${RED}$issues potential issue(s) found${NC}"
fi
