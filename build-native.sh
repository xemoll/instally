#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT/native/fyne"
GOOS_VAL="$(go env GOOS)"
GOARCH_VAL="$(go env GOARCH)"
OUT_DIR="$ROOT/dist/${GOOS_VAL}-${GOARCH_VAL}"
mkdir -p "$OUT_DIR"
go mod tidy
go build -trimpath -o "$OUT_DIR/instally-native" .
echo "built: $OUT_DIR/instally-native"
