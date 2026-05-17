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
| `b` (browser preview) on any file | local `file://` URI | Open the rendered HTML in your default browser | Don't press `b` |
| `b` (browser preview) on a Markdown file, once the page loads | `fonts.googleapis.com` / `fonts.gstatic.com` | Fetched by your browser for the paper-note fonts (Caveat for headings, Patrick Hand for body, Kiwi Maru for CJK) | Don't press `b` on Markdown, or block the hosts |
| `b` (browser preview) on a Markdown file that contains a `mermaid` code block, once the page loads | `cdn.jsdelivr.net` | Fetched by your browser for mermaid.js | Don't press `b`, omit mermaid blocks, or block the host |

The Markdown browser preview always embeds the Google Fonts stylesheet (`https://fonts.googleapis.com/css2?family=Caveat&family=Patrick+Hand&family=Kiwi+Maru...`, which in turn fetches the woff2 files from `fonts.gstatic.com`) used to render the page as a handwritten paper note. The mermaid script (`https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs`) is embedded **only when the rendered Markdown contains a `mermaid` code block**. Source-file previews (non-Markdown files opened via `b`) inline chroma-generated styles and fetch nothing. **The third-party requests above are initiated by your browser, not by miru.** To avoid them, do not press `b` on Markdown, or use a browser with the resources blocked (uBlock Origin, NoScript).

`miru install` does not make any network requests. `miru <file>` (the default TUI view) does not make any network requests.

## Third-party services

- **GitHub Releases / GitHub raw content.** When you install via `curl -fsSL .../install.sh | sh` or `miru update`, your client fetches from `github.com` and `raw.githubusercontent.com`. GitHub's privacy policy applies to those connections.
- **Homebrew.** `brew install hir4ta/tap/miru` follows Homebrew's normal install pipeline; see [Homebrew's analytics policy](https://docs.brew.sh/Analytics).
- **jsDelivr CDN.** The Markdown browser preview's mermaid script is served by `cdn.jsdelivr.net` and is fetched only when the Markdown contains a `mermaid` code block. See [jsDelivr's privacy policy](https://www.jsdelivr.com/terms/privacy-policy-jsdelivr-net).
- **Google Fonts.** The Markdown browser preview's paper-note fonts (Caveat for headings, Patrick Hand for body, Kiwi Maru for CJK characters) are served by `fonts.googleapis.com` and `fonts.gstatic.com` and are fetched on every Markdown preview. Source-file previews do not use these fonts. See [Google's privacy policy](https://policies.google.com/privacy) and the [Google Fonts FAQ](https://developers.google.com/fonts/faq#what_does_using_the_google_fonts_api_mean_for_the_privacy_of_my_users).

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
