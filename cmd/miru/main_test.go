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
