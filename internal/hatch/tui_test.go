package hatch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFuzzyScore(t *testing.T) {
	t.Parallel()

	contiguous := fuzzyScore("2026-02-28-hatch", "hat")
	scattered := fuzzyScore("2026-02-28-hardhat", "hat")
	if contiguous <= scattered {
		t.Fatalf("expected contiguous match score (%d) to be greater than scattered (%d)", contiguous, scattered)
	}

	if got := fuzzyScore("hatch", "zzz"); got != noMatchScore {
		t.Fatalf("expected missing query score %d, got %d", noMatchScore, got)
	}

	if got := fuzzyScore("2026-03-01-spike-auth", "spike auth"); got < 0 {
		t.Fatalf("expected space-separated query to match, got %d", got)
	}
}

func TestBrowserViewContainsPolishedCopy(t *testing.T) {
	t.Parallel()

	projects := []Project{{Name: "2026-02-28-hatch", Path: "/tmp/2026-02-28-hatch"}}
	model := newBrowserModel("/tmp", projects)
	model.width = 110
	model.height = 28
	model.query = "hat"
	model.refreshFilter()

	view := model.View()
	mustContain := []string{"hatch", "Project hatchery", "Ctrl+A archive", "Ctrl+R remove", "Filter"}
	for _, snippet := range mustContain {
		if !strings.Contains(view, snippet) {
			t.Fatalf("view should contain %q, got:\n%s", snippet, view)
		}
	}
}

func TestBrowserFilterIncludesNonPrefixMatch(t *testing.T) {
	t.Parallel()

	project := Project{
		Name: "2026-03-01-alpha-super-long-project-name",
		Path: "/tmp/2026-03-01-alpha-super-long-project-name",
	}
	model := newBrowserModel("/tmp", []Project{project})
	model.query = "project"
	model.refreshFilter()

	if len(model.filtered) != 1 {
		t.Fatalf("expected non-prefix fuzzy match to remain visible, filtered=%v", model.filtered)
	}
}

func TestBrowserArchiveAction(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	projectPath := filepath.Join(root, "2026-02-28-hatch")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}

	model := newBrowserModel(root, []Project{{Name: "2026-02-28-hatch", Path: projectPath}})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	model = updated.(browserModel)
	if model.confirm != confirmArchive {
		t.Fatalf("expected confirmArchive state, got %v", model.confirm)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(browserModel)

	if model.confirm != confirmNone {
		t.Fatalf("expected confirmNone after archive, got %v", model.confirm)
	}
	if _, err := os.Stat(filepath.Join(root, "archive", "2026-02-28-hatch")); err != nil {
		t.Fatalf("expected archived directory: %v", err)
	}
	if len(model.projects) != 0 {
		t.Fatalf("expected project list to refresh and become empty, got %d", len(model.projects))
	}
}

func TestRunBrowserSelectsProject(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	alpha := filepath.Join(root, "2026-02-28-alpha")
	beta := filepath.Join(root, "2026-02-28-beta")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatalf("create beta: %v", err)
	}

	input := bytes.NewBufferString("beta\r")
	output := new(bytes.Buffer)

	selected, err := runBrowser(root, input, output)
	if err != nil {
		t.Fatalf("runBrowser returned error: %v", err)
	}
	if selected != beta {
		t.Fatalf("selected path = %q, want %q", selected, beta)
	}
	if !strings.Contains(output.String(), "\x1b[?25l") {
		t.Fatalf("expected terminal control output, got %q", output.String())
	}
}

func TestBrowserSpaceKeyAppendsToFilter(t *testing.T) {
	t.Parallel()

	model := newBrowserModel("/tmp", nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")})
	model = updated.(browserModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(browserModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("world")})
	model = updated.(browserModel)

	if model.query != "hello world" {
		t.Fatalf("expected query with space, got %q", model.query)
	}
}

func TestBrowserCreateNewOptionAndSelect(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	fixedNow := func() time.Time {
		return time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	}

	model := newBrowserModelWithClock(root, nil, fixedNow)
	model.query = "new project"
	model.refreshFilter()

	view := model.View()
	if !strings.Contains(view, "Create New: new project") {
		t.Fatalf("expected create option in view, got:\n%s", view)
	}
	if !model.isCreateRow(model.cursor) {
		t.Fatalf("expected create option to be selected when no projects, cursor=%d", model.cursor)
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(browserModel)

	want := filepath.Join(root, "2026-03-01-new-project")
	if model.selectedPath != want {
		t.Fatalf("selected path = %q, want %q", model.selectedPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected created project at %q: %v", want, err)
	}
}

func TestBrowserEnterOpensMatchBeforeCreateOption(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	beta := filepath.Join(root, "2026-02-28-beta")
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatalf("create beta: %v", err)
	}

	model := newBrowserModel(root, []Project{{Name: "2026-02-28-beta", Path: beta}})
	model.query = "beta"
	model.refreshFilter()
	if model.cursor != 0 {
		t.Fatalf("expected cursor to remain on first project, got %d", model.cursor)
	}
	if model.isCreateRow(model.cursor) {
		t.Fatalf("expected first row to be project, got create row")
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(browserModel)
	if model.selectedPath != beta {
		t.Fatalf("selected path = %q, want %q", model.selectedPath, beta)
	}
}
