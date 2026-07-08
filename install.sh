#!/usr/bin/env sh
set -eu
DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ -x "$DIR/instally" ]; then BIN="$DIR/instally"; elif [ -x "$DIR/dist/linux-amd64/instally" ]; then BIN="$DIR/dist/linux-amd64/instally"; else echo "Build first: go build -o instally ./cmd/instally" >&2; exit 1; fi
"$BIN" --install-self --yes
