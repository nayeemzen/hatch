package hatch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, time.February, 28, 10, 0, 0, 0, time.UTC)
}

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "hatch", want: "hatch"},
		{input: "  Project Name  ", want: "project-name"},
		{input: "foo/bar", want: "foo-bar"},
		{input: "___", want: "___"},
		{input: "   ", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeName returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCreateProject(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	path, err := createProject(root, "Hatch", fixedNow())
	if err != nil {
		t.Fatalf("createProject returned error: %v", err)
	}

	if got, want := filepath.Base(path), "2026-02-28-hatch"; got != want {
		t.Fatalf("project directory name = %q, want %q", got, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("project directory should exist: %v", err)
	}

	if _, err := createProject(root, "Hatch", fixedNow()); err == nil {
		t.Fatalf("expected duplicate project creation to fail")
	}
}

func TestCopyProject(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	target, err := copyProject(root, source, "Replica", fixedNow())
	if err != nil {
		t.Fatalf("copyProject returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "nested", "hello.txt"))
	if err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("copied file content = %q, want hello", string(content))
	}
}

func TestListProjectsExcludesArchive(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	dirs := []string{
		"2026-02-27-older",
		"2026-02-28-newer",
		"archive",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("create dir %s: %v", dir, err)
		}
	}

	projects, err := listProjects(root)
	if err != nil {
		t.Fatalf("listProjects returned error: %v", err)
	}

	got := make([]string, 0, len(projects))
	for _, project := range projects {
		got = append(got, project.Name)
	}
	want := []string{"2026-02-28-newer", "2026-02-27-older"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("project order = %#v, want %#v", got, want)
	}
}

func TestArchiveProjectCollision(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "hatchery")
	projectPath := filepath.Join(root, "2026-02-28-hatch")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "archive", "2026-02-28-hatch"), 0o755); err != nil {
		t.Fatalf("create existing archive project: %v", err)
	}

	archivedPath, err := archiveProject(root, projectPath)
	if err != nil {
		t.Fatalf("archiveProject returned error: %v", err)
	}

	if got, want := filepath.Base(archivedPath), "2026-02-28-hatch-2"; got != want {
		t.Fatalf("archive path = %q, want %q", got, want)
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived directory should exist: %v", err)
	}
}
