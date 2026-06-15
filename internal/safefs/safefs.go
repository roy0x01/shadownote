package safefs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResetDir(path string, forbidden ...string) error {
	abs, err := ValidateManagedDir(path, forbidden...)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("clear %s: %w", abs, err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return fmt.Errorf("create %s: %w", abs, err)
	}
	return nil
}

func RemoveManagedDir(path, parent string, forbidden ...string) error {
	abs, err := ValidateManagedDir(path, forbidden...)
	if err != nil {
		return err
	}
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return fmt.Errorf("resolve parent: %w", err)
	}
	parentAbs = filepath.Clean(parentAbs)
	if !isWithin(abs, parentAbs) || abs == parentAbs {
		return fmt.Errorf("refusing to remove %s: not a child of %s", abs, parentAbs)
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("remove %s: %w", abs, err)
	}
	return nil
}

func ValidateManagedDir(path string, forbidden ...string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("refusing destructive operation on empty path")
	}
	cleanInput := filepath.Clean(path)
	if cleanInput == "." || cleanInput == ".." || cleanInput == string(filepath.Separator) {
		return "", fmt.Errorf("refusing destructive operation on unsafe path %q", path)
	}
	if strings.Contains(cleanInput, string(filepath.Separator)+".."+string(filepath.Separator)) || strings.HasSuffix(cleanInput, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("refusing destructive operation on path containing '..': %q", path)
	}

	abs, err := filepath.Abs(cleanInput)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	abs = filepath.Clean(abs)

	if isFilesystemRoot(abs) {
		return "", fmt.Errorf("refusing destructive operation on filesystem root %s", abs)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	cwd = filepath.Clean(cwd)

	if abs == cwd {
		return "", fmt.Errorf("refusing destructive operation on protected path %s", abs)
	}

	protectedSubtrees := []string{
		filepath.Join(cwd, ".git"),
		filepath.Join(cwd, "content"),
		filepath.Join(cwd, "internal"),
		filepath.Join(cwd, "assets"),
		filepath.Join(cwd, "themes"),
		filepath.Join(cwd, "templates"),
	}
	if home, herr := os.UserHomeDir(); herr == nil && strings.TrimSpace(home) != "" {
		dataRoot := filepath.Join(home, ".shadownote")
		protectedSubtrees = append(protectedSubtrees,
			filepath.Join(dataRoot, "assets"),
			filepath.Join(dataRoot, "content"),
			filepath.Join(dataRoot, "logs"),
			filepath.Join(dataRoot, "templates"),
			filepath.Join(dataRoot, "themes"),
			filepath.Join(dataRoot, "shadownote.db"),
		)
		homeAbs := filepath.Clean(home)
		dataAbs := filepath.Clean(dataRoot)
		if abs == homeAbs {
			return "", fmt.Errorf("refusing destructive operation on home directory %s", abs)
		}
		if abs == dataAbs {
			return "", fmt.Errorf("refusing destructive operation on data directory root %s", abs)
		}
	}
	protectedSubtrees = append(protectedSubtrees, forbidden...)
	for _, p := range protectedSubtrees {
		if strings.TrimSpace(p) == "" {
			continue
		}
		forbidAbs, err := filepath.Abs(filepath.Clean(p))
		if err != nil {
			return "", fmt.Errorf("resolve forbidden path %q: %w", p, err)
		}
		forbidAbs = filepath.Clean(forbidAbs)
		if abs == forbidAbs || isWithin(abs, forbidAbs) {
			return "", fmt.Errorf("refusing destructive operation on protected path %s", abs)
		}
	}
	return abs, nil
}

func isFilesystemRoot(abs string) bool {
	parent := filepath.Dir(abs)
	return parent == abs
}

func isWithin(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
