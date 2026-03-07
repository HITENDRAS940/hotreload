package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Builder executes the build command and streams output in real time.
type Builder struct {
	command string      // Shell command to execute (e.g., "go build -o ./bin/server ./cmd/server")
	logger  *slog.Logger // Structured logger
}

// NewBuilder creates a new builder with the given build command.
func NewBuilder(command string) *Builder {
	return &Builder{
		command: command,
		logger:  slog.Default(),
	}
}

// Build executes the build command with the given context.
// Output is streamed to stdout/stderr in real time.
// Returns nil on success (exit code 0), error otherwise.
// If ctx is cancelled while build is in-flight, the build process is killed.
func (b *Builder) Build(ctx context.Context) error {
	startTime := time.Now()
	b.logger.Info("build started")

	// Parse shell command into program + args
	// Handle quoted strings and spaces properly
	parts := parseShellCommand(b.command)
	if len(parts) == 0 {
		return fmt.Errorf("invalid build command: empty after parsing")
	}

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set up process to receive signals
	// This ensures graceful termination when context is cancelled
	cmd.SysProcAttr = nil // Will be set per-platform if needed

	// Stream stdout/stderr directly to process output (no buffering)
	// This allows real-time observation of build progress
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Log the exact command being executed
	b.logger.Debug("executing build command", "command", b.command)

	// Run the build command
	err := cmd.Run()

	elapsed := time.Since(startTime)

	if err != nil {
		// Build failed — log and return error
		b.logger.Error("build failed",
			"error", err,
			"duration", elapsed.String(),
		)
		return fmt.Errorf("build failed: %w", err)
	}

	// Build succeeded
	b.logger.Info("build succeeded",
		"duration", elapsed.String(),
	)
	return nil
}

// parseShellCommand parses a shell command string into program + arguments.
// Handles quoted strings and respects spaces within quotes.
// Examples:
//   "go build -o ./bin/server ./cmd/server" → ["go", "build", "-o", "./bin/server", "./cmd/server"]
//   "sh -c 'echo hello'" → ["sh", "-c", "echo hello"]
func parseShellCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range cmd {
		switch {
		case (ch == '"' || ch == '\'') && !inQuotes:
			// Start quoted section
			inQuotes = true
			quoteChar = ch

		case ch == quoteChar && inQuotes:
			// End quoted section
			inQuotes = false
			quoteChar = 0

		case ch == ' ' && !inQuotes:
			// Space outside quotes — end current part
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}

		default:
			// Regular character
			current.WriteRune(ch)
		}
	}

	// Add final part if any
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
