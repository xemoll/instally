#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT="${1:-coverage}"
mkdir -p "$OUT"

echo "== Instally Coverage =="
echo ""

echo "Running tests with coverage..."
go test ./... -count=1 -timeout=300s -coverprofile="$OUT/coverage.out" -covermode=atomic 2>&1 | tail -20

echo ""
echo "Generating HTML report..."
go tool cover -html="$OUT/coverage.out" -o "$OUT/coverage.html"

echo ""
echo "Coverage summary:"
go tool cover -func="$OUT/coverage.out" | tail -30

echo ""
echo "Total coverage:"
go tool cover -func="$OUT/coverage.out" | grep "^total:" | awk '{print $3}'

echo ""
echo "Reports:"
echo "  $OUT/coverage.out"
echo "  $OUT/coverage.html"

# Open in browser if on desktop
if [ -n "${DISPLAY:-}" ] && command -v xdg-open &>/dev/null; then
  xdg-open "$OUT/coverage.html" 2>/dev/null || true
fi
