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
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

//go:embed assets/*
var assetsFS embed.FS

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>{{.CSS}}</style>
<style>
:root { color-scheme: dark light; }
body.markdown-body {
  box-sizing: border-box;
  min-width: 200px;
  max-width: 980px;
  margin: 0 auto;
  padding: 45px;
}
@media (max-width: 767px) { body.markdown-body { padding: 15px; } }
.mermaid { display: flex; justify-content: center; margin: 1em 0; }
.mermaid svg { max-width: 100%; height: auto; }
</style>
</head>
<body class="markdown-body">
{{.Body}}
<script type="module">
import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default", securityLevel: "loose" });
document.querySelectorAll("pre > code.language-mermaid").forEach(code => {
  const div = document.createElement("div");
  div.className = "mermaid";
  div.textContent = code.textContent;
  code.parentElement.replaceWith(div);
});
mermaid.run();
</script>
</body>
</html>`

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
	err = tmpl.Execute(&out, struct {
		Title string
		CSS   template.CSS
		Body  template.HTML
	}{
		Title: filepath.Base(filename),
		CSS:   template.CSS(css),
		Body:  template.HTML(body.String()),
	})
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
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
