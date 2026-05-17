package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(tmp, "miru", "config.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPathHomeFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(home, ".config", "miru", "config.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	want := Config{Theme: "gruvbox"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSaveCreatesParentDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if err := Save(Config{Theme: "nord"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "miru", "config.json")); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestLoadMissingFileReturnsZero(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (Config{}) {
		t.Errorf("got %+v, want zero value", got)
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	p, _ := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Error("Load: want error for corrupt JSON, got nil")
	}
}
