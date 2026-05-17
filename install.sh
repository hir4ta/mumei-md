#!/bin/sh
# install.sh: install the mumei-md `mm` binary from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hir4ta/mumei-md/main/install.sh | sh
#
# Environment overrides:
#   VERSION                pin a release tag, e.g. v0.1.0 (default: latest)
#   INSTALL_DIR            target install directory (default: $HOME/.local/bin)
#   MUMEI_NO_MODIFY_PATH   set to 1 to skip shell rc PATH update

set -eu

REPO="hir4ta/mumei-md"
BIN="mm"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"
NO_MODIFY_PATH="${MUMEI_NO_MODIFY_PATH:-0}"

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
	BOLD=$(printf '\033[1m')
	GREEN=$(printf '\033[32m')
	YELLOW=$(printf '\033[33m')
	RED=$(printf '\033[31m')
	DIM=$(printf '\033[2m')
	RESET=$(printf '\033[0m')
else
	BOLD=""; GREEN=""; YELLOW=""; RED=""; DIM=""; RESET=""
fi

step() { printf '%s==>%s %s\n' "$BOLD" "$RESET" "$*"; }
ok()   { printf '    %s%s%s\n' "$GREEN" "$*" "$RESET"; }
warn() { printf '%swarn:%s %s\n' "$YELLOW" "$RESET" "$*" >&2; }
die()  { printf '%serror:%s %s\n' "$RED" "$RESET" "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"; }
need curl
need tar
need uname

case "$(uname -s)" in
	Linux)  OS=linux  ;;
	Darwin) OS=darwin ;;
	*) die "unsupported OS: $(uname -s)" ;;
esac

case "$(uname -m)" in
	x86_64|amd64)  ARCH=amd64 ;;
	arm64|aarch64) ARCH=arm64 ;;
	*) die "unsupported architecture: $(uname -m)" ;;
esac

if [ -z "$VERSION" ]; then
	step "resolving latest release"
	redirect_url=$(curl -fsSI -o /dev/null -w '%{redirect_url}' \
		"https://github.com/${REPO}/releases/latest" || true)
	VERSION="${redirect_url##*/}"
	[ -n "$VERSION" ] || die "could not resolve latest release for ${REPO}"
fi
ok "version ${VERSION}"

version_no_v="${VERSION#v}"
asset="mm_${version_no_v}_${OS}_${ARCH}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

step "downloading ${asset}"
curl -fsSL --proto '=https' --tlsv1.2 "$url" -o "$tmp/$asset" \
	|| die "download failed: $url"

step "extracting"
tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/$BIN" ] || die "archive did not contain expected binary: $BIN"

step "installing to ${INSTALL_DIR}"
mkdir -p "$INSTALL_DIR"
mv "$tmp/$BIN" "$INSTALL_DIR/$BIN"
chmod +x "$INSTALL_DIR/$BIN"
ok "installed ${BIN} ${VERSION}"

#
# Make `mm` available in future shells.
#
in_path() {
	case ":$PATH:" in
		*":$1:"*) return 0 ;;
		*) return 1 ;;
	esac
}

detect_rc() {
	shell_name=$(basename "${SHELL:-}" 2>/dev/null || true)
	case "$shell_name" in
		zsh)  printf '%s\n' "${ZDOTDIR:-$HOME}/.zshrc" ;;
		bash)
			if [ "$(uname -s)" = "Darwin" ]; then
				printf '%s\n' "$HOME/.bash_profile"
			else
				printf '%s\n' "$HOME/.bashrc"
			fi
			;;
		fish) printf '%s\n' "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish" ;;
		*)    printf '%s\n' "" ;;
	esac
}

write_path_line() {
	rc=$1
	case "$rc" in
		*/config.fish) line="fish_add_path \"$INSTALL_DIR\"" ;;
		*)             line="export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
	esac

	if [ -f "$rc" ] && grep -Fxq "$line" "$rc" 2>/dev/null; then
		ok "PATH already configured in $rc"
		return
	fi

	mkdir -p "$(dirname "$rc")"
	{
		printf '\n# added by mumei-md installer (%s)\n' "$(date +%Y-%m-%d)"
		printf '%s\n' "$line"
	} >> "$rc"
	ok "added $INSTALL_DIR to PATH in $rc"
	printf '    %srun `exec $SHELL` (or open a new terminal) to pick up the change%s\n' "$DIM" "$RESET"
}

step "configuring PATH"
if [ "$NO_MODIFY_PATH" = "1" ]; then
	ok "skipped (MUMEI_NO_MODIFY_PATH=1)"
elif in_path "$INSTALL_DIR"; then
	ok "$INSTALL_DIR already in PATH"
else
	rc=$(detect_rc)
	if [ -z "$rc" ]; then
		warn "could not detect shell rc file (\$SHELL=${SHELL:-unset}); add this to your shell config:"
		warn "  export PATH=\"$INSTALL_DIR:\$PATH\""
	else
		write_path_line "$rc"
	fi
fi

printf '\n%sdone.%s try it now:\n\n    %s%s sample.md%s\n' \
	"$BOLD" "$RESET" "$DIM" "$BIN" "$RESET"
