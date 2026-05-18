// Package installer implements the `miru install` subcommand: a Bubble Tea
// driven, rich-UI installer that self-copies the running binary into
// INSTALL_DIR and (optionally) wires the user's shell rc PATH.
package installer

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/miru/internal/render"
)

// Version is injected at build time via -ldflags by GoReleaser. Falls back
// to module build info for `go install` builds, then "dev".
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			Version = v
		}
	}
}

// Run executes the installer. Returns a process exit code.
func Run(args []string) int {
	cfg := defaultConfig()

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--install-dir":
			i++
			if i < len(args) {
				cfg.InstallDir = args[i]
			}
		case "--no-modify-path":
			cfg.NoModifyPath = true
		case "--theme":
			i++
			if i < len(args) {
				cfg.Theme = args[i]
			}
		case "-h", "--help":
			printHelp()
			return 0
		}
	}

	m := newModel(cfg)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "installer:", err)
		return 1
	}
	if fm, ok := final.(model); ok && fm.failed {
		return 1
	}
	return 0
}

func printHelp() {
	fmt.Println("usage: miru install [--install-dir DIR] [--no-modify-path] [--theme NAME]")
	fmt.Println()
	fmt.Println("environment overrides:")
	fmt.Println("  INSTALL_DIR             target directory (default: $HOME/.local/bin)")
	fmt.Println("  MIRU_NO_MODIFY_PATH    set to 1 to skip shell rc PATH update")
	fmt.Println("  MIRU_THEME             color theme for installer UI")
}

type Config struct {
	InstallDir   string
	NoModifyPath bool
	Theme        string
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	cfg := Config{
		InstallDir: home + "/.local/bin",
		Theme:      render.DefaultTheme,
	}
	if v := os.Getenv("INSTALL_DIR"); v != "" {
		cfg.InstallDir = v
	}
	if os.Getenv("MIRU_NO_MODIFY_PATH") == "1" {
		cfg.NoModifyPath = true
	}
	if v := os.Getenv("MIRU_THEME"); v != "" {
		cfg.Theme = v
	}
	return cfg
}

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepDone
	stepSkipped
	stepFailed
)

type step struct {
	name   string
	detail string
	status stepStatus
}

type stepResult struct {
	index  int
	status stepStatus
	detail string
	err    error
}

type model struct {
	cfg      Config
	steps    []step
	finished bool
	failed   bool

	spinner  spinner.Model
	progress progress.Model

	accent color.Color
	muted  color.Color
	bad    color.Color
}

func newModel(cfg Config) model {
	accentHex := render.AccentColor(cfg.Theme)
	mutedHex := render.MutedColor(cfg.Theme)
	accent := lipgloss.Color(accentHex)
	muted := lipgloss.Color(mutedHex)
	bad := lipgloss.Color("#e06c75")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

	pr := progress.New(
		progress.WithColors(accent, muted),
		progress.WithoutPercentage(),
		progress.WithWidth(44),
	)

	return model{
		cfg:      cfg,
		spinner:  sp,
		progress: pr,
		accent:   accent,
		muted:    muted,
		bad:      bad,
		steps: []step{
			{name: "Install binary", status: stepRunning},
			{name: "Configure PATH"},
			{name: "Verify"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, runStep(m.cfg, 0))
}

func runStep(cfg Config, i int) tea.Cmd {
	return func() tea.Msg {
		// brief artificial latency so each transition is visually readable
		time.Sleep(260 * time.Millisecond)
		switch i {
		case 0:
			path, err := installBinary(cfg)
			if err != nil {
				return stepResult{index: i, err: err}
			}
			return stepResult{index: i, status: stepDone, detail: friendlyPath(path)}
		case 1:
			if cfg.NoModifyPath {
				return stepResult{index: i, status: stepSkipped, detail: "skipped (MIRU_NO_MODIFY_PATH=1)"}
			}
			rc, action, err := configurePath(cfg)
			if err != nil {
				if errors.Is(err, errCouldNotDetectShell) {
					return stepResult{index: i, status: stepSkipped, detail: "shell not detected; add to PATH manually"}
				}
				return stepResult{index: i, err: err}
			}
			var detail string
			switch action {
			case pathAlreadyInPath:
				detail = "already in PATH"
			case pathAlreadyConfigured:
				detail = friendlyPath(rc) + " (already)"
			default:
				detail = friendlyPath(rc)
			}
			return stepResult{index: i, status: stepDone, detail: detail}
		case 2:
			if err := verifyInstall(cfg); err != nil {
				return stepResult{index: i, err: err}
			}
			return stepResult{index: i, status: stepDone, detail: "binary OK"}
		}
		return stepResult{index: i, err: fmt.Errorf("unknown step %d", i)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := 44
		if msg.Width > 0 && w > msg.Width-8 {
			w = msg.Width - 8
		}
		m.progress.SetWidth(w)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	case stepResult:
		if msg.err != nil {
			m.steps[msg.index].status = stepFailed
			m.steps[msg.index].detail = msg.err.Error()
			m.failed = true
			m.finished = true
			return m, tea.Tick(1800*time.Millisecond, func(time.Time) tea.Msg { return tea.Quit() })
		}
		m.steps[msg.index].status = msg.status
		m.steps[msg.index].detail = msg.detail
		next := msg.index + 1
		pct := float64(next) / float64(len(m.steps))
		progCmd := m.progress.SetPercent(pct)
		if next >= len(m.steps) {
			m.finished = true
			return m, tea.Batch(
				progCmd,
				tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return tea.Quit() }),
			)
		}
		m.steps[next].status = stepRunning
		return m, tea.Batch(progCmd, runStep(m.cfg, next))
	}
	return m, nil
}

func (m model) View() tea.View {
	muted := lipgloss.NewStyle().Foreground(m.muted)
	accent := lipgloss.NewStyle().Foreground(m.accent)
	bold := lipgloss.NewStyle().Bold(true).Foreground(m.accent)
	bad := lipgloss.NewStyle().Foreground(m.bad)

	title := bold.Render("miru installer")
	subtitle := muted.Render(fmt.Sprintf("%s · %s/%s", Version, runtime.GOOS, runtime.GOARCH))

	var sb strings.Builder
	sb.WriteString(title + "  " + subtitle + "\n\n")
	for _, s := range m.steps {
		var icon string
		switch s.status {
		case stepPending:
			icon = muted.Render("○")
		case stepRunning:
			icon = m.spinner.View()
		case stepDone:
			icon = accent.Render("✓")
		case stepSkipped:
			icon = muted.Render("⊘")
		case stepFailed:
			icon = bad.Render("✗")
		}
		line := fmt.Sprintf(" %s  %-16s", icon, s.name)
		if s.detail != "" {
			line += "  " + muted.Render(s.detail)
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n " + m.progress.View() + "\n")

	if m.finished {
		sb.WriteString("\n")
		if m.failed {
			sb.WriteString(bad.Render(" install failed — see error above") + "\n")
		} else {
			sb.WriteString(" " + accent.Render("ready.") + " try it: " + bold.Render("miru sample.md") + "\n")
		}
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.accent).
		Padding(1, 2)
	return tea.NewView("\n" + box.Render(sb.String()) + "\n")
}
