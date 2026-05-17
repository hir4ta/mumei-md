package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/hir4ta/miru/internal/server"
)

type browserOpenedMsg struct{ err error }

func openInBrowser(srv *server.Server, filename string) tea.Cmd {
	return func() tea.Msg {
		return browserOpenedMsg{err: srv.OpenInBrowser(filename)}
	}
}
