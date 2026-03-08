package watcher

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/HITENDRAS940/hotreload/internal/ui"
)

const ignoreFileName = ".hotreloadignore"

const defaultIgnoreContent = `# Hotreload Ignore File
# Files and folders listed here will be excluded from file watching.
# One pattern per line. Lines starting with # are comments.
#
# Examples:
#   .cache          -> ignores any folder/file named .cache
#   *.log           -> ignores all .log files
#   tmp             -> ignores any folder/file named tmp
#   .env.local      -> ignores a specific file

# Version control
.git

# Dependencies
node_modules
vendor

# Build outputs
dist
build
bin

# IDE / Editor
.vscode
.idea

# Temporary / cache files
.cache
tmp
*.tmp
*.log
*.swp
*.swo

# OS files
.DS_Store
thumbs.db

# Go module checksum (changes frequently, not source)
go.sum
`

// LoadIgnorePatterns reads .hotreloadignore from root, prompts if missing,
// and returns the list of active ignore patterns.
func LoadIgnorePatterns(root string) []string {
	patterns, found := readIgnoreFile(root)
	if found {
		return patterns
	}

	// File not found — warn and prompt
	ui.Warn(".hotreloadignore not found in: " + root)
	ui.Warn("Without it, ALL file changes will trigger rebuilds.")

	response := ui.Prompt("Create .hotreloadignore with sensible defaults?")

	if response == "n" || response == "no" {
		if err := createIgnoreFile(root); err != nil {
			ui.Fatal("Could not create .hotreloadignore: " + err.Error())
		}
		ui.Success("Created .hotreloadignore in " + root)
		ui.Info("Using default ignore patterns — edit the file to customize.")

		// Load and return the newly created file's patterns
		patterns, _ = readIgnoreFile(root)
		return patterns
	}

	ui.Info("Continuing without .hotreloadignore — all file changes will be tracked.")
	return nil
}

func readIgnoreFile(root string) ([]string, bool) {
	path := filepath.Join(root, ignoreFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, true
}

func createIgnoreFile(root string) error {
	path := filepath.Join(root, ignoreFileName)
	return os.WriteFile(path, []byte(defaultIgnoreContent), 0644)
}
