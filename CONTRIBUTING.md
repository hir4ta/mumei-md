# Contributing to miru

Thanks for considering a contribution. miru is a small project and aims to stay that way. This document covers how to set up your environment, run the tests, and propose a change.

## Code of Conduct

By participating in this project you agree to abide by the [Code of Conduct](./CODE_OF_CONDUCT.md).

## Setting up

```sh
git clone https://github.com/hir4ta/miru.git
cd miru
go test ./...
go vet ./...
go build ./...
./miru sample.md           # smoke test the TUI
```

Requirements:
- Go 1.26 or newer (see `go.mod`).
- A terminal that supports 24-bit color for the best theme experience.
- `actionlint` (optional) if you touch `.github/workflows/*.yml`.
- `shellcheck` (optional) if you touch `install.sh`.

## Before submitting a pull request

1. **Tests pass.** `go test ./...` is green.
2. **No new vet warnings.** `go vet ./...` is silent.
3. **Build succeeds for all targets.** `go build ./...`.
4. **Commit messages follow Conventional Commits.** `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `ci:`, `perf:`, `build:`. Imperative mood, one line.
5. **No unrelated changes.** A fix is a fix; reformatting / renaming go in their own PR.
6. **README and SECURITY.md updated** when behavior changes (new key binding, new network destination, new file written).

## Scope guidance

miru is a viewer, not an editor or a file manager. Features that align well:

- New themes.
- New file types that benefit from chroma's lexer detection.
- Improvements to glamour markdown rendering or chroma syntax highlighting.
- Performance fixes to the TUI render path.

Features outside scope (please open an issue to discuss before coding):

- File editing.
- Multi-file picker / directory browser.
- Network-dependent features (remote file fetch, integrations).
- Plugin / extension system.

## Security issues

**Do not open a public issue for security reports.** See [SECURITY.md](./SECURITY.md).

## Releases

Releases are cut by the maintainer via `git tag -a vX.Y.Z` → push. GoReleaser publishes the binaries and updates the Homebrew tap automatically. Contributors do not need to bump a version file (there is none — the binary's version is injected from the git tag via `-ldflags`).
