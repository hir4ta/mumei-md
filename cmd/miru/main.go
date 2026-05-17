package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/hir4ta/miru/internal/config"
	"github.com/hir4ta/miru/internal/installer"
	"github.com/hir4ta/miru/internal/render"
	"github.com/hir4ta/miru/internal/tui"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "install":
			os.Exit(installer.Run(os.Args[2:]))
		case "update":
			os.Exit(installer.UpdateRun(os.Args[2:]))
		case "version":
			fmt.Println(installer.Version)
			return
		}
	}

	var (
		theme       string
		listThemes  bool
		showVersion bool
	)
	flag.StringVar(&theme, "theme", "", "color theme")
	flag.BoolVar(&listThemes, "list-themes", false, "list themes")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.Usage = usage
	flag.Parse()

	if showVersion {
		fmt.Println(installer.Version)
		return
	}

	if listThemes {
		for _, t := range render.AvailableThemes() {
			fmt.Println(t)
		}
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}
	path := args[0]

	resolvedTheme := resolveTheme(theme)

	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}

	m := tui.New(path, string(raw), resolvedTheme)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
	if fm, ok := final.(tui.Model); ok {
		if e := fm.Err(); e != nil {
			fmt.Fprintf(os.Stderr, "miru: %v\n", e)
			os.Exit(1)
		}
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  miru [--theme NAME] <file>               view a markdown, source, or config file
  miru install                             install/repair & wire up PATH
  miru update                              replace this binary with the latest release
  miru version                             print version

flags:
  --theme NAME                             color theme (see --list-themes)
  --list-themes                            list available themes and exit
  --version                                print version and exit`)
}

// resolveTheme picks the theme using precedence:
//
//	--theme flag > $MIRU_THEME > config file > render.DefaultTheme
func resolveTheme(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return flagValue
	}
	if env := strings.TrimSpace(os.Getenv("MIRU_THEME")); env != "" {
		return env
	}
	if cfg, err := config.Load(); err == nil && strings.TrimSpace(cfg.Theme) != "" {
		return cfg.Theme
	}
	return render.DefaultTheme
}
