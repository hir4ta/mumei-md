package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
}
.mermaid .nodeLabel,
.mermaid .edgeLabel,
.mermaid .cluster-label,
.mermaid text { font-family: inherit; }
{{- end}}
</style>
</head>
<body class="markdown-body">
{{.Body}}
{{- if .HasMermaid}}
<script type="module">
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
mermaid.run();
</script>
{{- end}}
</body>
</html>`

type htmlTemplateData struct {
	Title      string
	CSS        template.CSS
	Body       template.HTML
	HasMermaid bool
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

	var out bytes.Buffer
	err = tmpl.Execute(&out, htmlTemplateData{
		Title:      filepath.Base(filename),
		CSS:        template.CSS(css),
		Body:       template.HTML(body.String()),
		HasMermaid: hasMermaidBlock(markdown),
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

func OpenInBrowser(filename, content string) error {
	var htmlBytes []byte
	var err error
	if IsMarkdown(filename) {
		htmlBytes, err = ToHTML(filename, content)
	} else {
		htmlBytes, err = SourceToHTML(filename, content)
	}
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "miru-*.html")
	if err != nil {
		return err
	}
	if _, err := f.Write(htmlBytes); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return openCmd(f.Name())
}

func openCmd(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
