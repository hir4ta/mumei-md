#!/bin/sh
# install.sh: bootstrap the miru installer.
#
# Downloads a prebuilt `miru` binary and hands off to its `install`
# subcommand, which presents a rich Bubble Tea install UI and performs
# self-installation + PATH configuration.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hir4ta/miru/main/install.sh | sh
#
# Environment overrides (passed through to `miru install`):
#   VERSION                pin a release tag, e.g. v0.1.0 (default: latest)
#   INSTALL_DIR            target install directory (default: $HOME/.local/bin)
#   MIRU_NO_MODIFY_PATH   set to 1 to skip shell rc PATH update
#   MIRU_THEME            color theme for the installer UI

set -eu

REPO="hir4ta/miru"
VERSION="${VERSION:-}"

err() { printf 'install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"; }

need curl
need tar
need uname
need awk

# sha256 utility differs by OS; require one of the standard pair.
if command -v sha256sum >/dev/null 2>&1; then
	SHA256="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	SHA256="shasum -a 256"
else
	err "required command not found: sha256sum or shasum"
fi

case "$(uname -s)" in
	Linux)  OS=linux  ;;
	Darwin) OS=darwin ;;
	*) err "unsupported OS: $(uname -s)" ;;
esac

case "$(uname -m)" in
	x86_64|amd64)  ARCH=amd64 ;;
	arm64|aarch64) ARCH=arm64 ;;
	*) err "unsupported architecture: $(uname -m)" ;;
esac

if [ -z "$VERSION" ]; then
	redirect_url=$(curl -fsSI -o /dev/null -w '%{redirect_url}' \
		"https://github.com/${REPO}/releases/latest" || true)
	VERSION="${redirect_url##*/}"
	[ -n "$VERSION" ] || err "could not resolve latest release for ${REPO}"
fi

version_no_v="${VERSION#v}"
asset="miru_${version_no_v}_${OS}_${ARCH}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

curl -fsSL --proto '=https' --tlsv1.2 "$url" -o "$tmp/$asset" \
	|| err "download failed: $url"

curl -fsSL --proto '=https' --tlsv1.2 "$checksums_url" -o "$tmp/checksums.txt" \
	|| err "checksum download failed: $checksums_url"

expected_sha=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$expected_sha" ] || err "no checksum entry for ${asset}"

actual_sha=$($SHA256 "$tmp/$asset" | awk '{print $1}')
[ "$expected_sha" = "$actual_sha" ] || err "checksum mismatch for ${asset} (expected ${expected_sha}, got ${actual_sha})"

tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/miru" ] || err "archive did not contain expected binary"
chmod +x "$tmp/miru"

# Hand off to the rich Bubble Tea installer.
"$tmp/miru" install
