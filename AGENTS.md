# Agent reviewer guidance

This file follows the [AGENTS.md open standard](https://agents.md/) and
is read automatically by AI coding / review agents that respect it
(OpenAI Codex, Anthropic Claude, Gemini CLI, Cursor, Aider, and
others). It supplies project-specific context so the reviewer flags
the right things and stays away from the wrong things.

The same conventions are documented in `CONTRIBUTING.md` for human
contributors. This file is the agent-side mirror.

## What miru is

miru is a quiet terminal file viewer written in Go. Markdown is
rendered to ANSI through Glamour in the TUI and to HTML through
goldmark + mermaid.js in the browser; source and config files are
syntax-highlighted through Chroma in both contexts. It ships as a
single static binary distributed via Homebrew (`hir4ta/homebrew-tap`),
a `curl | sh` bootstrap, and `go install`. The binary contains a
self-install Bubble Tea UI (`miru install`) and an in-place updater
(`miru update`) so users do not need a Go toolchain on the target
machine.

Repository layout:

- `cmd/miru/` — entrypoint. Subcommand routing (`install` / `update`
  / `version`) is hand-rolled on `os.Args[1]`; Cobra is intentionally
  not used.
- `internal/config/` — `~/.config/miru/config.json` load / save.
  Single field today: `theme`.
- `internal/render/` — Glamour (ANSI), Chroma (`Source` /
  `SourceToHTML`), and goldmark (Markdown → HTML) renderers, plus
  embedded theme JSON under `assets/themes/`.
- `internal/nav/` — heading extraction + line mapping for `{` / `}`
  jumps in the Markdown TUI.
- `internal/tui/` — Bubble Tea `Model` / `Update` / `View` plus the
  settings (theme picker) and help overlays.
- `internal/installer/` — `miru install` and `miru update` Bubble
  Tea UIs. The installer self-copies the running binary and wires
  the user's shell rc PATH; the updater fetches the latest release
  tarball, verifies its SHA-256 against the published
  `checksums.txt`, and atomically replaces the binary.
- `install.sh` — thin shell bootstrap. Downloads + verifies the
  tarball then hands off to `miru install`. No UX in the script
  itself.
- `.goreleaser.yaml` + `.github/workflows/release.yml` — GoReleaser
  builds 4-arch tarballs (linux/darwin × amd64/arm64), SBOMs (syft),
  `checksums.txt`, cosign keyless signatures, and SLSA build
  provenance attestations. The Homebrew tap is updated by the same
  workflow.
- `CLAUDE.md` / `.claude/` — maintainer's local Claude Code rules.
  Dev-only. Japanese is fine here; everything else must be English.

## Setup commands

```sh
go test ./...                   # all tests
go vet ./...                    # static analysis
go build ./...                  # build everything
go build -o ./miru ./cmd/miru   # build the binary
./miru sample.md                # smoke test the TUI; press b for browser
shellcheck install.sh           # if you touched the bootstrap
actionlint                      # if you touched .github/workflows/*
```

Requirements: Go 1.26+ (`go.mod`). A 24-bit colour terminal for the
best theme experience.

## Distribution boundary — flag violations

- The version string is injected at build time by GoReleaser via
  `-ldflags -X github.com/hir4ta/miru/internal/installer.Version=...`.
  Source must not hard-code a release version. The only literal is
  `Version = "dev"` as the non-release fallback; do not change it.
- README's `VERSION=v0.1.0` is an *example* in the install one-liner.
  PRs that "update" this example to a current tag are noise — flag.
- Files under `cmd/`, `internal/`, `README.md`, `install.sh`,
  `LICENSE`, `SECURITY.md`, `PRIVACY.md`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `AGENTS.md` are SHIPPED to users as
  **explanatory text** and MUST be **English**. Japanese intent
  notes go in `<!-- HTML comments -->` only.
- `sample*` files are shipped as **demo content** for renderer
  testing, not as explanatory text. Multilingual sections are
  fine and in fact desirable — Markdown demos benefit from
  showing how the renderer handles different writing systems
  (the paper-note CSS pulls Kiwi Maru for CJK exactly for this
  reason). Do not flag Japanese / other-language sections in
  `sample*`.
- `CLAUDE.md` and `.claude/` are dev-only — Japanese is fine.
- `install.sh` MUST stay a thin bootstrap. Adding interactive UX,
  PATH editing, or `eval` of downloaded content there is out of
  scope. `miru install` owns the UX.

## Code style

### Go conventions

- Go 1.26+. Stdlib > new dependency. Modern stdlib idioms:
  `slices.Contains` / `slices.Sort`, builtin `min` / `max`,
  `cmp.Or`, `errors.Is` / `errors.As`. Prefer them to hand-rolled
  equivalents.
- `internal/<topic>/` packages are organised by concern (one
  package = one responsibility). No `internal/util` or
  `internal/common` catch-alls. `internal/render/` legitimately
  covers Glamour + Chroma + goldmark because they share the
  `Markdown → output` concern.
- Package names: short, single-word, lowercase. Don't repeat the
  package name in type names — `render.ANSI`, not
  `render.Renderer`.
- Errors: wrap with `fmt.Errorf("operation: %w", err)`. Defensive
  checks happen at system boundaries (user input, I/O, network),
  not on internal returns.
- Receivers: value receivers by default; pointer receivers only
  when the method mutates. Follow existing patterns — Bubble Tea
  v2 `Model` types are intentionally value-receiver.
- `init()` only for inert bookkeeping (e.g. `debug.ReadBuildInfo`
  fallback for `installer.Version`). No global state, no side
  effects.
- No abstract `interface` pre-defined "for testability". Add one
  when there are two real implementations.
- No `context.Context` plumbing yet — miru is a single-shot CLI;
  add it when a real timeout / cancel scenario appears.

### Charm stack (`charm.land/*` v2)

- Bubble Tea v2, bubbles v2, Lipgloss v2, Glamour v2 are mandatory.
  Do not mix in `github.com/charmbracelet/*` v1 — types diverge and
  builds fail.
- v2 `View()` returns `tea.View`, not `string`. Construct with
  `tea.NewView(content)`.
- Lipgloss v2 `lipgloss.Color()` is a function returning
  `image/color.Color`. Fields holding colour are typed
  `image/color.Color`.
- Overlays: use `internal/tui/overlay.go` `Overlay()` and
  `DimBackground()` only. Don't add a parallel implementation.
- Floating panel content does NOT set its own background — the
  terminal's default background is intentional. Flag any
  `Background(...)` on the settings / help boxes.

### Markdown → HTML browser path

- Browser preview is opt-in (`b` key). Source files use
  `SourceToHTML`; Markdown uses `ToHTML`. The Markdown HTML page
  is rendered as a handwritten paper note (cream background +
  Caveat for headings + Patrick Hand for body + Kiwi Maru for
  CJK), so it always embeds the Google Fonts stylesheet. When the Markdown also contains a `mermaid`
  code block, the page additionally embeds mermaid.js and
  svg-pan-zoom from jsDelivr; the diagram renders in
  `look: "handDrawn"` and supports click-to-zoom (clicking the
  SVG opens a fullscreen modal with wheel-zoom + drag-pan,
  dismissed by Escape / outside-click / ×). The
  Source HTML page is intentionally **not** styled as a note —
  it stays a dark, monospace, fully-local page so reading code
  is not impaired. Both behaviours are documented in
  `PRIVACY.md`.
- A PR that introduces additional third-party network egress (a
  new CDN, an analytics ping, a remote font on the Source path)
  MUST update `PRIVACY.md` and `SECURITY.md` in the same PR.
  Flag if missing.
- goldmark runs with `WithUnsafe()` and mermaid with
  `securityLevel: "loose"` to match Obsidian / VS Code Markdown
  preview semantics. Documented as a deliberate, scoped choice in
  `SECURITY.md`. Do not flag the loose setting in isolation.

## Testing instructions

- Tests live alongside the file under test in the same package
  (`internal/render/html_test.go`, not a separate `tests/` tree).
- Use table-driven + `t.Run(name, ...)`.
- TUI / rendered output contains ANSI; strip with
  `github.com/charmbracelet/x/ansi.Strip()` before substring
  assertions (see `internal/render/lists_test.go`).
- Fuzz tests live next to their target (`*_fuzz_test.go`).
- Temp files: `t.TempDir()`. Never `os.MkdirTemp` with manual
  cleanup.
- No network / browser-spawning tests. They are CI-flaky and
  exercise behaviour Go cannot meaningfully assert.

## Review guidelines

When reviewing pull requests in this repository, focus on:

1. **Distribution boundary** (above). Shipped artifacts in
   Japanese, dev paths leaked into the binary, hard-coded version
   strings, README pinned to a stale tag.
2. **Go modernity**: hand-rolled loops where a stdlib helper
   exists, custom `min` / `max`, sentinel-error string compares
   in place of `errors.Is`, unnecessary `init()` side effects.
3. **Charm v2 hygiene**: v1 imports, wrong `View()` return type,
   manual overlay re-implementation, background colour on
   floating panels, missing `tea.WindowSizeMsg` plumbing.
4. **Render correctness**: ANSI-aware string handling (use
   `ansi.Strip` / `ansi.Cut`, never raw `strings.Index` on styled
   output), Glamour list post-processing drift
   (`internal/render/lists.go`).
5. **Browser-preview privacy**: any new third-party `<script>`,
   `<link>`, font, analytics ping in `internal/render/html.go`
   without a matching `PRIVACY.md` row.
6. **Installer / updater safety**: shell rc edits MUST stay
   idempotent (`rcContainsLine`) and in a single marked block
   (`# added by miru installer`). The updater MUST verify SHA-256
   against `checksums.txt` before applying — never skip the gate.
7. **Release pipeline integrity**: `.goreleaser.yaml`
   `name_template` is contractual with `install.sh` and
   `internal/installer/update.go`'s URL assembly. Renaming the
   template silently breaks every prior client. The cosign signing
   block and SBOM step MUST stay on the release path.
8. **CI workflow integrity**: every `uses:` in
   `.github/workflows/` MUST be SHA-pinned with a trailing
   `# vN.N.N` comment. `actionlint` catches yaml bugs but does
   NOT catch a mutable-tag reference — reviewers must.

### What NOT to flag

- **Cobra / CLI-framework adoption.** The hand-rolled
  `os.Args[1]` routing is a deliberate KISS choice. Two
  subcommands do not justify a framework.
- **Premature abstraction for one-off helpers.** Three repetitions
  before extraction.
- **Forward-compatibility shims** — renamed `_var` placeholders,
  `// removed` comment trails, feature flags running old + new in
  parallel. The project prefers direct rewrites.
- **Defensive checks on internal returns.** System-boundary
  defence only.
- **Style-only nits** — `gofmt -s` handles those. Surface only
  when they cause functional issues.
- **Windows native build proposals.** Out of scope; the release
  pipeline targets Linux + macOS by design.
- **Re-introducing a `CHANGELOG.md`.** GoReleaser generates
  release notes from Conventional Commits; we do not maintain a
  separate file.
- **`context.Context` plumbing.** miru is single-shot; revisit
  when a real cancel / timeout scenario shows up.

### Severity rubric (calibration)

- **HIGH** — silently breaks user-facing behaviour, leaks a
  secret, bypasses a security gate, breaks the release tarball,
  or inverts a documented invariant (e.g. removes the SHA-256
  check in `miru update`).
- **MEDIUM** — degrades observability, introduces silent drift,
  weakens an existing check, or contradicts shipped
  documentation.
- **LOW** — style polish, minor clarity, suggestion-grade.
- **NIT** — rarely useful; prefer to omit.

If you cannot articulate a concrete failure scenario, do not raise
the finding. Hypothetical concerns without a chain to user impact
are noise.

## Security considerations

- Threat model: `SECURITY.md`. Primary surfaces: release
  distribution (mitigated by SHA-256 + cosign + SLSA), browser
  preview's `unsafe`-HTML + mermaid `loose`-securityLevel
  (documented), and the installer's shell rc edit (idempotent,
  marked).
- Never commit `.env`, credentials, or private keys. Gitleaks runs
  PR-time and weekly. CodeQL runs PR-time + weekly with
  `+security-extended`. govulncheck gates the release workflow.
- Workflow `uses:` MUST be SHA-pinned.
- A change that adds a network destination (a new CDN, a font
  host, an API) MUST update `PRIVACY.md` and `SECURITY.md` in
  the same PR.

## PR guidelines

- **Conventional Commits** subject line (`feat:` / `fix:` /
  `docs:` / `refactor:` / `chore:` / `ci:` / `perf:` / `build:`
  / `test:`). Imperative mood, one line.
- No `Co-Authored-By` trailers.
- Single-line subject; PR body follows
  `.github/PULL_REQUEST_TEMPLATE.md`.
- No `--no-verify`, no `--force` push to `main`, no unsigned
  tags.
- `main` has branch protection (`enforce_admins: true`) and
  `required_conversation_resolution: true`. Required status
  checks: `test`, `shellcheck`, `actionlint`, `codeql (go)`,
  `codeql (actions)`, `govulncheck`, `scan`. All must be green
  before squash-merge.
- Releases: the maintainer cuts an annotated tag (`v*`);
  `release.yml` runs GoReleaser. No version commit is made; the
  binary's version is injected from the tag via ldflags.
