package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func startServer(t *testing.T, root string) *Server {
	t.Helper()
	srv, err := Start(root)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	return srv
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestServer_RendersMarkdown(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("# Hello\n\n[link](./other.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, dir)

	url, err := srv.URLFor(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := get(t, url)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type %q, want text/html*", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Hello") {
		t.Errorf("body missing 'Hello': %s", string(body[:min(200, len(body))]))
	}
}

func TestServer_RendersSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, dir)

	url, _ := srv.URLFor(src)
	resp := get(t, url)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<html") {
		t.Errorf("body not wrapped as HTML")
	}
}

func TestServer_ServesBinaryRaw(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "tiny.png")
	payload := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(img, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, dir)

	url, _ := srv.URLFor(img)
	resp := get(t, url)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		t.Errorf("content-type %q, want image/*", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, payload) {
		t.Errorf("body altered for raw asset")
	}
}

func TestServer_RejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	evil := filepath.Join(other, "evil.md")
	if err := os.WriteFile(evil, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, root)

	url, _ := srv.URLFor(evil)
	resp := get(t, url)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", resp.StatusCode)
	}
}

func TestServer_NotFound(t *testing.T) {
	root := t.TempDir()
	srv := startServer(t, root)

	url, _ := srv.URLFor(filepath.Join(root, "missing.md"))
	resp := get(t, url)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", resp.StatusCode)
	}
}

func TestServer_FollowsRelativeMarkdownLink(t *testing.T) {
	// End-to-end check of the whole point of this server: render index.md,
	// pull the rendered <a href="..."> out of the response, resolve it
	// against the index URL the way a browser would, fetch the resolved
	// URL, and verify the linked file rendered.
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.md")
	otherPath := filepath.Join(dir, "other.md")
	if err := os.WriteFile(indexPath, []byte("# Index\n\n[next](./other.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherPath, []byte("# Other Page\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, dir)

	indexURL, err := srv.URLFor(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	indexResp := get(t, indexURL)
	if indexResp.StatusCode != http.StatusOK {
		t.Fatalf("index status %d", indexResp.StatusCode)
	}
	body, _ := io.ReadAll(indexResp.Body)

	hrefRe := regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>next</a>`)
	m := hrefRe.FindStringSubmatch(string(body))
	if m == nil {
		t.Fatalf("rendered HTML missing 'next' link: %s", string(body[:min(500, len(body))]))
	}
	base, err := url.Parse(indexURL)
	if err != nil {
		t.Fatal(err)
	}
	target, err := base.Parse(m[1])
	if err != nil {
		t.Fatal(err)
	}

	linkedResp := get(t, target.String())
	if linkedResp.StatusCode != http.StatusOK {
		t.Fatalf("linked status %d for %s", linkedResp.StatusCode, target)
	}
	linkedBody, _ := io.ReadAll(linkedResp.Body)
	if !strings.Contains(string(linkedBody), "Other Page") {
		t.Errorf("linked body missing 'Other Page'")
	}
}

func TestServer_RejectsSymlinkEscape(t *testing.T) {
	// A symlink living inside the root but pointing outside must be rejected;
	// otherwise the sandbox is bypassed via any attacker-placed symlink.
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(target, []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "trap.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	srv := startServer(t, root)

	// Build the URL by hand: we want to request the symlink path itself,
	// without URLFor's EvalSymlinks step that would canonicalise it away.
	addr := srv.listener.Addr().String()
	resp := get(t, "http://"+addr+link)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", resp.StatusCode)
	}
}

func TestServer_URLForEncodesUnicode(t *testing.T) {
	dir := t.TempDir()
	jp := filepath.Join(dir, "前提.md")
	if err := os.WriteFile(jp, []byte("# 前提\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, dir)

	url, err := srv.URLFor(jp)
	if err != nil {
		t.Fatal(err)
	}
	// URL string must be ASCII-safe — Japanese path segments are percent-encoded.
	for i, r := range url {
		if r > 127 {
			t.Fatalf("url[%d]=%q is non-ASCII; URL: %s", i, r, url)
		}
	}
	resp := get(t, url)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
