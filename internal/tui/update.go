package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/miru/internal/config"
	"github.com/hir4ta/miru/internal/nav"
	"github.com/hir4ta/miru/internal/render"
)

func (m *Model) layout() {
	if m.winW == 0 || m.winH == 0 {
		return
	}
	headerH := lipgloss.Height(m.headerView())
	footerH := lipgloss.Height(m.footerView())
	helpH := lipgloss.Height(m.help.View(m.keys))
	m.viewport.SetWidth(m.winW)
	m.viewport.SetHeight(m.winH - headerH - footerH - helpH)
	m.help.SetWidth(m.winW)
}

func (m *Model) renderContent() error {
	var (
		rendered string
		err      error
	)
	if m.isMarkdown {
		rendered, err = m.ansi.Render(m.raw)
		if err != nil {
			return err
		}
		m.headings = nav.MapToLines(nav.Extract(m.raw), rendered)
	} else {
		rendered, err = m.source.Render(m.filename, m.raw)
		if err != nil {
			return err
		}
		m.headings = nil
	}
	m.viewport.SetContent(rendered)
	return nil
}

// applyTheme rebuilds the active renderer with the given theme, re-renders
// the content, and persists the choice to the config file. Failures are
// recorded in m.err so they surface on exit instead of failing silently.
func (m *Model) applyTheme(name string) {
	if m.isMarkdown {
		ansi, err := render.NewANSI(m.winW, name)
		if err != nil {
			m.err = err
			return
		}
		m.ansi = ansi
	} else {
		src, err := render.NewSource(m.winW, name)
		if err != nil {
			m.err = err
			return
		}
		m.source = src
	}
	m.theme = name
	if err := m.renderContent(); err != nil {
		m.err = err
		return
	}
	if err := config.Save(config.Config{Theme: name}); err != nil {
		m.err = err
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.winW, m.winH = msg.Width, msg.Height

		if !m.ready {
			if m.isMarkdown {
				ansi, err := render.NewANSI(m.winW, m.theme)
				if err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.ansi = ansi
			} else {
				src, err := render.NewSource(m.winW, m.theme)
				if err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.source = src
			}
			m.viewport = viewport.New(
				viewport.WithWidth(m.winW),
				viewport.WithHeight(m.winH),
			)
			m.settings = newSettings(render.AvailableThemes(), m.theme, m.accent(), m.muted())
			m.ready = true
		} else if m.isMarkdown {
			_ = m.ansi.Resize(m.winW)
		} else {
			_ = m.source.Resize(m.winW)
		}

		if err := m.renderContent(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.layout()

	case tea.KeyPressMsg:
		// Settings overlay captures input when open.
		if m.settings.open {
			switch msg.String() {
			case "esc", "s", "q":
				m.settings.open = false
				return m, nil
			case "enter":
				if it, ok := m.settings.list.SelectedItem().(themeItem); ok {
					m.applyTheme(it.name)
					m.settings.list = newSettings(render.AvailableThemes(), m.theme, m.accent(), m.muted()).list
				}
				m.settings.open = false
				return m, nil
			}
			var lc tea.Cmd
			m.settings.list, lc = m.settings.list.Update(msg)
			return m, lc
		}

		// Help overlay captures input when open.
		if m.helpOpen {
			switch msg.String() {
			case "esc", "?", "q":
				m.helpOpen = false
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Settings):
			m.settings.list.Select(indexOfTheme(render.AvailableThemes(), m.theme))
			m.settings.open = true
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.helpOpen = true
			return m, nil
		case key.Matches(msg, m.keys.Top):
			m.viewport.GotoTop()
		case key.Matches(msg, m.keys.Bottom):
			m.viewport.GotoBottom()
		case key.Matches(msg, m.keys.PrevSection):
			if line, ok := nav.Prev(m.headings, m.viewport.YOffset()); ok {
				m.viewport.SetYOffset(line)
			} else {
				m.viewport.GotoTop()
			}
		case key.Matches(msg, m.keys.NextSection):
			if line, ok := nav.Next(m.headings, m.viewport.YOffset()); ok {
				m.viewport.SetYOffset(line)
			}
		case key.Matches(msg, m.keys.Browser):
			cmds = append(cmds, openInBrowser(m.filename, m.raw))
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func indexOfTheme(themes []string, target string) int {
	for i, t := range themes {
		if t == target {
			return i
		}
	}
	return 0
}
