package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/HITENDRAS940/hotreload/internal/ui"
)

type Builder struct {
	command string
}

func NewBuilder(command string) *Builder {
	return &Builder{command: command}
}

func (b *Builder) Build(ctx context.Context) error {
	startTime := time.Now()
	ui.Step("build started")

	parts := parseShellCommand(b.command)
	if len(parts) == 0 {
		return fmt.Errorf("invalid build command: empty after parsing")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	cmd.SysProcAttr = nil

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	elapsed := time.Since(startTime)

	if err != nil {
		ui.Fail("build failed", elapsed.Round(time.Millisecond).String())
		return fmt.Errorf("build failed: %w", err)
	}

	ui.Done("build complete", elapsed.Round(time.Millisecond).String())
	return nil
}

func parseShellCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range cmd {
		switch {
		case (ch == '"' || ch == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = ch

		case ch == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0

		case ch == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
