package installer

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	want := []byte("hello miru")
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstallBinaryCopiesAndChmods(t *testing.T) {
	tmp := t.TempDir()
	cfg := Config{InstallDir: tmp}

	// installBinary uses os.Executable() (the test binary). That's a real
	// file we can copy from.
	got, err := installBinary(cfg)
	if err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	want := filepath.Join(tmp, "miru")
	if got != want {
		t.Errorf("got dst %q, want %q", got, want)
	}
	st, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0o111 == 0 {
		t.Error("destination is not executable")
	}
}

func TestInstallBinaryReinstallIsNoop(t *testing.T) {
	// When the running binary already lives at the destination, installBinary
	// should return the path without copying.
	exe, err := os.Executable()
	if err != nil {
		t.Skip("os.Executable unavailable:", err)
	}
	cfg := Config{InstallDir: filepath.Dir(exe)}

	// Test only works when the executable is literally named "miru".
	if filepath.Base(exe) != "miru" {
		t.Skip("executable name is not 'miru'; skipping no-op check")
	}

	got, err := installBinary(cfg)
	if err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	if got != exe {
		t.Errorf("got %q, want %q", got, exe)
	}
}

func TestIsDirInPATH(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+"/usr/bin")
	if !isDirInPATH(tmp) {
		t.Errorf("expected %q in PATH", tmp)
	}
	if isDirInPATH("/nonexistent") {
		t.Errorf("did not expect /nonexistent in PATH")
	}
}

func TestDetectShellRCZsh(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("ZDOTDIR", "")

	rc, line, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	if rc != filepath.Join(home, ".zshrc") {
		t.Errorf("rc = %q, want %s/.zshrc", rc, home)
	}
	if !strings.Contains(line, "/opt/bin") || !strings.Contains(line, "export PATH") {
		t.Errorf("line = %q, want export PATH with /opt/bin", line)
	}
}

func TestDetectShellRCZshWithZDOTDIR(t *testing.T) {
	zdot := t.TempDir()
	t.Setenv("SHELL", "/usr/bin/zsh")
	t.Setenv("ZDOTDIR", zdot)

	rc, _, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	if rc != filepath.Join(zdot, ".zshrc") {
		t.Errorf("rc = %q, want %s/.zshrc", rc, zdot)
	}
}

func TestDetectShellRCBash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	rc, _, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	var want string
	if runtime.GOOS == "darwin" {
		want = filepath.Join(home, ".bash_profile")
	} else {
		want = filepath.Join(home, ".bashrc")
	}
	if rc != want {
		t.Errorf("rc = %q, want %q", rc, want)
	}
}

func TestDetectShellRCFish(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/usr/local/bin/fish")
	t.Setenv("XDG_CONFIG_HOME", "")

	rc, line, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	want := filepath.Join(home, ".config", "fish", "config.fish")
	if rc != want {
		t.Errorf("rc = %q, want %q", rc, want)
	}
	if !strings.HasPrefix(line, "fish_add_path ") {
		t.Errorf("line = %q, want fish_add_path prefix", line)
	}
}

func TestDetectShellRCFishWithXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	rc, _, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	want := filepath.Join(xdg, "fish", "config.fish")
	if rc != want {
		t.Errorf("rc = %q, want %q", rc, want)
	}
}

func TestDetectShellRCUnknown(t *testing.T) {
	t.Setenv("SHELL", "/bin/ksh")
	rc, _, err := detectShellRC("/opt/bin")
	if err != nil {
		t.Fatalf("detectShellRC: %v", err)
	}
	if rc != "" {
		t.Errorf("rc = %q, want empty for unknown shell", rc)
	}
}

func TestRCContainsLine(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, "rc")
	content := "# comment\nexport PATH=\"/opt/bin\":$PATH\nother\n"
	if err := os.WriteFile(rc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	has, err := rcContainsLine(rc, `export PATH="/opt/bin":$PATH`)
	if err != nil {
		t.Fatalf("rcContainsLine: %v", err)
	}
	if !has {
		t.Error("expected line to be present")
	}

	has, err = rcContainsLine(rc, "export PATH=/missing:$PATH")
	if err != nil {
		t.Fatalf("rcContainsLine: %v", err)
	}
	if has {
		t.Error("expected line to be absent")
	}
}

func TestRCContainsLineMissingFile(t *testing.T) {
	has, err := rcContainsLine(filepath.Join(t.TempDir(), "nope"), "x")
	if err != nil {
		t.Fatalf("rcContainsLine: want nil for missing file, got %v", err)
	}
	if has {
		t.Error("missing file should not contain line")
	}
}

func TestAppendRCCreatesAndAppends(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, "nested", "dir", "rc")
	line := `export PATH="/opt/bin":$PATH`

	if err := appendRC(rc, line); err != nil {
		t.Fatalf("appendRC: %v", err)
	}

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, line) {
		t.Errorf("rc missing line: got %q", got)
	}
	if !strings.Contains(got, "added by miru installer") {
		t.Errorf("rc missing marker comment: got %q", got)
	}
}

func TestAppendRCIdempotentViaContains(t *testing.T) {
	// Demonstrate the contains-then-append pattern used by configurePath.
	tmp := t.TempDir()
	rc := filepath.Join(tmp, "rc")
	line := `export PATH="/opt/bin":$PATH`

	for i := 0; i < 3; i++ {
		has, _ := rcContainsLine(rc, line)
		if !has {
			if err := appendRC(rc, line); err != nil {
				t.Fatal(err)
			}
		}
	}

	data, _ := os.ReadFile(rc)
	if got := strings.Count(string(data), line); got != 1 {
		t.Errorf("line written %d times, want 1", got)
	}
}

func TestVerifyInstallExecutable(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "miru"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verifyInstall(Config{InstallDir: tmp}); err != nil {
		t.Errorf("verifyInstall: %v", err)
	}
}

func TestVerifyInstallNotExecutable(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "miru"), []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyInstall(Config{InstallDir: tmp})
	if err == nil || !strings.Contains(err.Error(), "not executable") {
		t.Errorf("verifyInstall: want 'not executable', got %v", err)
	}
}

func TestVerifyInstallMissing(t *testing.T) {
	tmp := t.TempDir()
	if err := verifyInstall(Config{InstallDir: tmp}); err == nil {
		t.Error("verifyInstall: want error for missing binary, got nil")
	}
}

func TestFriendlyPathHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := friendlyPath(filepath.Join(home, "Projects", "miru"))
	want := "~/Projects/miru"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFriendlyPathOutsideHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got := friendlyPath("/usr/local/bin/miru")
	if got != "/usr/local/bin/miru" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestConfigurePathSkipsWhenInPATH(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("HOME", t.TempDir())

	rc, action, err := configurePath(Config{InstallDir: tmp})
	if err != nil {
		t.Fatalf("configurePath: %v", err)
	}
	if action != pathAlreadyInPath {
		t.Errorf("action = %v, want pathAlreadyInPath", action)
	}
	if rc != "" {
		t.Errorf("rc = %q, want empty", rc)
	}
}

func TestConfigurePathAppendsToRC(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(t.TempDir(), "bin")
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("ZDOTDIR", "")
	t.Setenv("PATH", "/usr/bin")

	rc, action, err := configurePath(Config{InstallDir: installDir})
	if err != nil {
		t.Fatalf("configurePath: %v", err)
	}
	if action != pathUpdated {
		t.Errorf("action = %v, want pathUpdated", action)
	}
	if rc != filepath.Join(home, ".zshrc") {
		t.Errorf("rc = %q", rc)
	}
	data, _ := os.ReadFile(rc)
	if !strings.Contains(string(data), installDir) {
		t.Errorf("rc missing installDir: %s", data)
	}
}

func TestConfigurePathDetectsAlreadyConfigured(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(t.TempDir(), "bin")
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("ZDOTDIR", "")
	t.Setenv("PATH", "/usr/bin")

	// First pass: should append.
	if _, _, err := configurePath(Config{InstallDir: installDir}); err != nil {
		t.Fatal(err)
	}
	// Second pass: should detect existing line.
	_, action, err := configurePath(Config{InstallDir: installDir})
	if err != nil {
		t.Fatal(err)
	}
	if action != pathAlreadyConfigured {
		t.Errorf("second call action = %v, want pathAlreadyConfigured", action)
	}
}

func TestConfigurePathUnknownShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/ksh")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "/usr/bin")

	_, _, err := configurePath(Config{InstallDir: t.TempDir()})
	if !errors.Is(err, errCouldNotDetectShell) {
		t.Errorf("got %v, want errCouldNotDetectShell", err)
	}
}
