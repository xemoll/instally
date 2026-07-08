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

build_native() {
  local os="$1" arch="$2" ext=""
  [ "$os" = "windows" ] && ext=".exe"
  local dir="$OUTDIR/${os}-${arch}"

  if [ -d native/fyne ]; then
    local nbinary="${dir}/instally-native${ext}"
    if [ "$os" = "linux" ] || [ "$os" = "darwin" ]; then
      echo "  building native GUI ${os}/${arch}..."
      cd native/fyne
      GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -ldflags="-s -w" -o "../../${nbinary}" .
      cd "$ROOT"
    fi
  fi
}

# Build all targets
build linux amd64
build linux arm64
build windows amd64
build darwin amd64
build darwin arm64

# Optional: native GUI (Fyne) — only on linux/darwin
build_native linux amd64
build_native darwin amd64
build_native darwin arm64

echo ""

# Package into archives
echo "== Packaging =="
for dir in "$OUTDIR"/*/; do
  os_arch="$(basename "$dir")"

  if [[ "$os_arch" == windows-* ]]; then
    (cd "$OUTDIR" && zip -qr "instally-${os_arch}.zip" "$os_arch")
    echo "  instally-${os_arch}.zip"
  else
    tar -C "$OUTDIR" -czf "$OUTDIR/../instally-${os_arch}.tar.gz" "$os_arch"
    echo "  instally-${os_arch}.tar.gz"
  fi
done

echo ""

# Checksums
echo "== Checksums =="
(cd "$OUTDIR" && sha256sum ./*.tar.gz ./*.zip ./*.sha256 > SHA256SUMS.txt 2>/dev/null || true)
sha256sum "$OUTDIR"/*.tar.gz "$OUTDIR"/*.zip 2>/dev/null

echo ""
echo "== Done =="
echo "Output: $OUTDIR"
echo "Version: $VERSION"
