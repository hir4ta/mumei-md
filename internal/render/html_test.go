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
	_ = os.WriteFile("/tmp/miru-test.html", out, 0644)
	t.Logf("wrote /tmp/miru-test.html (%d bytes)", len(out))
}

func TestToHTML_FontsAlwaysLoadedMermaidGated(t *testing.T) {
	withMermaid := "# x\n\n```mermaid\nflowchart LR\n A --> B\n```\n"
	withoutMermaid := "# x\n\nplain markdown, no diagram.\n\n```go\nfunc main() {}\n```\n"

	// Every Markdown preview pulls Caveat / Patrick Hand / Klee One for the
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
	} {
		if strings.Contains(string(out), dontWant) {
			t.Errorf("no-mermaid: html should not contain %q", dontWant)
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
