package main

import (
	"os"
	"path/filepath"
	"sort"
)

// resolveAbsPath turns any path into a cleaned absolute path.
func resolveAbsPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// listDirs returns all immediate subdirectory names under dir,
// sorted alphabetically. Hidden dirs (starting with .) are included.
func listDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
		// Also follow symlinks that point to directories
		if e.Type()&os.ModeSymlink != 0 {
			target := filepath.Join(dir, e.Name())
			info, err := os.Stat(target)
			if err == nil && info.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}
	}

	sort.Strings(dirs)
	return dirs, nil
}

// parentDir returns the parent of a path. Returns the same path if already root.
func parentDir(p string) string {
	parent := filepath.Dir(p)
	if parent == p {
		return p // already at filesystem root
	}
	return parent
}
