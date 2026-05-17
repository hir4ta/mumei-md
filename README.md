# miru

A quiet file viewer for your terminal. Markdown gets a full glamour render with mermaid in the browser; source and config files get chroma-powered syntax highlighting.

<p align="center">
  <img src="https://github.com/user-attachments/assets/d5f7afa5-d8be-401d-bda5-04dae2ad8bbe" alt="miru rendering sample.md" width="32%">
  <img src="https://github.com/user-attachments/assets/1be2b6a1-d058-4819-99df-64a75eab4810" alt="miru rendering sample.js" width="32%">
  <img src="https://github.com/user-attachments/assets/a7ad20b7-47cd-4f8f-a657-a0de22bee189" alt="miru rendering sample.py" width="32%">
</p>

## Install

One-liner. No Go toolchain required:

```sh
curl -fsSL https://raw.githubusercontent.com/hir4ta/miru/main/install.sh | sh
```

The shell bootstrap downloads a prebuilt static binary for your platform, then hands off to `miru install` — a Bubble Tea install UI that self-installs the binary, configures your shell rc PATH, and verifies the result.

Supported platforms: macOS (Intel / Apple Silicon), Linux (x86_64 / arm64). Shell rc handled: zsh, bash, fish.

You can also run the installer directly any time after the binary is on disk:

```sh
miru install                     # re-run with the rich UI
miru install --no-modify-path    # only install the binary, skip PATH update
```

Environment overrides:

| Variable | Default | Purpose |
|---|---|---|
| `VERSION` | latest release | pin a specific tag (e.g. `v0.1.0`) |
| `INSTALL_DIR` | `$HOME/.local/bin` | binary destination |
| `MIRU_NO_MODIFY_PATH` | `0` | set to `1` to skip the shell rc update |

Example:

```sh
VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin MIRU_NO_MODIFY_PATH=1 \
  curl -fsSL https://raw.githubusercontent.com/hir4ta/miru/main/install.sh | sh
```

If you have a Go toolchain, you can also build from source:

```sh
go install github.com/hir4ta/miru/cmd/miru@latest
```

## Usage

```sh
miru README.md           # markdown (glamour)
miru main.go             # source (chroma syntax highlight)
miru config.yaml         # config (chroma)
miru Dockerfile          # filename-detected (chroma)
miru --theme gruvbox README.md
miru --list-themes
```

Files with a `.md` / `.markdown` extension take the markdown path (glamour ANSI in the TUI, goldmark + github-markdown.css + mermaid.js in the browser). Everything else takes the chroma path with line numbers in the TUI and a styled HTML page in the browser.

## Key bindings

| Key | Action |
|---|---|
| `j` / `k` | Scroll one line |
| `Ctrl+d` / `Ctrl+u` | Half page scroll |
| `g` / `G` | Top / bottom |
| `{` / `}` | Jump to previous / next heading (markdown only) |
| `b` | Open in browser |
| `s` | Settings (theme picker) |
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

Press `s` in the TUI to open the theme picker — selections apply live and persist to the config file. The CLI flag, env var, and config file below are still supported for scripting and pinning a default.

### Precedence

```
--theme flag  >  $MIRU_THEME env var  >  config file  >  claude
```

### Config file

`~/.config/miru/config.json` (or `$XDG_CONFIG_HOME/miru/config.json`):

```json
{
  "theme": "gruvbox"
}
```
