#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
OUTDIR="${2:-dist/release}"
mkdir -p "$OUTDIR"

echo "== Instally Release Builder v$VERSION =="
echo ""

build() {
  local os="$1" arch="$2" ext=""
  [ "$os" = "windows" ] && ext=".exe"
  local dir="$OUTDIR/${os}-${arch}"
  mkdir -p "$dir"

  local binary="${dir}/instally${ext}"
  echo "  building ${os}/${arch}..."

  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags='-s -w -X main.version='"$VERSION" \
    -o "$binary" ./cmd/instally

  if [ -f "$binary" ]; then
    sha256sum "$binary" | awk '{print $1}' > "${binary}.sha256"
    echo "    ok  $(ls -lh "$binary" | awk '{print $5}')"
  fi
}

# Build all targets
build linux amd64
build linux arm64
build windows amd64
build darwin amd64
build darwin arm64

echo ""

# Package into archives
echo "== Packaging =="
mkdir -p "$ROOT/dist/archives"
for dir in "$OUTDIR"/*/; do
  os_arch="$(basename "$dir")"

  if [[ "$os_arch" == windows-* ]]; then
    zip_name="instally-${os_arch}.zip"
    (cd "$OUTDIR" && zip -qr "$ROOT/dist/archives/${zip_name}" "$os_arch")
    echo "  $zip_name"
  else
    tar_name="instally-${os_arch}.tar.gz"
    tar -C "$OUTDIR" -czf "$ROOT/dist/archives/${tar_name}" "$os_arch"
    echo "  $tar_name"
  fi
done

echo ""

# Checksums
echo "== Checksums =="
(cd "$ROOT/dist/archives" && sha256sum ./*.tar.gz ./*.zip > SHA256SUMS.txt)
cat "$ROOT/dist/archives/SHA256SUMS.txt"

echo ""
echo "== Done =="
echo "Output: $OUTDIR"
echo "Version: $VERSION"
