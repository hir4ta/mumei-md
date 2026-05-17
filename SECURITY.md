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

## Verifying releases

From `v0.7.0` onward, every release artifact is signed and attested:

- **`checksums.txt`** is signed with a [Sigstore](https://www.sigstore.dev/) keyless cosign signature. The certificate identity is the workflow that produced the release (`https://github.com/hir4ta/miru/.github/workflows/release.yml@refs/tags/<tag>`), and the issuer is GitHub's OIDC provider.
- **Each tarball** carries an [SLSA build provenance](https://slsa.dev/) attestation published to the GitHub attestation store and the Sigstore transparency log.

End-to-end verification (use `cosign` from <https://docs.sigstore.dev/cosign/installation/> and `gh` from <https://cli.github.com/>):

```sh
TAG=v0.7.0
ASSET=miru_${TAG#v}_$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz
URL=https://github.com/hir4ta/miru/releases/download/${TAG}

curl -fsSL "${URL}/${ASSET}"           -o "${ASSET}"
curl -fsSL "${URL}/checksums.txt"      -o checksums.txt
curl -fsSL "${URL}/checksums.txt.sig"  -o checksums.txt.sig
curl -fsSL "${URL}/checksums.txt.pem"  -o checksums.txt.pem

# 1. Cosign signature on checksums.txt
cosign verify-blob \
  --certificate-identity-regexp "https://github.com/hir4ta/miru/.+" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt

# 2. Tarball SHA-256 matches checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing

# 3. SLSA build provenance attestation
gh attestation verify "${ASSET}" --repo hir4ta/miru
```

Any failure of step 1 or 3 means the artifact did not come from this repository's release pipeline — do not run it.

## Threat model summary

miru's primary attack surfaces:

- **Release distribution.** `curl | sh`, `miru update`, and `brew install` all fetch prebuilt binaries from GitHub Releases. Both the shell bootstrap and `miru update` verify the SHA-256 of the downloaded tarball against the release's `checksums.txt` before extraction. A compromise of the release pipeline (workflow secrets, runner) would still bypass this gate; the workflow uses pinned action versions, least-privilege permissions, and a fine-grained PAT scoped only to the Homebrew tap repository.
- **Browser preview (`b` key).** Pressing `b` starts a loopback HTTP server (`127.0.0.1:<random>`, lives only while the TUI is open) and opens the rendered page in the default browser. The server is sandboxed to the file's project root (the first ancestor with `.git`, `go.mod`, `package.json`, `Cargo.toml`, or `pyproject.toml`; otherwise the file's own directory), enforces the sandbox both lexically and after `EvalSymlinks` so symlinks cannot escape, and rejects any request outside the root with 403 before touching disk. The HTML page rendered for Markdown enables goldmark's `unsafe` mode (raw HTML pass-through) and runs mermaid with `securityLevel: "loose"` — matching the behavior of Obsidian and the VS Code Markdown preview. Because the loopback origin would otherwise let a `<script>` smuggled in via raw HTML fetch sibling files under the root, miru applies a per-render nonce-based Content-Security-Policy (`default-src 'none'; script-src 'nonce-…' 'strict-dynamic'; …`) that blocks any non-nonced inline script, inline event handler, or third-party CDN script. Users should still only browser-render Markdown they trust. The TUI view itself executes nothing. The Markdown preview also fetches a Google Fonts stylesheet (Caveat / Patrick Hand / Kiwi Maru) on every load to render the page as a handwritten paper note; this is a passive `<link rel="stylesheet">` reference, the same trust model as the mermaid script and (when a mermaid block is present) the svg-pan-zoom script that powers click-to-zoom on diagrams. Source-file previews (non-Markdown) inline chroma-generated styles and fetch nothing. Full destination list and how to disable each is in [PRIVACY.md](./PRIVACY.md).
- **Config and PATH manipulation.** `miru install` writes to `~/.config/miru/config.json` and (optionally) appends a single line to the user's shell rc. The rc edit is idempotent (`rcContainsLine`) and clearly marked with `# added by miru installer`. No daemons, no startup hooks.

## Review model

miru is solo-maintained. To keep changes auditable without a co-maintainer:

- **PRs are required for every change to `main`** (enforced by repository ruleset). Direct push, force push, and branch deletion are blocked. Linear history is required, so merges land without merge commits (squash or rebase only — the project convention is squash).
- **Required CI checks** (all must pass before merge), listed as `workflow / job`:
  - `ci / test` — `go test -race ./...` (`.github/workflows/ci.yml`)
  - `ci / shellcheck` — `shellcheck install.sh` (`ci.yml`)
  - `ci / actionlint` — workflow YAML lint (`ci.yml`)
  - `codeql / codeql (go)` — CodeQL `+security-extended` on Go code (`codeql.yml`)
  - `codeql / codeql (actions)` — CodeQL on workflow files (`codeql.yml`)
  - `govulncheck / govulncheck` — Go vulnerability database scan (`govulncheck.yml`)
  - `gitleaks / scan` — secret leak scan (`gitleaks.yml`)
- **Required conversation resolution** — any reviewer thread on a PR must be resolved before merge.
- **Required approving reviewers: 0**, because GitHub disallows the PR author from approving their own PR and there is no co-maintainer. The gap is intentionally filled by automated reviewers, not by a rubber-stamp human:
  - **Copilot code review** (ruleset-triggered) runs on every PR open and re-runs on push.
  - **OpenAI Codex code review** (GitHub App + Codex Cloud) runs on every PR open. `AGENTS.md` at the repo root supplies the calibration (focus areas + "what NOT to flag" + severity rubric) so Codex's output is project-specific.
- **Secret + supply-chain scanning** runs on every PR and weekly: gitleaks (PR + cron), CodeQL `+security-extended` (PR + cron), govulncheck (PR + cron + release gate), OpenSSF Scorecards (cron + branch-protection-rule trigger).

Consequence: the OpenSSF Scorecard `Code-Review` and `Branch-Protection` `require approvers` items score below maximum, and we have accepted this trade-off (the corresponding alerts are dismissed with `won't fix` and link back here). The substitute controls above are stricter than a single human approver in several axes (Copilot + Codex catch issues humans miss; CI gates can't be waived) and they cost the maintainer nothing per PR.

## Out of scope

- Issues in upstream dependencies that are not exploitable through miru's API. Report those to the upstream project; we will track via dependabot.
- Vulnerabilities in third-party tools (`mermaid.js` via CDN, the user's terminal, the browser). We do not control these and cannot patch them.
- Theoretical CPU side-channel attacks against the Go runtime.
