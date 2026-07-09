#!/usr/bin/env sh
set -eu
REPO="xemoll/instally"
VERSION="v1.2.0"
URL="https://github.com/$REPO/releases/download/$VERSION"

detect_os_arch() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    armv6l|armv7l)
      case "$arch" in
        armv6l) arch="armv6" ;;
        armv7l) arch="armv7" ;;
      esac ;;
    386|i386|i686) arch="386" ;;
  esac
  case "$os" in
    linux) echo "linux-$arch" ;;
    darwin) echo "darwin-$arch" ;;
    mingw*|msys*|cygwin*) echo "windows-$arch.exe" ;;
    *) echo "unsupported: $os $arch" >&2; exit 1 ;;
  esac
}

asset=$(detect_os_arch)
echo "Downloading Instally $VERSION for $asset..."
curl -fL "$URL/instally-$asset" -o /tmp/instally
chmod +x /tmp/instally
exec /tmp/instally --install-self --set-default-installer --yes
