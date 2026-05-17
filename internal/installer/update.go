package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/miru/internal/render"
)

const repo = "hir4ta/miru"

// UpdateRun executes the `miru update` subcommand. Returns a process exit code.
func UpdateRun(args []string) int {
	cfg := defaultConfig()

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--install-dir":
			i++
			if i < len(args) {
				cfg.InstallDir = args[i]
			}
		case "--theme":
			i++
			if i < len(args) {
				cfg.Theme = args[i]
			}
		case "-h", "--help":
			printUpdateHelp()
			return 0
		}
	}

	m := newUpdateModel(cfg)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		return 1
	}
	if fm, ok := final.(updateModel); ok && fm.failed {
		return 1
	}
	return 0
}

func printUpdateHelp() {
	fmt.Println("usage: miru update [--install-dir DIR] [--theme NAME]")
	fmt.Println()
	fmt.Println("Replaces the installed miru binary with the latest GitHub release.")
	fmt.Println()
	fmt.Println("environment overrides:")
	fmt.Println("  INSTALL_DIR    target install directory (default: $HOME/.local/bin)")
	fmt.Println("  MIRU_THEME    color theme for update UI")
}

type updateModel struct {
	cfg            Config
	currentVersion string
	latestVersion  string
	tarballPath    string

	steps    []step
	finished bool
	failed   bool
	upToDate bool

	spinner  spinner.Model
	progress progress.Model

	accent color.Color
	muted  color.Color
	bad    color.Color
}

type updateStepResult struct {
	index         int
	status        stepStatus
	detail        string
	err           error
	latestVersion string
	tarball       string
}

func newUpdateModel(cfg Config) updateModel {
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

	return updateModel{
		cfg:            cfg,
		currentVersion: Version,
		spinner:        sp,
		progress:       pr,
		accent:         accent,
		muted:          muted,
		bad:            bad,
		steps: []step{
			{name: "Resolve latest", status: stepRunning},
			{name: "Download"},
			{name: "Replace"},
			{name: "Verify"},
		},
	}
}

func (m updateModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, runUpdateStep(m, 0))
}

func runUpdateStep(m updateModel, i int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(260 * time.Millisecond)
		switch i {
		case 0:
			v, err := resolveLatestVersion(repo)
			if err != nil {
				return updateStepResult{index: i, err: err}
			}
			detail := v
			if equalVersions(m.currentVersion, v) {
				detail = v + " (current)"
			}
			return updateStepResult{index: i, status: stepDone, detail: detail, latestVersion: v}
		case 1:
			tarball, err := downloadLatest(m.latestVersion)
			if err != nil {
				return updateStepResult{index: i, err: err}
			}
			return updateStepResult{index: i, status: stepDone, detail: filepath.Base(tarball), tarball: tarball}
		case 2:
			dst := filepath.Join(m.cfg.InstallDir, "miru")
			if err := replaceBinary(m.tarballPath, dst); err != nil {
				return updateStepResult{index: i, err: err}
			}
			return updateStepResult{index: i, status: stepDone, detail: friendlyPath(dst)}
		case 3:
			if err := verifyInstall(m.cfg); err != nil {
				return updateStepResult{index: i, err: err}
			}
			return updateStepResult{index: i, status: stepDone, detail: "binary OK"}
		}
		return updateStepResult{index: i, err: fmt.Errorf("unknown step %d", i)}
	}
}

func (m updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case updateStepResult:
		if msg.err != nil {
			m.steps[msg.index].status = stepFailed
			m.steps[msg.index].detail = msg.err.Error()
			m.failed = true
			m.finished = true
			return m, tea.Tick(1800*time.Millisecond, func(time.Time) tea.Msg { return tea.Quit() })
		}
		m.steps[msg.index].status = msg.status
		m.steps[msg.index].detail = msg.detail
		if msg.latestVersion != "" {
			m.latestVersion = msg.latestVersion
		}
		if msg.tarball != "" {
			m.tarballPath = msg.tarball
		}

		// After Resolve, short-circuit if already on latest.
		if msg.index == 0 && equalVersions(m.currentVersion, m.latestVersion) {
			for j := 1; j < len(m.steps); j++ {
				m.steps[j].status = stepSkipped
				m.steps[j].detail = "—"
			}
			m.upToDate = true
			m.finished = true
			progCmd := m.progress.SetPercent(1.0)
			return m, tea.Batch(progCmd, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return tea.Quit() }))
		}

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
		return m, tea.Batch(progCmd, runUpdateStep(m, next))
	}
	return m, nil
}

func (m updateModel) View() tea.View {
	muted := lipgloss.NewStyle().Foreground(m.muted)
	accent := lipgloss.NewStyle().Foreground(m.accent)
	bold := lipgloss.NewStyle().Bold(true).Foreground(m.accent)
	bad := lipgloss.NewStyle().Foreground(m.bad)

	title := bold.Render("miru update")
	var subtitle string
	if m.latestVersion != "" {
		subtitle = muted.Render(fmt.Sprintf("%s → %s · %s/%s",
			m.currentVersion, m.latestVersion, runtime.GOOS, runtime.GOARCH))
	} else {
		subtitle = muted.Render(fmt.Sprintf("%s · %s/%s",
			m.currentVersion, runtime.GOOS, runtime.GOARCH))
	}

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
		line := fmt.Sprintf(" %s  %-15s", icon, s.name)
		if s.detail != "" {
			line += "  " + muted.Render(s.detail)
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n " + m.progress.View() + "\n")

	if m.finished {
		sb.WriteString("\n")
		switch {
		case m.failed:
			sb.WriteString(bad.Render(" update failed — see error above") + "\n")
		case m.upToDate:
			sb.WriteString(" " + accent.Render("already on latest") + " — no action needed\n")
		default:
			sb.WriteString(" " + accent.Render("updated.") + " new version active in any new shell\n")
		}
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.accent).
		Padding(1, 2)
	return tea.NewView("\n" + box.Render(sb.String()) + "\n")
}

// equalVersions compares two version strings ignoring an optional leading "v".
// "dev" never matches a release tag, ensuring local dev builds always proceed
// with the update.
func equalVersions(a, b string) bool {
	if a == "dev" || b == "dev" {
		return false
	}
	return strings.TrimPrefix(a, "v") == strings.TrimPrefix(b, "v")
}

// resolveLatestVersion queries GitHub's /releases/latest redirect to find the
// most recent published tag.
func resolveLatestVersion(repo string) (string, error) {
	req, err := http.NewRequest("HEAD", "https://github.com/"+repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve latest: %w", err)
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("no Location header on /releases/latest")
	}
	idx := strings.LastIndex(loc, "/")
	if idx < 0 {
		return "", fmt.Errorf("malformed Location: %s", loc)
	}
	return loc[idx+1:], nil
}

// downloadLatest fetches the platform-appropriate release tarball into a
// temp file, verifies its SHA-256 against the published checksums.txt, and
// returns the temp file path. A mismatch deletes the temp file and returns
// an error so a tampered or partial download cannot reach the install step.
func downloadLatest(version string) (string, error) {
	versionNoV := strings.TrimPrefix(version, "v")
	asset := fmt.Sprintf("miru_%s_%s_%s.tar.gz", versionNoV, runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, asset)
	checksumsURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", repo, version)

	tmp, err := os.CreateTemp("", "miru-update-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}

	if err := verifyChecksum(checksumsURL, asset, tmp.Name()); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// verifyChecksum fetches checksums.txt from checksumsURL, looks up the
// SHA-256 entry for asset, and confirms it matches the digest of localPath.
// Returns an error if the entry is missing or the digests differ — protects
// `miru update` from a tampered or partial tarball reaching the install step.
func verifyChecksum(checksumsURL, asset, localPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(checksumsURL)
	if err != nil {
		return fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch checksums: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	var expected string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum entry for %s", asset)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", asset, expected, actual)
	}
	return nil
}

// replaceBinary extracts the miru binary from tarball and atomically renames
// it onto dstBinary. Atomic rename onto a running executable is permitted
// on macOS and Linux.
func replaceBinary(tarball, dstBinary string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var binData bytes.Buffer
	found := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != "miru" {
			continue
		}
		if _, err := io.Copy(&binData, tr); err != nil {
			return err
		}
		found = true
		break
	}
	if !found {
		return errors.New("miru binary not found in tarball")
	}

	if err := os.MkdirAll(filepath.Dir(dstBinary), 0o755); err != nil {
		return err
	}
	tmpPath := dstBinary + ".new"
	if err := os.WriteFile(tmpPath, binData.Bytes(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dstBinary); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
