package hatch

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var version = "0.1.0"

type cliOptions struct {
	cwdFile string
	init    string
	showVer bool
}

func Main(args []string, in io.Reader, out, errOut io.Writer) int {
	if err := run(args, in, out, errOut, time.Now); err != nil {
		fmt.Fprintln(errOut, errorStyle().Render("error: "+err.Error()))
		return 1
	}
	return 0
}

func run(args []string, in io.Reader, out, errOut io.Writer, now func() time.Time) error {
	options, remaining, usageText, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(out, usageText)
			return nil
		}
		return err
	}

	if options.showVer {
		fmt.Fprintf(out, "hatch %s\n", version)
		return nil
	}

	if options.init != "" {
		hook, err := shellInit(options.init)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, hook)
		return nil
	}

	root, err := hatcheryRoot()
	if err != nil {
		return err
	}

	switch len(remaining) {
	case 0:
		selected, err := runBrowser(root, in, out)
		if err != nil {
			if errors.Is(err, errNoSelection) {
				return nil
			}
			return err
		}
		if err := writeCWD(options.cwdFile, selected); err != nil {
			return err
		}
		fmt.Fprintln(out, successStyle().Render("Opened: "+selected))
		return nil
	case 1:
		projectPath, err := createProject(root, remaining[0], now())
		if err != nil {
			return err
		}
		if err := writeCWD(options.cwdFile, projectPath); err != nil {
			return err
		}
		fmt.Fprintln(out, successStyle().Render("Created: "+projectPath))
		return nil
	case 2:
		projectPath, err := copyProject(root, remaining[0], remaining[1], now())
		if err != nil {
			return err
		}
		if err := writeCWD(options.cwdFile, projectPath); err != nil {
			return err
		}
		fmt.Fprintln(out, successStyle().Render("Copied into: "+projectPath))
		return nil
	default:
		return fmt.Errorf("invalid argument count (%d)\n\n%s", len(remaining), usageText)
	}
}

func parseArgs(args []string) (cliOptions, []string, string, error) {
	var options cliOptions
	usageText := usage()
	fs := flag.NewFlagSet("hatch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&options.cwdFile, "cwd-file", "", "internal: write selected path to file")
	fs.StringVar(&options.init, "init", "", "print shell hook for zsh, bash, or fish")
	fs.BoolVar(&options.showVer, "version", false, "print version")
	fs.Usage = func() {}

	err := fs.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options, nil, usageText, flag.ErrHelp
		}
		return options, nil, usageText, fmt.Errorf("parse flags: %w", err)
	}

	return options, fs.Args(), usageText, nil
}

func usage() string {
	copy := []string{
		"hatch incubates short-lived projects in ~/hatchery.",
		"",
		"Usage:",
		"  hatch <name>",
		"      Create ~/hatchery/<yyyy-mm-dd>-<name> and enter it.",
		"",
		"  hatch <path> <name>",
		"      Copy <path> into ~/hatchery/<yyyy-mm-dd>-<name> and enter it.",
		"",
		"  hatch",
		"      Open the interactive browser with live fuzzy filtering.",
		"",
		"Actions in browser:",
		"  Enter     Open selected project",
		"  Ctrl+A    Archive selected project to ~/hatchery/archive",
		"  Ctrl+R    Remove selected project",
		"  Esc       Exit without selecting",
		"",
		"Shell integration (required for automatic cd):",
		"  eval \"$(hatch --init zsh)\"",
		"",
		"Options:",
		"  --init <shell>   Print shell hook for zsh, bash, or fish",
		"  --version        Print version",
		"  --help           Show this help message",
	}
	return strings.Join(copy, "\n") + "\n"
}

func writeCWD(cwdFile, path string) error {
	if strings.TrimSpace(cwdFile) == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path for cwd file: %w", err)
	}
	if err := os.WriteFile(cwdFile, []byte(abs), 0o644); err != nil {
		return fmt.Errorf("write cwd file: %w", err)
	}
	return nil
}

func successStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0F766E"))
}

func errorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B91C1C"))
}
