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
var cloneProjectFn = cloneProject

type cliOptions struct {
	cwdFile string
	init    string
	showVer bool
	showUse bool
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

	if options.showUse {
		fmt.Fprintln(out, usageCard())
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
		var (
			projectPath string
			action      string
		)
		if isGitURL(remaining[0]) {
			projectPath, err = cloneProjectFn(root, remaining[0], now())
			action = "Cloned into: "
		} else {
			projectPath, err = createProject(root, remaining[0], now())
			action = "Created: "
		}
		if err != nil {
			return err
		}
		if err := writeCWD(options.cwdFile, projectPath); err != nil {
			return err
		}
		fmt.Fprintln(out, successStyle().Render(action+projectPath))
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
	fs.BoolVar(&options.showUse, "usage", false, "show styled usage guide")
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
		"  hatch <git-url>",
		"      Clone ssh/https git URL into ~/hatchery/<yyyy-mm-dd>-<repo-name> and enter it.",
		"",
		"  hatch <path> <name>",
		"      Copy <path> into ~/hatchery/<yyyy-mm-dd>-<name> and enter it.",
		"",
		"  hatch",
		"      Open the interactive browser with live fuzzy filtering.",
		"",
		"Actions in browser:",
		"  Enter     Open selected project or create from input",
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
		"  --usage          Show styled usage guide",
		"  --help           Show this help message",
	}
	return strings.Join(copy, "\n") + "\n"
}

func usageCard() string {
	neutralText := lipgloss.AdaptiveColor{Light: "#334155", Dark: "#E2E8F0"}
	neutralMuted := lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#A5B4CF"}
	accentLavender := lipgloss.AdaptiveColor{Light: "#9B8FC9", Dark: "#C5B7F2"}
	accentTeal := lipgloss.AdaptiveColor{Light: "#6FAFAE", Dark: "#8ED8D4"}
	accentPeach := lipgloss.AdaptiveColor{Light: "#D6A382", Dark: "#F2C6AD"}
	accentMint := lipgloss.AdaptiveColor{Light: "#72B79A", Dark: "#9FDABE"}

	title := lipgloss.NewStyle().Bold(true).Foreground(accentTeal).Render("hatch")
	subtitle := lipgloss.NewStyle().Foreground(neutralMuted).Render("Usage Guidelines")

	headline := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", subtitle)

	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(accentPeach)
	command := lipgloss.NewStyle().Bold(true).Foreground(accentLavender)
	body := lipgloss.NewStyle().Foreground(neutralText)
	note := lipgloss.NewStyle().Foreground(accentMint)

	sections := []string{
		headline,
		"",
		sectionTitle.Render("Create"),
		body.Render("  " + command.Render("hatch <name>")),
		body.Render("    Create a dated project in ~/hatchery and enter it."),
		"",
		sectionTitle.Render("Clone"),
		body.Render("  " + command.Render("hatch <git-url>")),
		body.Render("    Clone ssh/https URL into ~/hatchery/<yyyy-mm-dd>-<repo-name>."),
		"",
		sectionTitle.Render("Copy Template"),
		body.Render("  " + command.Render("hatch <path> <name>")),
		body.Render("    Copy a local directory into a dated project."),
		"",
		sectionTitle.Render("Browser"),
		body.Render("  " + command.Render("hatch")),
		body.Render("    Type to fuzzy filter, Enter to open/create."),
		body.Render("    Ctrl+A archive  •  Ctrl+R remove  •  Esc quit"),
		"",
		sectionTitle.Render("Shell Hook"),
		body.Render("  " + command.Render(`eval "$(hatch --init zsh)"`)),
		"",
		note.Render("Tip: run hatch --help for full plain-text flags."),
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentLavender).
		Padding(1, 2).
		MaxWidth(96)

	return card.Render(strings.Join(sections, "\n"))
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
