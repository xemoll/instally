#!/usr/bin/env sh
set -eu
REPO="xemoll/instally"

detect_os_arch() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    armv6l) arch="armv6" ;;
    armv7l) arch="armv7" ;;
    386|i386|i686) arch="386" ;;
  esac
  case "$os" in
    linux) echo "linux-$arch" ;;
    darwin) echo "darwin-$arch" ;;
    mingw*|msys*|cygwin*) echo "windows-$arch.exe" ;;
    *) echo "unsupported: $os $arch" >&2; exit 1 ;;
  esac
}

dl() {
  if command -v curl >/dev/null 2>&1; then
    curl -fL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$1" -O "$2"
  else
    echo "need curl or wget" >&2; exit 1
  fi
}

echo "Detecting latest Instally release..."
if command -v curl >/dev/null 2>&1; then
  tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
elif command -v wget >/dev/null 2>&1; then
  tag=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
else
  tag="v1.2.0"
fi
if [ -z "$tag" ]; then tag="v1.2.0"; fi

asset=$(detect_os_arch)
url="https://github.com/$REPO/releases/download/$tag/instally-$asset"
sums_url="https://github.com/$REPO/releases/download/$tag/SHA256SUMS.txt"

echo "Downloading Instally $tag for $asset..."
dl "$url" /tmp/instally
chmod +x /tmp/instally

echo "Verifying SHA256..."
dl "$sums_url" /tmp/instally.sha256
if command -v sha256sum >/dev/null 2>&1; then
  if sha256sum -c /tmp/instally.sha256 --ignore-missing 2>/dev/null; then
    echo "Checksum OK"
  else
    echo "Checksum mismatch!" >&2
    rm -f /tmp/instally /tmp/instally.sha256
    exit 1
  fi
elif command -v shasum >/dev/null 2>&1; then
  if shasum -a 256 -c /tmp/instally.sha256 --ignore-missing 2>/dev/null; then
    echo "Checksum OK"
  else
    echo "Checksum mismatch!" >&2
    rm -f /tmp/instally /tmp/instally.sha256
    exit 1
  fi
else
  echo "warning: no sha256sum/shasum available, skipping verification"
fi
rm -f /tmp/instally.sha256

exec /tmp/instally --install-self --set-default-installer --yes
