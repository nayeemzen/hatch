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
}
