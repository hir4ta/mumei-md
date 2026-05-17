# mumei-md

A rich-looking Markdown viewer you can open in your terminal.

## Install

One-liner. No Go toolchain required — downloads a prebuilt static binary for your platform, drops it in `~/.local/bin`, and appends a PATH entry to your shell rc so `mm` works in any new shell:

```sh
curl -fsSL https://raw.githubusercontent.com/hir4ta/mumei-md/main/install.sh | sh
```

Supported platforms: macOS (Intel / Apple Silicon), Linux (x86_64 / arm64). Shell rc handled: zsh, bash, fish.

Environment overrides:

| Variable | Default | Purpose |
|---|---|---|
| `VERSION` | latest release | pin a specific tag (e.g. `v0.1.0`) |
| `INSTALL_DIR` | `$HOME/.local/bin` | binary destination |
| `MUMEI_NO_MODIFY_PATH` | `0` | set to `1` to skip the shell rc update |

Example:

```sh
VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin MUMEI_NO_MODIFY_PATH=1 \
  curl -fsSL https://raw.githubusercontent.com/hir4ta/mumei-md/main/install.sh | sh
```

If you have a Go toolchain, you can also build from source:

```sh
go install github.com/hir4ta/mumei-md/cmd/mm@latest
```

## Usage

```sh
mm README.md
mm --theme gruvbox README.md
mm --list-themes
```

## Key bindings

| Key | Action |
|---|---|
| `j` / `k` | Scroll one line |
| `Ctrl+d` / `Ctrl+u` | Half page scroll |
| `g` / `G` | Top / bottom |
| `{` / `}` | Jump to previous / next heading |
| `b` | Open in browser |
| `?` | Help |
| `q` | Quit |

## Color themes

The default is a warm coral/terracotta theme inspired by Claude Code (`claude`).

| Theme | Mood |
|---|---|
| `claude` (default) | warm cream + coral + tan + gold |
| `gruvbox` | retro warm yellow/orange on brown |
| `everforest` | forest green + muted earthy |
| `nord` | cool arctic blue/gray minimalist |
| `dracula` | classic purple/pink/cyan dark |
| `tokyo-night` | deep blue cyberpunk |

### Precedence

```
--theme flag  >  $MUMEI_THEME env var  >  config file  >  claude
```

### Config file

`~/.config/mumei-md/config.json` (or `$XDG_CONFIG_HOME/mumei-md/config.json`):

```json
{
  "theme": "gruvbox"
}
```
