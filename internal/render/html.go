package render

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"path/filepath"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

//go:embed assets/*
var assetsFS embed.FS

const htmlTemplate = `<!DOCTYPE html>
<html lang="mul">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-{{.Nonce}}' 'strict-dynamic'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'; base-uri 'none'; form-action 'none'">
<title>{{.Title}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Caveat:wght@400..700&family=Patrick+Hand&family=Kiwi+Maru:wght@400;500&display=swap" rel="stylesheet">
<style>{{.CSS}}</style>
<style>
:root { color-scheme: light; --paper: #fdf6e3; --ink: #3a342c; --accent: #d97757; --muted: #7a6c52; --highlight: #ffe599; }
html, body { background: var(--paper); }
body.markdown-body {
  box-sizing: border-box;
  min-width: 200px;
  max-width: 1100px;
  margin: 0 auto;
  padding: 56px 64px;
  background-color: var(--paper);
  color: var(--ink);
  font-family: "Patrick Hand", "Kiwi Maru", "ヒラギノ丸ゴ ProN", "Hiragino Maru Gothic ProN", "Yu Gothic", system-ui, sans-serif;
  font-size: 18px;
  line-height: 1.7em;
}
@media (max-width: 767px) { body.markdown-body { padding: 24px 18px; } }
.markdown-body h1, .markdown-body h2, .markdown-body h3,
.markdown-body h4, .markdown-body h5, .markdown-body h6 {
  font-family: "Caveat", "Kiwi Maru", cursive;
  font-weight: 700;
  color: var(--accent);
  line-height: 1.2;
  border-bottom: none;
  padding-bottom: 0;
  margin-top: 1.5em;
  margin-bottom: 0.5em;
}
.markdown-body h1 { font-size: 2.6em; }
.markdown-body h2 { font-size: 2.0em; }
.markdown-body h3 { font-size: 1.55em; color: var(--ink); }
.markdown-body h4 { font-size: 1.25em; color: var(--ink); }
.markdown-body h1, .markdown-body h2 {
  border-bottom: 2px dashed var(--muted);
  padding-bottom: 0.1em;
}
.markdown-body p, .markdown-body li { color: var(--ink); }
.markdown-body strong {
  font-weight: inherit;
  background: linear-gradient(transparent 55%, var(--highlight) 55%);
  padding: 0 2px;
}
.markdown-body mark {
  background: linear-gradient(transparent 55%, var(--highlight) 55%);
  color: inherit;
  padding: 0 2px;
}
.markdown-body em {
  font-style: italic;
  color: var(--accent);
}
.markdown-body a {
  color: var(--accent);
  text-decoration-line: underline;
  text-decoration-style: wavy;
  text-decoration-thickness: 1px;
  text-underline-offset: 3px;
}
.markdown-body blockquote {
  border-left: 4px solid var(--accent);
  color: var(--muted);
  font-style: italic;
  padding: 0 1em;
  margin: 1em 0;
}
.markdown-body ul, .markdown-body ol { padding-left: 1.6em; }
.markdown-body ul { list-style: none; }
.markdown-body ul li { position: relative; }
.markdown-body ul li::before {
  content: "●";
  color: var(--accent);
  position: absolute;
  left: -1.2em;
  top: 0;
}
.markdown-body ul ul li::before { content: "○"; }
.markdown-body ul ul ul li::before { content: "▪"; }
.markdown-body ul ul ul ul li::before { content: "◦"; }
.markdown-body ul li.task-list-item::before { content: ""; }
.markdown-body .task-list-item-checkbox {
  appearance: none;
  -webkit-appearance: none;
  width: 1em;
  height: 1em;
  border: 1.5px solid var(--muted);
  border-radius: 2px;
  vertical-align: -2px;
  margin-right: 0.4em;
  position: relative;
  background: transparent;
}
.markdown-body .task-list-item-checkbox:checked::after {
  content: "✓";
  font-family: "Caveat", cursive;
  font-size: 1.4em;
  font-weight: 700;
  color: var(--accent);
  position: absolute;
  left: -2px;
  top: -10px;
}
.markdown-body hr {
  border: 0;
  height: auto;
  background: transparent;
  text-align: center;
  margin: 2em 0;
  position: relative;
  line-height: 1;
}
.markdown-body hr::after {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  border-top: 1px solid var(--muted);
  opacity: 0.35;
  z-index: 0;
}
.markdown-body hr::before {
  content: "〜〜〜";
  display: inline-block;
  position: relative;
  z-index: 1;
  background-color: var(--paper);
  padding: 0 0.6em;
  font-family: "Caveat", "Kiwi Maru", cursive;
  font-weight: 700;
  font-size: 1.6em;
  color: var(--muted);
  letter-spacing: 0.15em;
}
.markdown-body table {
  border-collapse: collapse;
  margin: 1em 0;
  background: transparent;
}
.markdown-body table th, .markdown-body table td {
  border: 1.5px solid var(--muted);
  padding: 0.4em 0.9em;
  background: transparent;
  color: var(--ink);
}
.markdown-body table th {
  font-family: "Caveat", "Kiwi Maru", cursive;
  font-weight: 700;
  font-size: 1.2em;
  color: var(--accent);
  background: transparent;
}
.markdown-body table tr {
  background: transparent !important;
  border-top: none;
}
.markdown-body code {
  font-family: ui-monospace, "SF Mono", "Cascadia Mono", Menlo, Consolas, monospace;
  font-size: 0.9em;
  background: rgba(122, 108, 82, 0.12);
  border-radius: 3px;
  padding: 0.1em 0.35em;
  color: var(--ink);
}
.markdown-body pre {
  background: #1f1e1d !important;
  color: #f0eee6;
  border-radius: 6px;
  padding: 14px 18px !important;
  line-height: 1.45;
  box-shadow: 2px 2px 0 var(--muted);
}
.markdown-body pre code {
  background: transparent;
  padding: 0;
  color: inherit;
  font-size: 0.9em;
}
{{- if .HasMermaid}}
.mermaid { display: flex; justify-content: center; margin: 1.5em 0; }
.mermaid svg {
  max-width: 100%;
  height: auto;
  font-family: "Caveat", "Virgil", cursive;
  font-size: 18px;
  cursor: zoom-in;
}
.mermaid .nodeLabel,
.mermaid .edgeLabel,
.mermaid .cluster-label,
.mermaid text { font-family: inherit; }
.mermaid-zoom {
  position: fixed;
  inset: 0;
  background: rgba(31, 30, 29, 0.78);
  display: none;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.mermaid-zoom.open { display: flex; }
.mermaid-zoom__stage {
  width: 90vw;
  height: 90vh;
  background: var(--paper);
  border-radius: 8px;
  box-shadow: 0 6px 32px rgba(0, 0, 0, 0.45);
  padding: 28px;
  box-sizing: border-box;
  position: relative;
  overflow: hidden;
}
.mermaid-zoom__stage > svg { width: 100%; height: 100%; cursor: grab; }
.mermaid-zoom__stage > svg:active { cursor: grabbing; }
.mermaid-zoom__close {
  position: absolute;
  top: 8px;
  right: 14px;
  background: transparent;
  border: 0;
  color: var(--muted);
  font-family: "Caveat", cursive;
  font-weight: 700;
  font-size: 2em;
  line-height: 1;
  cursor: pointer;
  padding: 4px 8px;
}
.mermaid-zoom__close:hover { color: var(--accent); }
.mermaid-zoom__hint {
  position: absolute;
  bottom: 10px;
  left: 0;
  right: 0;
  text-align: center;
  color: var(--muted);
  font-family: "Caveat", cursive;
  font-size: 1.2em;
  pointer-events: none;
}
{{- end}}
</style>
</head>
<body class="markdown-body">
{{.Body}}
{{- if .HasMermaid}}
<div class="mermaid-zoom" id="mermaid-zoom" role="dialog" aria-modal="true" aria-hidden="true" aria-label="Diagram zoom view">
  <div class="mermaid-zoom__stage" id="mermaid-zoom-stage">
    <button type="button" class="mermaid-zoom__close" id="mermaid-zoom-close" aria-label="Close diagram zoom">×</button>
    <div class="mermaid-zoom__hint">scroll to zoom · drag to pan · esc to close</div>
  </div>
</div>
<script src="https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.2/dist/svg-pan-zoom.min.js" nonce="{{.Nonce}}"></script>
<script type="module" nonce="{{.Nonce}}">
import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
mermaid.initialize({
  startOnLoad: false,
  look: "handDrawn",
  theme: "neutral",
  securityLevel: "loose",
  fontFamily: "Caveat, Virgil, cursive",
});
document.querySelectorAll("pre > code.language-mermaid").forEach(code => {
  const div = document.createElement("div");
  div.className = "mermaid";
  div.textContent = code.textContent;
  code.parentElement.replaceWith(div);
});
await mermaid.run();

const modal = document.getElementById("mermaid-zoom");
const stage = document.getElementById("mermaid-zoom-stage");
const closeBtn = document.getElementById("mermaid-zoom-close");
let panZoom = null;
let previousFocus = null;

function closeModal() {
  if (panZoom) { panZoom.destroy(); panZoom = null; }
  stage.querySelectorAll("svg").forEach(svg => svg.remove());
  modal.classList.remove("open");
  modal.setAttribute("aria-hidden", "true");
  if (previousFocus && typeof previousFocus.focus === "function") {
    try { previousFocus.focus(); } catch (_) { /* element may be gone */ }
  }
  previousFocus = null;
}
function openModal(sourceSvg) {
  previousFocus = document.activeElement;
  const clone = sourceSvg.cloneNode(true);
  clone.removeAttribute("style");
  clone.setAttribute("width", "100%");
  clone.setAttribute("height", "100%");
  stage.appendChild(clone);
  modal.classList.add("open");
  modal.setAttribute("aria-hidden", "false");
  closeBtn.focus();
  panZoom = svgPanZoom(clone, {
    controlIconsEnabled: false,
    fit: true,
    center: true,
    zoomScaleSensitivity: 0.3,
    minZoom: 0.3,
    maxZoom: 20,
  });
}
closeBtn.addEventListener("click", closeModal);
modal.addEventListener("click", e => { if (e.target === modal) closeModal(); });
document.addEventListener("keydown", e => {
  if (!modal.classList.contains("open")) return;
  if (e.key === "Escape") { closeModal(); return; }
  if (e.key === "Tab") {
    // The dialog only exposes one focusable element (the close button);
    // keep keyboard navigation inside the dialog while it is open.
    e.preventDefault();
    closeBtn.focus();
  }
});
// Defensive trap: if focus escapes (e.g. assistive tech jump), pull it back.
document.addEventListener("focusin", e => {
  if (modal.classList.contains("open") && !modal.contains(e.target)) {
    closeBtn.focus();
  }
});
// Make inline diagrams keyboard-accessible: Enter / Space opens the zoom modal.
document.querySelectorAll(".mermaid > svg").forEach(svg => {
  svg.setAttribute("role", "button");
  svg.setAttribute("tabindex", "0");
  svg.setAttribute("aria-label", "Open diagram in zoom view");
  svg.addEventListener("click", () => openModal(svg));
  svg.addEventListener("keydown", e => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      openModal(svg);
    }
  });
});
</script>
{{- end}}
</body>
</html>`

type htmlTemplateData struct {
	Title      string
	CSS        template.CSS
	Body       template.HTML
	HasMermaid bool
	Nonce      string
}

// makeNonce returns a base64-encoded random nonce for use in the page's CSP
// `script-src` directive. A fresh nonce per render keeps the inline mermaid
// loader executable while blocking any unauthorised `<script>` smuggled in
// from raw HTML in the user's Markdown. An error here means crypto/rand is
// not functioning; serving a page with a weak nonce would advertise a CSP
// that does not actually protect anything, so we propagate the failure.
func makeNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func ToHTML(filename, markdown string) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github-dark"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	var body bytes.Buffer
	if err := md.Convert([]byte(markdown), &body); err != nil {
		return nil, fmt.Errorf("goldmark convert: %w", err)
	}

	css, err := assetsFS.ReadFile("assets/github-markdown.css")
	if err != nil {
		return nil, fmt.Errorf("read css: %w", err)
	}

	tmpl, err := template.New("page").Parse(htmlTemplate)
	if err != nil {
		return nil, err
	}

	nonce, err := makeNonce()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, htmlTemplateData{
		Title:      filepath.Base(filename),
		CSS:        template.CSS(css),
		Body:       template.HTML(body.String()),
		HasMermaid: hasMermaidBlock(markdown),
		Nonce:      nonce,
	})
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// hasMermaidBlock reports whether the markdown contains at least one fenced
// code block declared as `mermaid`. Used to gate mermaid.js loading so plain
// Markdown previews avoid the jsDelivr fetch.
func hasMermaidBlock(markdown string) bool {
	source := []byte(markdown)
	root := goldmark.New().Parser().Parse(text.NewReader(source))
	found := false
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		cb, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}
		if string(cb.Language(source)) == "mermaid" {
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

