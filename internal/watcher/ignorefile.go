package watcher

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	fmt.Fprintf(os.Stderr, "\n[WARNING] .hotreloadignore not found in: %s\n", root)
	fmt.Fprintf(os.Stderr, "Without it, ALL file changes in the project will trigger rebuilds.\n\n")
	fmt.Fprintf(os.Stderr, "Would you like to continue without .hotreloadignore? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "n" || response == "no" {
		if err := createIgnoreFile(root); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Could not create .hotreloadignore: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\n[OK] Created .hotreloadignore in %s\n", root)
		fmt.Fprintf(os.Stderr, "Using default ignore patterns. Edit .hotreloadignore to customize.\n\n")

		// Load and return the newly created file's patterns
		patterns, _ = readIgnoreFile(root)
		return patterns
	}

	fmt.Fprintf(os.Stderr, "Continuing without .hotreloadignore — all file changes will be tracked.\n\n")
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
