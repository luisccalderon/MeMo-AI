#!/bin/sh
# Build this checkout and install the memo-ai binary.
#
# Usage:
#   ./scripts/install.sh
#   PREFIX=/usr/local ./scripts/install.sh
#
# Environment:
#   PREFIX  Install prefix (default: ~/.local)
#   GO      Go binary to use (default: go)
set -eu

PREFIX="${PREFIX:-$HOME/.local}"
GO="${GO:-go}"
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

if ! command -v "$GO" >/dev/null 2>&1; then
	echo "error: Go is required to build memo-ai (set GO=/path/to/go)." >&2
	exit 1
fi

VERSION=$(cd "$ROOT" && git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Building memo-ai ($VERSION)..."
(cd "$ROOT" && "$GO" build -trimpath \
	-ldflags "-s -w -X github.com/luisccalderon/MeMo-AI/internal/version.Version=$VERSION" \
	-o "$TMP/memo-ai" ./cmd/memo-ai)

install -d "$PREFIX/bin"
install -m 0755 "$TMP/memo-ai" "$PREFIX/bin/memo-ai"

echo "Installed $PREFIX/bin/memo-ai"
case ":${PATH}:" in
	*":$PREFIX/bin:"*) ;;
	*) echo "note: add $PREFIX/bin to PATH to run memo-ai directly." ;;
esac
echo "Next: memo-ai install"
