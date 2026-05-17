package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/hir4ta/miru/internal/nav"
	"github.com/hir4ta/miru/internal/render"
	"github.com/hir4ta/miru/internal/server"
)

type Model struct {
	filename   string
	raw        string
	theme      string
	isMarkdown bool
	srv        *server.Server

	ansi   *render.ANSI   // populated when isMarkdown
	source *render.Source // populated otherwise

	viewport viewport.Model
	help     help.Model
	keys     KeyMap

	headings []nav.Heading
	settings settingsModel
	helpOpen bool

	winW, winH int
	ready      bool
	err        error
}

func New(filename, raw, theme string, srv *server.Server) Model {
	return Model{
		filename:   filename,
		raw:        raw,
		theme:      theme,
		isMarkdown: render.IsMarkdown(filename),
		srv:        srv,
		keys:       DefaultKeyMap(),
		help:       help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Err returns the most recent error encountered during the session, or nil.
// Used by main to surface a non-zero exit code when rendering or persistence
// failed mid-run.
func (m Model) Err() error {
	return m.err
}
