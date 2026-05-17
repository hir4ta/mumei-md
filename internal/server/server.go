// Package server runs a loopback HTTP server that re-renders Markdown (and
// neighbouring source files) on demand, so that relative links in the rendered
// HTML keep working when the user clicks through to another file in the same
// tree.
package server

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hir4ta/miru/internal/render"
)

type Server struct {
	listener net.Listener
	server   *http.Server
	root     string
}

// Start binds a TCP listener on 127.0.0.1 with an OS-assigned port and begins
// serving. Requests are sandboxed to rootDir (after symlink resolution); any
// path that escapes the root is rejected with 403. Call Stop to shut down.
func Start(rootDir string) (*Server, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	s := &Server{listener: ln, root: root}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = s.server.Serve(ln) }()
	return s, nil
}

// Stop gracefully shuts the server down with a short timeout.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// URLFor builds the loopback URL that renders the given file. The path is
// resolved to absolute form and symlinks are followed so the URL matches
// where the file actually lives; relative links inside the rendered HTML
// then resolve relative to the real file's directory rather than the
// symlink's parent. If the file does not exist (e.g. a URL for a broken
// link the caller still wants to fetch), the parent directory is
// canonicalised instead so the resulting URL still survives the server's
// lexical sandbox check.
func (s *Server) URLFor(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	} else if dir, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		abs = filepath.Join(dir, filepath.Base(abs))
	}
	u := url.URL{
		Scheme: "http",
		Host:   s.listener.Addr().String(),
		Path:   abs,
	}
	return u.String(), nil
}

// OpenInBrowser opens the platform browser pointed at the given file.
func (s *Server) OpenInBrowser(path string) error {
	target, err := s.URLFor(path)
	if err != nil {
		return err
	}
	return openCmd(target)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	requested := filepath.Clean(r.URL.Path)
	if !filepath.IsAbs(requested) {
		http.Error(w, "absolute path required", http.StatusBadRequest)
		return
	}
	// Lexical sandbox check FIRST. Calling Stat / EvalSymlinks on an outside
	// path would leak existence (200/404/500 vs 403) and let a probe map the
	// filesystem; reject before touching disk.
	if !pathInside(requested, s.root) {
		http.Error(w, "outside server root", http.StatusForbidden)
		return
	}
	resolved, err := filepath.EvalSymlinks(requested)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Re-check after symlink resolution: a symlink inside the root may point
	// to a target outside it.
	if !pathInside(resolved, s.root) {
		http.Error(w, "outside server root", http.StatusForbidden)
		return
	}
	info, err := os.Stat(resolved)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}
	if err := s.serveFile(w, resolved); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serveFile(w http.ResponseWriter, path string) error {
	if render.IsMarkdown(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		html, err := render.ToHTML(path, string(data))
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err = w.Write(html)
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if looksLikeText(data, path) {
		html, err := render.SourceToHTML(path, string(data))
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err = w.Write(html)
		return err
	}
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, err = w.Write(data)
	return err
}

func pathInside(target, root string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// looksLikeText decides whether a non-markdown asset should be wrapped in
// SourceToHTML (syntax-highlighted preview) or served as raw bytes. Anything
// not clearly text — images, archives, wasm, executables — is served raw so
// the browser can render it natively.
func looksLikeText(data []byte, path string) bool {
	if ct := mime.TypeByExtension(filepath.Ext(path)); strings.HasPrefix(ct, "text/") {
		return true
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	return strings.HasPrefix(http.DetectContentType(head), "text/")
}

func openCmd(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
