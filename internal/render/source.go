package render

import (
	"bytes"
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// IsMarkdown reports whether filename has a Markdown extension. It is the
// switch between the glamour pipeline (markdown) and the chroma pipeline
// (everything else).
func IsMarkdown(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown"
}

// LanguageFor returns a human-readable lexer name for the file, e.g.
// "Python", "Docker", "JSON". Used for the TUI footer chrome.
func LanguageFor(filename string) string {
	return detectLexer(filename).Config().Name
}

// Source renders non-markdown files as ANSI-styled text for the TUI viewport.
type Source struct {
	width int
	theme string
}

func NewSource(width int, theme string) (*Source, error) {
	return &Source{width: width, theme: theme}, nil
}

func (s *Source) Resize(width int) error {
	s.width = width
	return nil
}

// Render syntax-highlights content for the terminal and prefixes each line
// with a dim line number.
func (s *Source) Render(filename, content string) (string, error) {
	lexer := chroma.Coalesce(detectLexer(filename))
	iter, err := lexer.Tokenise(nil, content)
	if err != nil {
		return "", fmt.Errorf("tokenise: %w", err)
	}

	style := chromaStyleFor(s.theme)
	formatter := formatters.Get("terminal16m")

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return "", fmt.Errorf("format: %w", err)
	}
	return prefixLineNumbers(buf.String()), nil
}

// SourceToHTML renders content as a chroma-highlighted standalone HTML page
// for the browser open path.
func SourceToHTML(filename, content string) ([]byte, error) {
	lexer := chroma.Coalesce(detectLexer(filename))
	iter, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil, fmt.Errorf("tokenise: %w", err)
	}

	style := styles.Get("github-dark")
	if style == nil {
		style = styles.Fallback
	}

	formatter := chromahtml.New(
		chromahtml.WithLineNumbers(true),
		chromahtml.LineNumbersInTable(true),
		chromahtml.WithClasses(true),
	)

	var codeBuf bytes.Buffer
	if err := formatter.Format(&codeBuf, style, iter); err != nil {
		return nil, fmt.Errorf("format html: %w", err)
	}
	var cssBuf bytes.Buffer
	if err := formatter.WriteCSS(&cssBuf, style); err != nil {
		return nil, fmt.Errorf("write css: %w", err)
	}

	title := html.EscapeString(filepath.Base(filename))
	lang := html.EscapeString(lexer.Config().Name)
	var out bytes.Buffer
	fmt.Fprintf(&out, sourceHTMLTemplate, title, cssBuf.String(), title, lang, codeBuf.String())
	return out.Bytes(), nil
}

// detectLexer picks a chroma lexer by filename, falling back to plaintext.
// chroma already handles most extensions and special filenames (Dockerfile,
// Makefile, etc.); we add fallbacks only for things chroma misses.
func detectLexer(filename string) chroma.Lexer {
	base := filepath.Base(filename)
	if l := lexers.Match(base); l != nil {
		return l
	}
	switch strings.ToLower(base) {
	case "justfile":
		return lexers.Get("Makefile")
	case "taskfile.yml", "taskfile.yaml":
		return lexers.Get("YAML")
	case "containerfile":
		return lexers.Get("Docker")
	}
	if strings.HasPrefix(strings.ToLower(base), ".env") {
		return lexers.Get("Bash")
	}
	return lexers.Fallback
}

// chromaStyleFor picks a chroma style that pairs reasonably with the active
// glamour theme. Falls back to github-dark for themes without a matching
// chroma style.
func chromaStyleFor(theme string) *chroma.Style {
	name := "github-dark"
	switch theme {
	case "gruvbox":
		name = "gruvbox"
	case "nord":
		name = "nord"
	case "dracula":
		name = "dracula"
	case "tokyo-night":
		name = "tokyonight-night"
	case "everforest":
		name = "solarized-dark"
	}
	if st := styles.Get(name); st != nil {
		return st
	}
	return styles.Fallback
}

const (
	lineNumSeq = "\x1b[2;90m"
	resetSeq   = "\x1b[0m"
)

func prefixLineNumbers(rendered string) string {
	rendered = strings.TrimRight(rendered, "\n")
	lines := strings.Split(rendered, "\n")
	width := len(strconv.Itoa(len(lines)))
	if width < 3 {
		width = 3
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = fmt.Sprintf("%s%*d │%s %s", lineNumSeq, width, i+1, resetSeq, line)
	}
	return strings.Join(out, "\n") + "\n"
}

const sourceHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
:root { color-scheme: dark light; }
body {
  margin: 0 auto;
  max-width: 1100px;
  padding: 45px 30px;
  font-family: ui-monospace, "SF Mono", "Cascadia Mono", Menlo, Consolas, monospace;
  font-size: 14px;
  line-height: 1.5;
  background: #0d1117;
  color: #e6edf3;
}
h1 {
  font-size: 16px;
  font-weight: 500;
  color: #7d8590;
  margin: 0 0 1em 0;
  padding-bottom: .5em;
  border-bottom: 1px solid #30363d;
  font-family: ui-sans-serif, -apple-system, "Segoe UI", "Helvetica Neue", sans-serif;
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
h1 .lang {
  font-size: 12px;
  color: #6e7681;
  text-transform: lowercase;
  letter-spacing: 0.05em;
}
pre { margin: 0; }
table.lntable {
  border-collapse: collapse;
  width: 100%%;
}
table.lntable td.lntd:first-child {
  user-select: none;
  text-align: right;
  padding-right: 1em;
  color: #6e7681;
  border-right: 1px solid #30363d;
}
table.lntable td.lntd:last-child {
  padding-left: 1em;
  width: 100%%;
}
@media (max-width: 767px) { body { padding: 15px; } }
%s
</style>
</head>
<body>
<h1>%s<span class="lang">%s</span></h1>
%s
</body>
</html>
`
