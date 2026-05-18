package render

import (
	"os"
	"strings"
	"testing"
)

func TestToHTML(t *testing.T) {
	raw, err := os.ReadFile("../../sample.md")
	if err != nil {
		t.Fatal(err)
	}
	out, err := ToHTML("sample.md", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 1000 {
		t.Fatalf("html too small: %d bytes", len(out))
	}
	for _, want := range []string{
		"<!DOCTYPE html>",
		`class="markdown-body"`,
		"miru sample",
		"<table>",
		"<code",
		"sample.md",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("html missing %q", want)
		}
	}
	_ = os.WriteFile("/tmp/miru-test.html", out, 0o644)
	t.Logf("wrote /tmp/miru-test.html (%d bytes)", len(out))
}

func TestToHTML_FontsAlwaysLoadedMermaidGated(t *testing.T) {
	withMermaid := "# x\n\n```mermaid\nflowchart LR\n A --> B\n```\n"
	withoutMermaid := "# x\n\nplain markdown, no diagram.\n\n```go\nfunc main() {}\n```\n"

	// Every Markdown preview pulls Caveat / Patrick Hand / Kiwi Maru for the
	// paper-note look — regardless of mermaid presence.
	cases := map[string]string{"with-mermaid": withMermaid, "no-mermaid": withoutMermaid}
	for name, md := range cases {
		t.Run("fonts-always-loaded/"+name, func(t *testing.T) {
			out, err := ToHTML("x.md", md)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				`fonts.googleapis.com/css2?family=Caveat`,
				`family=Patrick+Hand`,
				`family=Kiwi+Maru`,
				`"Patrick Hand"`,
			} {
				if !strings.Contains(string(out), want) {
					t.Errorf("html missing %q", want)
				}
			}
		})
	}

	// Mermaid-specific assets stay gated on the presence of a mermaid block.
	out, err := ToHTML("x.md", withMermaid)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`look: "handDrawn"`,
		`cdn.jsdelivr.net/npm/mermaid@11`,
		`cdn.jsdelivr.net/npm/svg-pan-zoom`,
		`class="mermaid-zoom"`,
		`svgPanZoom(`,
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("with-mermaid: html missing %q", want)
		}
	}

	out, err = ToHTML("x.md", withoutMermaid)
	if err != nil {
		t.Fatal(err)
	}
	for _, dontWant := range []string{
		"handDrawn",
		"cdn.jsdelivr.net/npm/mermaid",
		"svg-pan-zoom",
		"mermaid-zoom",
	} {
		if strings.Contains(string(out), dontWant) {
			t.Errorf("no-mermaid: html should not contain %q", dontWant)
		}
	}
}

func TestToHTML_PaperNoteStyleTokens(t *testing.T) {
	// Regression guards for the paper-note look. These tokens are easy to
	// drop accidentally during a CSS rewrite, so pin them down.
	out, err := ToHTML("x.md", "# x\n\ntext\n\n---\n\nmore\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`lang="mul"`,        // multilingual <html> attribute
		`content: "〜〜〜"`,    // Caveat-glyph <hr> divider
		`--paper: #fdf6e3`,  // cream paper background variable
		`max-width: 1100px`, // widened page
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("paper-note html missing %q", want)
		}
	}
	// Old paper-note artefacts that have been removed must not creep back.
	// (Pin SVG-wave <hr> by its unique viewBox, not by `data:image/svg+xml`
	// alone — the upstream github-markdown.css legitimately uses data URIs.)
	for _, dontWant := range []string{
		`repeating-linear-gradient`, // the dropped horizontal ruling
		`viewBox='0 0 120 12'`,      // the dropped SVG-wave <hr>
		`"Klee One"`,                // replaced by Kiwi Maru
		`content: "※"`,              // dropped blockquote gutter marker
	} {
		if strings.Contains(string(out), dontWant) {
			t.Errorf("paper-note html should not contain %q (regression)", dontWant)
		}
	}
}

func TestSourceToHTML_StaysLocal(t *testing.T) {
	out, err := SourceToHTML("main.go", "package main\n\nfunc main() {}\n")
	if err != nil {
		t.Fatal(err)
	}
	// Source-file preview must not reach any third-party host — chroma
	// inlines its CSS and the page contains no scripts or remote fonts.
	for _, dontWant := range []string{
		"fonts.googleapis.com",
		"fonts.gstatic.com",
		"cdn.jsdelivr.net",
		"mermaid",
	} {
		if strings.Contains(string(out), dontWant) {
			t.Errorf("source html should not contain %q (must stay local)", dontWant)
		}
	}
}

func TestToHTML_AppliesNonceBasedCSP(t *testing.T) {
	// goldmark runs with WithUnsafe, so a Markdown file can contain a literal
	// <script>. Under the new loopback origin that script would be same-origin
	// with every other file under the server root and could exfiltrate them.
	// A nonce-based CSP keeps the legitimate inline mermaid loader executable
	// while making the browser block any <script> from user Markdown.
	md := "# Hi\n\n```mermaid\ngraph LR; A-->B\n```\n\n<script>fetch('/etc/passwd')</script>\n"
	out, err := ToHTML("doc.md", md)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)

	if !strings.Contains(s, `http-equiv="Content-Security-Policy"`) {
		t.Errorf("CSP meta tag missing")
	}
	if !strings.Contains(s, "default-src 'none'") {
		t.Errorf("CSP missing default-src 'none'")
	}
	if !strings.Contains(s, "script-src 'nonce-") {
		t.Errorf("CSP missing script-src nonce")
	}
	// Our inline module loader must carry the nonce; otherwise mermaid breaks.
	if !strings.Contains(s, `<script type="module" nonce="`) {
		t.Errorf("inline mermaid script missing nonce attribute")
	}
}

func TestToHTML_NonceIsFreshPerRender(t *testing.T) {
	out1, err := ToHTML("a.md", "```mermaid\nA-->B\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	out2, err := ToHTML("a.md", "```mermaid\nA-->B\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) == string(out2) {
		t.Errorf("expected different nonces between renders, got identical output")
	}
}

func TestSourceToHTML_EscapesFilename(t *testing.T) {
	// The filename is reflected into <title> and <h1> verbatim before this fix,
	// so any HTML in the basename (possible when the loopback server follows a
	// link to a hostile file name) would execute. Filenames must be escaped.
	out, err := SourceToHTML("<img src=x onerror=alert(1)>.txt", "hi\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "<img src=x onerror=alert(1)>") {
		t.Errorf("SourceToHTML rendered unescaped <img> from filename")
	}
	if !strings.Contains(string(out), "&lt;img src=x onerror=alert(1)&gt;") {
		t.Errorf("SourceToHTML did not HTML-escape filename")
	}
}
