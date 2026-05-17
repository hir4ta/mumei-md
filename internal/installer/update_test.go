package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEqualVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v0.1.0", "v0.1.0", true},
		{"v0.1.0", "0.1.0", true},
		{"0.1.0", "v0.1.0", true},
		{"v0.1.0", "v0.1.1", false},
		{"dev", "v0.1.0", false},
		{"v0.1.0", "dev", false},
		{"dev", "dev", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := equalVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("equalVersions(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestReplaceBinaryExtractsFromTarball(t *testing.T) {
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "release.tar.gz")
	dst := filepath.Join(tmp, "bin", "miru")
	want := []byte("\x7fELF-fake-binary-payload")

	if err := writeTarball(tarPath, map[string][]byte{
		"miru_0.1.0_darwin_arm64/README.md": []byte("hello"),
		"miru_0.1.0_darwin_arm64/miru":      want,
	}); err != nil {
		t.Fatalf("writeTarball: %v", err)
	}

	if err := replaceBinary(tarPath, dst); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("payload mismatch:\n got  %q\n want %q", got, want)
	}

	st, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0o111 == 0 {
		t.Error("destination is not executable")
	}
}

func TestReplaceBinaryMissingMiru(t *testing.T) {
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "release.tar.gz")
	dst := filepath.Join(tmp, "miru")

	if err := writeTarball(tarPath, map[string][]byte{
		"README.md": []byte("no binary here"),
	}); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(tarPath, dst); err == nil {
		t.Error("replaceBinary: want error when miru is absent from tarball")
	}
}

func TestReplaceBinaryAtomicSwap(t *testing.T) {
	// Pre-existing destination should be replaced cleanly without leaving
	// the ".new" temporary file behind.
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "release.tar.gz")
	dst := filepath.Join(tmp, "miru")
	if err := os.WriteFile(dst, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeTarball(tarPath, map[string][]byte{
		"miru": []byte("new"),
	}); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(tarPath, dst); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, _ := os.ReadFile(dst)
	if string(got) != "new" {
		t.Errorf("payload = %q, want %q", got, "new")
	}
	if _, err := os.Stat(dst + ".new"); !os.IsNotExist(err) {
		t.Errorf(".new tempfile should be gone, got %v", err)
	}
}

func TestVerifyChecksumMatches(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "miru_0.6.0_darwin_arm64.tar.gz")
	payload := []byte("pretend-this-is-a-tarball")
	if err := os.WriteFile(local, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	digest := sha256.Sum256(payload)
	checksums := fmt.Sprintf("%s  miru_0.6.0_darwin_arm64.tar.gz\n%s  miru_0.6.0_linux_amd64.tar.gz\n",
		hex.EncodeToString(digest[:]), "0000000000000000000000000000000000000000000000000000000000000000")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksums))
	}))
	defer srv.Close()

	if err := verifyChecksum(srv.URL, "miru_0.6.0_darwin_arm64.tar.gz", local); err != nil {
		t.Errorf("verifyChecksum: %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "miru_0.6.0_darwin_arm64.tar.gz")
	if err := os.WriteFile(local, []byte("real payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	checksums := "deadbeef  miru_0.6.0_darwin_arm64.tar.gz\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksums))
	}))
	defer srv.Close()

	err := verifyChecksum(srv.URL, "miru_0.6.0_darwin_arm64.tar.gz", local)
	if err == nil {
		t.Fatal("verifyChecksum: want mismatch error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("mismatch")) {
		t.Errorf("error should mention mismatch; got %v", err)
	}
}

func TestVerifyChecksumMissingEntry(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "miru_0.6.0_darwin_arm64.tar.gz")
	if err := os.WriteFile(local, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	checksums := "abcd1234  some-other-asset.tar.gz\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksums))
	}))
	defer srv.Close()

	err := verifyChecksum(srv.URL, "miru_0.6.0_darwin_arm64.tar.gz", local)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("no checksum entry")) {
		t.Errorf("want 'no checksum entry' error, got %v", err)
	}
}

func TestVerifyChecksumServerError(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "x.tar.gz")
	if err := os.WriteFile(local, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if err := verifyChecksum(srv.URL, "x.tar.gz", local); err == nil {
		t.Error("verifyChecksum: want error on HTTP 404, got nil")
	}
}

// writeTarball builds a gzipped tar archive containing the given entries.
func writeTarball(path string, entries map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, data := range entries {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	return nil
}
