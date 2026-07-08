#!/usr/bin/env bash
set -euo pipefail
if command -v pacman >/dev/null 2>&1; then
  sudo pacman -S --needed go gcc pkgconf libglvnd libx11 libxcursor libxrandr libxinerama libxi mesa
elif command -v apt >/dev/null 2>&1; then
  sudo apt update
  sudo apt install -y golang-go gcc pkg-config libgl1-mesa-dev xorg-dev
elif command -v dnf >/dev/null 2>&1; then
  sudo dnf install -y golang gcc pkgconf-pkg-config mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
elif command -v zypper >/dev/null 2>&1; then
  sudo zypper install -y go gcc pkg-config Mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel
else
  echo "Unknown package manager. Install Go, gcc, pkg-config, X11 development headers and Mesa/OpenGL development headers manually." >&2
  exit 1
fi
