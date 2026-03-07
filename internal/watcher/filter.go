package watcher

import (
	"path/filepath"
	"strings"
)

func shouldIgnore(path string) bool {
	path = filepath.ToSlash(path)

	ignorePatterns := []string{
		".git",
		".gitignore",
		"node_modules",
		".tmp",
		".swp",
		".swo",
		"~",
		"*.tmp",
		".DS_Store",
		"thumbs.db",
		"go.sum",
		"vendor",
		"dist",
		"build",
		"bin",
		".vscode",
		".idea",
	}

	for _, pattern := range ignorePatterns {
		if strings.Contains(path, "/"+pattern+"/") || strings.HasPrefix(path, pattern+"/") {
			return true
		}
		if strings.HasSuffix(path, "/"+pattern) || path == pattern {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasSuffix(path, pattern[1:]) {
			return true
		}
	}

	return false
}

func isWatchable(path string) bool {
	if strings.HasSuffix(path, ".go") {
		return true
	}

	if !strings.Contains(filepath.Base(path), ".") {
		return true
	}

	return false
}
