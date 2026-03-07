package watcher

import (
	"path/filepath"
	"strings"
)

// shouldIgnore returns true if the path should be ignored by the watcher.
// It filters out build artifacts, version control, editor temp files, and common dependencies.
func shouldIgnore(path string) bool {
	// Normalize path to forward slashes for consistent matching
	path = filepath.ToSlash(path)

	// List of patterns to ignore
	ignorePatterns := []string{
		".git",
		".gitignore",
		"node_modules",
		".tmp",
		".swp",
		".swo",
		"~",
		"*.tmp",
		".DS_Store", // macOS
		"thumbs.db", // Windows
		"go.sum",    // Often changes without rebuild need
		"vendor",    // Vendored deps
		"dist",      // Build output
		"build",     // Build output
		"bin",       // Build output
		".vscode",   // Editor config
		".idea",     // IDE config
	}

	// Check each pattern
	for _, pattern := range ignorePatterns {
		// Check if pattern is a full path component (e.g., ".git/")
		if strings.Contains(path, "/"+pattern+"/") || strings.HasPrefix(path, pattern+"/") {
			return true
		}
		// Check if it matches end-of-path for directories
		if strings.HasSuffix(path, "/"+pattern) || path == pattern {
			return true
		}
		// Check file extensions
		if strings.HasSuffix(pattern, "*") && strings.HasSuffix(path, pattern[1:]) {
			return true
		}
	}

	return false
}

// isWatchable checks if a file should trigger a rebuild.
// Only .go files and directory-level events are of interest.
func isWatchable(path string) bool {
	// Watch .go files
	if strings.HasSuffix(path, ".go") {
		return true
	}

	// Allow directory events (no extension means it's a directory)
	// Directories have IsDir=true in fsnotify, but we check by suffix here
	if !strings.Contains(filepath.Base(path), ".") {
		return true
	}

	return false
}
