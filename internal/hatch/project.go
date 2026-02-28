package hatch

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	errInvalidName = errors.New("project name must contain at least one valid character")
)

var invalidNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Project struct {
	Name string
	Path string
}

func hatcheryRoot() (string, error) {
	if env := strings.TrimSpace(os.Getenv("HATCHERY_HOME")); env != "" {
		return expandPath(env)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "hatchery"), nil
}

func expandPath(value string) (string, error) {
	if strings.HasPrefix(value, "~/") || value == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if value == "~" {
			return home, nil
		}
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}

	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", value, err)
	}
	return abs, nil
}

func normalizeName(name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", errInvalidName
	}

	clean = strings.ReplaceAll(clean, string(os.PathSeparator), "-")
	clean = invalidNameChars.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-.")
	clean = strings.ToLower(clean)
	if clean == "" {
		return "", errInvalidName
	}

	return clean, nil
}

func projectDirName(name string, now time.Time) (string, error) {
	norm, err := normalizeName(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", now.Format("2006-01-02"), norm), nil
}

func createProject(root, name string, now time.Time) (string, error) {
	dirName, err := projectDirName(name, now)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create hatchery root: %w", err)
	}

	target := filepath.Join(root, dirName)
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("project already exists: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check project directory: %w", err)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("create project directory: %w", err)
	}

	return target, nil
}

func copyProject(root, source, name string, now time.Time) (string, error) {
	dirName, err := projectDirName(name, now)
	if err != nil {
		return "", err
	}

	resolvedSource, err := expandPath(source)
	if err != nil {
		return "", err
	}

	srcInfo, err := os.Stat(resolvedSource)
	if err != nil {
		return "", fmt.Errorf("read source directory: %w", err)
	}
	if !srcInfo.IsDir() {
		return "", fmt.Errorf("source must be a directory: %s", resolvedSource)
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create hatchery root: %w", err)
	}

	target := filepath.Join(root, dirName)
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("project already exists: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check project directory: %w", err)
	}

	if err := copyDir(resolvedSource, target); err != nil {
		return "", err
	}

	return target, nil
}

func copyDir(source, target string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("resolve relative path: %w", err)
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(target, rel)
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("read entry metadata for %s: %w", path, err)
		}

		if d.IsDir() {
			if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create directory %s: %w", dstPath, err)
			}
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", path, err)
			}
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return fmt.Errorf("create parent directory for symlink %s: %w", dstPath, err)
			}
			if err := os.Symlink(linkTarget, dstPath); err != nil {
				return fmt.Errorf("create symlink %s: %w", dstPath, err)
			}
			return nil
		}

		if err := copyFile(path, dstPath, info.Mode().Perm()); err != nil {
			return err
		}
		return nil
	})
}

func copyFile(source, target string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent directory for file %s: %w", target, err)
	}

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", source, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", target, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy file %s -> %s: %w", source, target, err)
	}
	return nil
}

func listProjects(root string) ([]Project, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create hatchery root: %w", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read hatchery root: %w", err)
	}

	projects := make([]Project, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "archive" {
			continue
		}
		projects = append(projects, Project{
			Name: entry.Name(),
			Path: filepath.Join(root, entry.Name()),
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name > projects[j].Name
	})

	return projects, nil
}

func archiveProject(root, projectPath string) (string, error) {
	archiveRoot := filepath.Join(root, "archive")
	if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
		return "", fmt.Errorf("create archive directory: %w", err)
	}

	target := filepath.Join(archiveRoot, filepath.Base(projectPath))
	target = nextAvailablePath(target)

	if err := os.Rename(projectPath, target); err != nil {
		return "", fmt.Errorf("archive project: %w", err)
	}

	return target, nil
}

func removeProject(projectPath string) error {
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("remove project: %w", err)
	}
	return nil
}

func nextAvailablePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", path, i)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}
