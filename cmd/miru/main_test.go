package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hir4ta/miru/internal/render"
)

func writeConfig(t *testing.T, theme string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if theme == "" {
		return
	}
	dir := filepath.Join(tmp, "miru")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"theme": theme})
	if err := os.WriteFile(filepath.Join(dir, "config.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveThemeFlagWins(t *testing.T) {
	writeConfig(t, "gruvbox")
	t.Setenv("MIRU_THEME", "nord")

	if got := resolveTheme("dracula"); got != "dracula" {
		t.Errorf("got %q, want flag value dracula", got)
	}
}

func TestResolveThemeEnvBeatsConfig(t *testing.T) {
	writeConfig(t, "gruvbox")
	t.Setenv("MIRU_THEME", "nord")

	if got := resolveTheme(""); got != "nord" {
		t.Errorf("got %q, want env value nord", got)
	}
}

func TestResolveThemeConfigBeatsDefault(t *testing.T) {
	writeConfig(t, "gruvbox")
	t.Setenv("MIRU_THEME", "")

	if got := resolveTheme(""); got != "gruvbox" {
		t.Errorf("got %q, want config value gruvbox", got)
	}
}

func TestResolveThemeDefault(t *testing.T) {
	writeConfig(t, "")
	t.Setenv("MIRU_THEME", "")

	if got := resolveTheme(""); got != render.DefaultTheme {
		t.Errorf("got %q, want default %q", got, render.DefaultTheme)
	}
}

func TestResolveThemeIgnoresWhitespace(t *testing.T) {
	writeConfig(t, "gruvbox")
	t.Setenv("MIRU_THEME", "")

	if got := resolveTheme("   "); got != "gruvbox" {
		t.Errorf("whitespace-only flag should fall through; got %q", got)
	}
}

func TestProjectRootFindsMarker(t *testing.T) {
	// Mirror a typical layout: project root has a .git directory, the entry
	// markdown lives a few levels down. The server should sandbox to the root
	// so links like `../sibling/file.md` resolve, not be 403'd.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "docs", "guide", "page.md")
	if err := os.MkdirAll(filepath.Dir(deep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deep, []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := projectRoot(deep)
	// On macOS t.TempDir is under /var/folders/... which symlinks to /private/var.
	// Both projectRoot and root must compare under the same canonical form.
	want, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Errorf("projectRoot=%q, want %q", gotResolved, want)
	}
}

func TestProjectRootFallsBackToFileDir(t *testing.T) {
	// No marker anywhere up the chain (we'd hit / before finding one in the
	// test env). The fallback is the file's own directory.
	dir := t.TempDir()
	file := filepath.Join(dir, "lone.md")
	if err := os.WriteFile(file, []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := projectRoot(file)
	if got != dir {
		// May still legitimately find a marker high up (e.g. a developer's
		// /Users/foo/.git). In that case the fallback isn't exercised; skip
		// rather than fail.
		if _, err := os.Stat(filepath.Join(got, ".git")); err == nil {
			t.Skipf("environment has %q with marker; fallback path untested", got)
		}
		t.Errorf("projectRoot=%q, want fallback %q", got, dir)
	}
}
