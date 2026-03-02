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
var createProjectFn = createProject
var copyProjectFn = copyProject
var worktreeProjectFn = worktreeProject

type cliOptions struct {
	cwdFile string
	init    string
	showVer bool
	showUse bool
	forceCP bool
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
			projectPath, err = createProjectFn(root, remaining[0], now())
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
		var (
			projectPath string
			action      = "Copied into: "
		)
		if options.forceCP {
			projectPath, err = copyProjectFn(root, remaining[0], remaining[1], now())
		} else {
			projectPath, err = worktreeProjectFn(root, remaining[0], remaining[1], now())
			if errors.Is(err, errNotGitRepo) {
				projectPath, err = copyProjectFn(root, remaining[0], remaining[1], now())
			} else {
				action = "Worktree created: "
			}
		}
		if err != nil {
			return err
		}
		if err := writeCWD(options.cwdFile, projectPath); err != nil {
			return err
		}
		fmt.Fprintln(out, successStyle().Render(action+projectPath))
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
	fs.BoolVar(&options.forceCP, "copy", false, "force copy behavior for <path> <name>")
	fs.BoolVar(&options.forceCP, "c", false, "shorthand for --copy")
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
		"      If <path> is a git repo, create a git worktree in ~/hatchery/<yyyy-mm-dd>-<name>.",
		"      Otherwise copy <path> into ~/hatchery/<yyyy-mm-dd>-<name>.",
		"      Use --copy or -c to always copy.",
		"",
		"  hatch",
		"      Open the interactive browser with live fuzzy filtering.",
		"",
		"Actions in browser:",
		"  Enter     Open selected project or create from input",
		"  Ctrl+R    Rename selected project",
		"  Ctrl+W    Delete selected project",
		"  Ctrl+V    Duplicate selected project (asks for new name)",
		"  Ctrl+G    Create git worktree from selected project (asks for new name)",
		"  Esc       Exit without selecting",
		"",
		"Shell integration (required for automatic cd):",
		"  eval \"$(hatch --init zsh)\"",
		"",
		"Options:",
		"  --init <shell>   Print shell hook for zsh, bash, or fish",
		"  --version        Print version",
		"  --usage          Show styled usage guide",
		"  --copy, -c       Force copy behavior for hatch <path> <name>",
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

	command := lipgloss.NewStyle().Bold(true).Foreground(accentLavender)
	body := lipgloss.NewStyle().Foreground(neutralText)
	note := lipgloss.NewStyle().Foreground(accentMint)
	spacer := lipgloss.NewStyle().Foreground(accentPeach).Render(" ")

	sections := []string{
		headline,
		"",
		spacer,
		body.Render("  " + command.Render("hatch <name>")),
		body.Render("    Create a dated project in ~/hatchery and enter it."),
		"",
		spacer,
		body.Render("  " + command.Render("hatch <git-url>")),
		body.Render("    Clone ssh/https URL into ~/hatchery/<yyyy-mm-dd>-<repo-name>."),
		"",
		spacer,
		body.Render("  " + command.Render("hatch <path> <name>")),
		body.Render("    Create a worktree when <path> is git; otherwise copy."),
		body.Render("    Add --copy or -c to force copy mode."),
		"",
		spacer,
		body.Render("  " + command.Render("hatch")),
		body.Render("    Type to fuzzy filter, Enter to open/create."),
		body.Render("    Ctrl+R rename  •  Ctrl+W delete  •  Ctrl+V duplicate"),
		body.Render("    Ctrl+G git worktree  •  Esc quit"),
		"",
		spacer,
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
