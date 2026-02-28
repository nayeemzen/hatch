package hatch

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var errNoSelection = errors.New("no project selected")

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmArchive
	confirmRemove
)

type scoredIndex struct {
	index int
	score int
}

type browserStyles struct {
	app           lipgloss.Style
	title         lipgloss.Style
	subtitle      lipgloss.Style
	searchLabel   lipgloss.Style
	searchPrompt  lipgloss.Style
	query         lipgloss.Style
	placeholder   lipgloss.Style
	project       lipgloss.Style
	projectActive lipgloss.Style
	empty         lipgloss.Style
	detail        lipgloss.Style
	help          lipgloss.Style
	status        lipgloss.Style
	confirm       lipgloss.Style
}

func defaultBrowserStyles() browserStyles {
	return browserStyles{
		app: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0B7285")),
		title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B7285")),
		subtitle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6770")),
		searchLabel:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E67700")),
		searchPrompt:  lipgloss.NewStyle().Foreground(lipgloss.Color("#0B7285")).Bold(true),
		query:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1F2933")),
		placeholder:   lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8")),
		project:       lipgloss.NewStyle().Foreground(lipgloss.Color("#1F2933")),
		projectActive: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F9FA")).Background(lipgloss.Color("#0B7285")).Padding(0, 1),
		empty:         lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")),
		detail:        lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6770")),
		help:          lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6770")),
		status:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0F766E")),
		confirm: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#E67700")).
			Padding(0, 1).
			Foreground(lipgloss.Color("#7C2D12")),
	}
}

type browserModel struct {
	root         string
	projects     []Project
	filtered     []int
	cursor       int
	query        string
	width        int
	height       int
	status       string
	confirm      confirmAction
	selectedPath string
	err          error
	quitting     bool
	styles       browserStyles
}

func newBrowserModel(root string, projects []Project) browserModel {
	m := browserModel{
		root:     root,
		projects: projects,
		width:    100,
		height:   28,
		styles:   defaultBrowserStyles(),
		status:   "Use arrows to move, Enter to open",
	}
	m.refreshFilter()
	return m
}

func (m browserModel) Init() tea.Cmd {
	return nil
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.confirm != confirmNone {
			return m.updateConfirm(msg)
		}
		return m.updateMain(msg)
	default:
		return m, nil
	}
}

func (m browserModel) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil
	case tea.KeyEnter:
		selected := m.currentProject()
		if selected == nil {
			m.status = "No matching project"
			return m, nil
		}
		m.selectedPath = selected.Path
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlA:
		if m.currentProject() != nil {
			m.confirm = confirmArchive
		}
		return m, nil
	case tea.KeyCtrlR:
		if m.currentProject() != nil {
			m.confirm = confirmRemove
		}
		return m, nil
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.query) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.query)
			m.query = m.query[:len(m.query)-size]
			m.refreshFilter()
		}
		return m, nil
	case tea.KeyRunes:
		m.query += string(msg.Runes)
		m.refreshFilter()
		return m, nil
	default:
		switch msg.String() {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
		return m, nil
	}
}

func (m browserModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.confirm = confirmNone
		m.status = "Action cancelled"
		return m, nil
	case tea.KeyEnter:
		return m.applyConfirm()
	case tea.KeyRunes:
		switch strings.ToLower(string(msg.Runes)) {
		case "y":
			return m.applyConfirm()
		case "n":
			m.confirm = confirmNone
			m.status = "Action cancelled"
			return m, nil
		}
	}
	return m, nil
}

func (m browserModel) applyConfirm() (tea.Model, tea.Cmd) {
	selected := m.currentProject()
	if selected == nil {
		m.confirm = confirmNone
		m.status = "No matching project"
		return m, nil
	}

	switch m.confirm {
	case confirmArchive:
		target, err := archiveProject(m.root, selected.Path)
		if err != nil {
			m.err = err
			m.quitting = true
			return m, tea.Quit
		}
		m.status = fmt.Sprintf("Archived %s -> %s", selected.Name, filepath.Base(target))
	case confirmRemove:
		if err := removeProject(selected.Path); err != nil {
			m.err = err
			m.quitting = true
			return m, tea.Quit
		}
		m.status = fmt.Sprintf("Removed %s", selected.Name)
	}

	m.confirm = confirmNone
	projects, err := listProjects(m.root)
	if err != nil {
		m.err = err
		m.quitting = true
		return m, tea.Quit
	}
	m.projects = projects
	m.refreshFilter()
	return m, nil
}

func (m *browserModel) refreshFilter() {
	query := strings.TrimSpace(strings.ToLower(m.query))
	scored := make([]scoredIndex, 0, len(m.projects))
	for i, project := range m.projects {
		score := fuzzyScore(project.Name, query)
		if score < 0 {
			continue
		}
		scored = append(scored, scoredIndex{index: i, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return m.projects[scored[i].index].Name > m.projects[scored[j].index].Name
		}
		return scored[i].score > scored[j].score
	})

	m.filtered = m.filtered[:0]
	for _, item := range scored {
		m.filtered = append(m.filtered, item.index)
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m browserModel) currentProject() *Project {
	if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	item := m.projects[m.filtered[m.cursor]]
	return &item
}

func (m browserModel) View() string {
	if m.quitting {
		return ""
	}

	title := m.styles.title.Render("hatch")
	subtitle := m.styles.subtitle.Render("Project hatchery")
	searchLabel := m.styles.searchLabel.Render("Filter")

	query := m.styles.placeholder.Render("type to search")
	if m.query != "" {
		query = m.styles.query.Render(m.query)
	}
	searchLine := lipgloss.JoinHorizontal(lipgloss.Left, m.styles.searchPrompt.Render("› "), query)

	rows := m.renderRows()
	selectedInfo := ""
	if selected := m.currentProject(); selected != nil {
		selectedInfo = m.styles.detail.Render(selected.Path)
	}

	help := m.styles.help.Render("↑/↓ move  •  type to filter  •  Enter open  •  Ctrl+A archive  •  Ctrl+R remove  •  Esc quit")
	status := m.styles.status.Render(m.status)

	body := []string{
		lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", subtitle),
		"",
		searchLabel,
		searchLine,
		"",
		rows,
	}

	if selectedInfo != "" {
		body = append(body, "", selectedInfo)
	}

	body = append(body, "", status, help)

	if m.confirm != confirmNone {
		body = append(body, "", m.confirmPrompt())
	}

	content := strings.Join(body, "\n")
	if m.width > 0 {
		return m.styles.app.Width(max(80, min(m.width-2, 120))).Render(content)
	}
	return m.styles.app.Render(content)
}

func (m browserModel) renderRows() string {
	if len(m.filtered) == 0 {
		if strings.TrimSpace(m.query) == "" {
			return m.styles.empty.Render("No projects yet. Run: hatch <name>")
		}
		return m.styles.empty.Render("No matches")
	}

	maxRows := max(8, m.height-14)
	if maxRows > len(m.filtered) {
		maxRows = len(m.filtered)
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := min(start+maxRows, len(m.filtered))

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		project := m.projects[m.filtered[i]]
		line := fmt.Sprintf("  %s", project.Name)
		if i == m.cursor {
			line = m.styles.projectActive.Render("▸ " + project.Name)
		} else {
			line = m.styles.project.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m browserModel) confirmPrompt() string {
	selected := m.currentProject()
	if selected == nil {
		return ""
	}
	action := "Archive"
	if m.confirm == confirmRemove {
		action = "Remove"
	}
	copy := fmt.Sprintf("%s %s?  [y/Enter] confirm  [n/Esc] cancel", action, selected.Name)
	return m.styles.confirm.Render(copy)
}

func runBrowser(root string, in io.Reader, out io.Writer) (string, error) {
	projects, err := listProjects(root)
	if err != nil {
		return "", err
	}

	model := newBrowserModel(root, projects)
	program := tea.NewProgram(model, tea.WithInput(in), tea.WithOutput(out))
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("run browser: %w", err)
	}

	result, ok := finalModel.(browserModel)
	if !ok {
		return "", errors.New("unexpected browser model type")
	}
	if result.err != nil {
		return "", result.err
	}
	if result.selectedPath == "" {
		return "", errNoSelection
	}
	return result.selectedPath, nil
}

func fuzzyScore(candidate, query string) int {
	if query == "" {
		return 0
	}

	candidateRunes := []rune(strings.ToLower(candidate))
	queryRunes := []rune(strings.ToLower(query))
	if len(queryRunes) > len(candidateRunes) {
		return -1
	}

	score := 0
	lastMatch := -2
	cursor := 0
	for _, queryRune := range queryRunes {
		found := false
		for cursor < len(candidateRunes) {
			if candidateRunes[cursor] == queryRune {
				score += 10
				if cursor == lastMatch+1 {
					score += 8
				}
				if cursor == 0 || isWordBoundary(candidateRunes[cursor-1]) {
					score += 4
				}
				score -= cursor
				lastMatch = cursor
				cursor++
				found = true
				break
			}
			cursor++
		}
		if !found {
			return -1
		}
	}

	score -= len(candidateRunes) / 4
	return score
}

func isWordBoundary(r rune) bool {
	switch r {
	case '-', '_', '.', '/', ' ':
		return true
	default:
		return false
	}
}
