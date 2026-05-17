package nav

import (
	"strings"
	"testing"
)

// FuzzExtract feeds arbitrary markdown into Extract and asserts it does
// not panic and stays well-bounded. The check is intentionally minimal —
// any panic from goldmark on unusual input shows up as a test failure
// here without needing extra setup.
func FuzzExtract(f *testing.F) {
	f.Add("# heading\n\ntext\n## sub\n")
	f.Add("```\n# inside code, not a heading\n```\n# real\n")
	f.Add("")
	f.Add("---\n# heading after thematic break\n")
	f.Add("# h1\n## h2\n### h3\n#### h4\n##### h5\n###### h6\n")
	f.Add(strings.Repeat("# h\n", 50))

	f.Fuzz(func(t *testing.T, md string) {
		hs := Extract(md)
		for _, h := range hs {
			if h.Level < 1 || h.Level > 6 {
				t.Errorf("invalid heading level %d for text %q", h.Level, h.Text)
			}
		}
	})
}

// FuzzMapToLines feeds arbitrary headings + rendered text into the
// line-mapping pass and asserts it does not panic. Real input is the
// post-glamour render of the source markdown.
func FuzzMapToLines(f *testing.F) {
	f.Add("# alpha\n# beta\n", "# alpha\n# beta\n")
	f.Add("", "")
	f.Add("# x", "irrelevant lines\nwithout heading\n")

	f.Fuzz(func(t *testing.T, md, rendered string) {
		hs := Extract(md)
		out := MapToLines(hs, rendered)
		if len(out) > len(hs) {
			t.Errorf("MapToLines produced %d > %d input headings", len(out), len(hs))
		}
	})
}
