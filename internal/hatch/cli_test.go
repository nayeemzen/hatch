package hatch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCreateWritesCWDFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hatchery")
	t.Setenv("HATCHERY_HOME", root)

	cwdFile := filepath.Join(t.TempDir(), "cwd")
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	err := run([]string{"--cwd-file", cwdFile, "hatch"}, strings.NewReader(""), out, errOut, fixedNow)
	if err != nil {
		t.Fatalf("run create returned error: %v", err)
	}

	wantPath := filepath.Join(root, "2026-02-28-hatch")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected project directory: %v", err)
	}

	cwd, err := os.ReadFile(cwdFile)
	if err != nil {
		t.Fatalf("read cwd file: %v", err)
	}
	if string(cwd) != wantPath {
		t.Fatalf("cwd file = %q, want %q", string(cwd), wantPath)
	}
	if !strings.Contains(out.String(), "Created:") {
		t.Fatalf("expected success output, got %q", out.String())
	}
}

func TestRunCopy(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hatchery")
	t.Setenv("HATCHERY_HOME", root)

	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	err := run([]string{source, "copy-test"}, strings.NewReader(""), out, errOut, fixedNow)
	if err != nil {
		t.Fatalf("run copy returned error: %v", err)
	}

	copied := filepath.Join(root, "2026-02-28-copy-test", "README.md")
	if content, err := os.ReadFile(copied); err != nil {
		t.Fatalf("copied file missing: %v", err)
	} else if string(content) != "seed" {
		t.Fatalf("copied content = %q, want seed", string(content))
	}
}

func TestRunCloneFromGitURL(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hatchery")
	t.Setenv("HATCHERY_HOME", root)

	wantPath := filepath.Join(root, "2026-02-28-hatch")
	originalClone := cloneProjectFn
	cloneProjectFn = func(gotRoot, repoURL string, now time.Time) (string, error) {
		if gotRoot != root {
			t.Fatalf("clone root = %q, want %q", gotRoot, root)
		}
		if repoURL != "https://github.com/nayeemzen/hatch.git" {
			t.Fatalf("clone URL = %q", repoURL)
		}
		if now.Format("2006-01-02") != "2026-02-28" {
			t.Fatalf("unexpected now: %s", now.Format(time.RFC3339))
		}
		return wantPath, nil
	}
	t.Cleanup(func() {
		cloneProjectFn = originalClone
	})

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	err := run([]string{"https://github.com/nayeemzen/hatch.git"}, strings.NewReader(""), out, errOut, fixedNow)
	if err != nil {
		t.Fatalf("run clone returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Cloned into: "+wantPath) {
		t.Fatalf("expected clone output, got %q", out.String())
	}
}

func TestRunBrowseWritesSelection(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hatchery")
	t.Setenv("HATCHERY_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "2026-02-28-alpha"), 0o755); err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	selectedPath := filepath.Join(root, "2026-02-28-beta")
	if err := os.MkdirAll(selectedPath, 0o755); err != nil {
		t.Fatalf("create beta: %v", err)
	}

	cwdFile := filepath.Join(t.TempDir(), "cwd")
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	input := bytes.NewBufferString("beta\r")

	err := run([]string{"--cwd-file", cwdFile}, input, out, errOut, time.Now)
	if err != nil {
		t.Fatalf("run browse returned error: %v", err)
	}

	cwd, err := os.ReadFile(cwdFile)
	if err != nil {
		t.Fatalf("read cwd file: %v", err)
	}
	if string(cwd) != selectedPath {
		t.Fatalf("cwd file = %q, want %q", string(cwd), selectedPath)
	}
}

func TestHelpOutput(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	exitCode := Main([]string{"--help"}, strings.NewReader(""), out, errOut)
	if exitCode != 0 {
		t.Fatalf("help exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(out.String(), "Shell integration") {
		t.Fatalf("help output missing expected content: %q", out.String())
	}
	if !strings.Contains(out.String(), "hatch <git-url>") {
		t.Fatalf("help output missing git url usage: %q", out.String())
	}
}
