package render

import "testing"

// FuzzIsMarkdown feeds arbitrary filenames into the extension classifier.
// IsMarkdown is the switch between the glamour and chroma pipelines, so a
// panic here would break the very first dispatch in the binary.
func FuzzIsMarkdown(f *testing.F) {
	f.Add("test.md")
	f.Add("test.MD")
	f.Add("README.markdown")
	f.Add("file.tar.gz")
	f.Add("")
	f.Add(".md")
	f.Add("md")
	f.Add("/tmp/path/with/dots..//file.md")

	f.Fuzz(func(t *testing.T, filename string) {
		_ = IsMarkdown(filename)
	})
}

// FuzzLanguageFor exercises chroma's filename → lexer dispatch through
// detectLexer. A panic here would crash the TUI footer render every time
// the user opened a source file with that name.
func FuzzLanguageFor(f *testing.F) {
	f.Add("file.go")
	f.Add("Dockerfile")
	f.Add("Containerfile")
	f.Add("justfile")
	f.Add(".env")
	f.Add(".env.local")
	f.Add("")
	f.Add("name.with.many.dots.txt")

	f.Fuzz(func(t *testing.T, filename string) {
		_ = LanguageFor(filename)
	})
}
