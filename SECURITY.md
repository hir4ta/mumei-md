# Security Policy

miru is a terminal file viewer distributed as a static Go binary. It runs entirely on the user's machine; see [PRIVACY.md](./PRIVACY.md) for the full network egress policy.

## Supported versions

The latest published release is supported. miru follows semantic versioning on the `0.x` track; older `0.x.y` releases receive no backports. Users on unsupported versions should upgrade before reporting an issue.

| Version | Supported |
|---|---|
| Latest `0.x.y` (see [Releases](https://github.com/hir4ta/miru/releases)) | Yes |
| Older `0.x.y` | No |

## Reporting a vulnerability

**Do not open a public issue for security reports.** miru uses GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) channel exclusively.

To report a vulnerability:

1. Go to <https://github.com/hir4ta/miru/security/advisories/new>.
2. Fill out the advisory form with:
   - A clear summary of the issue (one sentence).
   - The affected component (`cmd/miru`, `internal/installer`, `internal/render`, `internal/tui`, `install.sh`, release pipeline).
   - A minimal reproduction (file or input, exact command, observed vs. expected behavior).
   - Your assessment of severity and impact.
3. The maintainer will acknowledge within 7 days and provide a triage decision within 14 days.

If you cannot use GitHub advisories for some reason, email shunichi@hir4ta.com with the same content. Do not post details on social media or public forums before a fix is published.

## Threat model summary

miru's primary attack surfaces:

- **Release distribution.** `curl | sh`, `miru update`, and `brew install` all fetch prebuilt binaries from GitHub Releases. Both the shell bootstrap and `miru update` verify the SHA-256 of the downloaded tarball against the release's `checksums.txt` before extraction. A compromise of the release pipeline (workflow secrets, runner) would still bypass this gate; the workflow uses pinned action versions, least-privilege permissions, and a fine-grained PAT scoped only to the Homebrew tap repository.
- **Browser preview (`b` key).** The HTML page rendered for Markdown enables goldmark's `unsafe` mode (raw HTML pass-through) and runs mermaid with `securityLevel: "loose"` — matching the behavior of Obsidian and the VS Code Markdown preview. This is documented in README. Users should only browser-render Markdown they trust. The TUI view itself executes nothing.
- **Config and PATH manipulation.** `miru install` writes to `~/.config/miru/config.json` and (optionally) appends a single line to the user's shell rc. The rc edit is idempotent (`rcContainsLine`) and clearly marked with `# added by miru installer`. No daemons, no startup hooks.

## Out of scope

- Issues in upstream dependencies that are not exploitable through miru's API. Report those to the upstream project; we will track via dependabot.
- Vulnerabilities in third-party tools (`mermaid.js` via CDN, the user's terminal, the browser). We do not control these and cannot patch them.
- Theoretical CPU side-channel attacks against the Go runtime.
