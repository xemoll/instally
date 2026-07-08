#!/usr/bin/env sh
set -eu
rm -f "$HOME/.local/bin/instally" "$HOME/.local/share/applications/instally.desktop" "$HOME/.local/share/mime/packages/instally.xml"
update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
update-mime-database "$HOME/.local/share/mime" 2>/dev/null || true
echo "Instally removed from user menu. Cache/data kept."
