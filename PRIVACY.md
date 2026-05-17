# Privacy Policy

miru is a terminal file viewer that runs entirely on the user's local machine.

## Data collection

miru collects, transmits, and stores no user data. No telemetry, no analytics, no error reporting. All persistent state is the single config file at `~/.config/miru/config.json` (or `$XDG_CONFIG_HOME/miru/config.json`), which currently contains only the user's chosen theme.

## Network egress

miru itself initiates network requests in exactly these situations:

| Trigger | Destination | Purpose | Cancelable |
|---|---|---|---|
| `miru update` | `github.com/hir4ta/miru/releases/latest` (HEAD) | Resolve the latest version tag from the redirect | Quit (`q` / `Ctrl+C`) before the request fires |
| `miru update` | `github.com/hir4ta/miru/releases/download/<tag>/miru_*.tar.gz` | Download the release tarball | Same |
| `miru update` | `github.com/hir4ta/miru/releases/download/<tag>/checksums.txt` | Verify the tarball's SHA-256 before applying | Same |
| `b` (browser preview) on a Markdown file | local `file://` URI | Open the rendered HTML in your default browser | Don't press `b` |

The browser preview page embeds `<script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs">`. **Your browser fetches that script when it loads the page.** This is a third-party network request initiated by your browser, not by miru. To avoid it, do not press `b`, or use a browser with the script blocked (uBlock Origin, NoScript).

`miru install` does not make any network requests. `miru <file>` (the default TUI view) does not make any network requests.

## Third-party services

- **GitHub Releases / GitHub raw content.** When you install via `curl -fsSL .../install.sh | sh` or `miru update`, your client fetches from `github.com` and `raw.githubusercontent.com`. GitHub's privacy policy applies to those connections.
- **Homebrew.** `brew install hir4ta/tap/miru` follows Homebrew's normal install pipeline; see [Homebrew's analytics policy](https://docs.brew.sh/Analytics).
- **jsDelivr CDN.** The browser preview's mermaid script is served by `cdn.jsdelivr.net`. See [jsDelivr's privacy policy](https://www.jsdelivr.com/terms/privacy-policy-jsdelivr-net).

## Local data written by miru

| Path | When | Contents |
|---|---|---|
| `~/.config/miru/config.json` | First time you change the theme via `s` | `{"theme": "..."}` |
| `<INSTALL_DIR>/miru` (default `~/.local/bin`) | After `curl | sh` bootstrap or `miru install` | The binary itself |
| Shell rc (`.zshrc` / `.bashrc` / `.bash_profile` / `config.fish`) | After `miru install` if PATH does not already contain the install dir | One marked block: `# added by miru installer\nexport PATH="..."` |
| `$TMPDIR/miru-*.html` | When you press `b` | Rendered HTML, removed by the OS when the temp directory is cleared |
| `$TMPDIR/miru-update-*.tar.gz` | During `miru update` | The release tarball, removed on success or checksum mismatch |

No other files are written. miru never writes to `~/.local/share/`, no daemons, no startup hooks.

## Uninstall

See [README.md → Uninstall](./README.md#uninstall) for the full procedure to remove all miru-related files and undo the PATH change.

## Changes to this policy

Material changes to network behavior will be announced in the release notes and a corresponding entry will be added to this file.
