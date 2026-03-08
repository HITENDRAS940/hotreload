package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	cyanBold   = color.New(color.FgCyan, color.Bold)
	greenBold  = color.New(color.FgGreen, color.Bold)
	yellowBold = color.New(color.FgYellow, color.Bold)
	redBold    = color.New(color.FgRed, color.Bold)
	whiteBold  = color.New(color.FgWhite, color.Bold)
	dim        = color.New(color.FgHiBlack)
	magenta    = color.New(color.FgMagenta, color.Bold)
	hiWhite    = color.New(color.FgHiWhite)
)

// Banner prints the startup banner to stderr.
func Banner() {
	w := os.Stderr
	fmt.Fprintln(w)
	cyanBold.Fprintln(w, "  \u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2557")
	fmt.Fprint(w, "  \u2551  ")
	magenta.Fprint(w, "\u26a1 hotreload")
	dim.Fprint(w, "  \u00b7  live reload for Go")
	cyanBold.Fprintln(w, "        \u2551")
	cyanBold.Fprintln(w, "  \u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u255d")
	fmt.Fprintln(w)
}

// Config prints a labelled key-value config line.
func Config(key, value string) {
	fmt.Fprintf(os.Stderr, "  %s  \u2192  %s\n",
		whiteBold.Sprintf("%-8s", key),
		cyanBold.Sprint(value),
	)
}

// Success prints a green checkmark line to stderr.
func Success(msg string) {
	greenBold.Fprint(os.Stderr, "  \u2714  ")
	hiWhite.Fprintln(os.Stderr, msg)
}

// Info prints a cyan info line to stderr.
func Info(msg string) {
	cyanBold.Fprint(os.Stderr, "  \u25c6  ")
	hiWhite.Fprintln(os.Stderr, msg)
}

// Warn prints a yellow warning line to stderr.
func Warn(msg string) {
	yellowBold.Fprint(os.Stderr, "  \u26a0  ")
	yellowBold.Fprintln(os.Stderr, msg)
}

// Error prints a red error line to stderr.
func Error(msg string) {
	redBold.Fprint(os.Stderr, "  \u2717  ")
	redBold.Fprintln(os.Stderr, msg)
}

// Fatal prints a red error then exits 1.
func Fatal(msg string) {
	Error(msg)
	os.Exit(1)
}

// Exclude prints a dimmed exclusion line.
func Exclude(pattern string) {
	dim.Fprint(os.Stderr, "     \u21b3  ")
	dim.Fprintln(os.Stderr, pattern)
}

// Step prints a build/runtime event line with a magenta bullet.
func Step(msg string) {
	magenta.Fprint(os.Stderr, "  \u25cf  ")
	hiWhite.Fprintln(os.Stderr, msg)
}

// Done prints a green checkmark with message and optional detail.
func Done(msg, detail string) {
	greenBold.Fprint(os.Stderr, "  \u2714  ")
	hiWhite.Fprint(os.Stderr, msg)
	if detail != "" {
		dim.Fprint(os.Stderr, "  (")
		dim.Fprint(os.Stderr, detail)
		dim.Fprint(os.Stderr, ")")
	}
	fmt.Fprintln(os.Stderr)
}

// Fail prints a red cross with message and optional detail.
func Fail(msg, detail string) {
	redBold.Fprint(os.Stderr, "  \u2717  ")
	redBold.Fprint(os.Stderr, msg)
	if detail != "" {
		dim.Fprint(os.Stderr, "  (")
		dim.Fprint(os.Stderr, detail)
		dim.Fprint(os.Stderr, ")")
	}
	fmt.Fprintln(os.Stderr)
}

// Prompt prints a styled prompt and reads a single line from stdin.
func Prompt(msg string) string {
	cyanBold.Fprint(os.Stderr, "  ?  ")
	hiWhite.Fprint(os.Stderr, msg)
	dim.Fprint(os.Stderr, " [y/n]: ")

	var input string
	fmt.Fscan(os.Stdin, &input)
	return strings.TrimSpace(strings.ToLower(input))
}

// Separator prints a dim horizontal rule.
func Separator() {
	dim.Fprintln(os.Stderr, "  "+strings.Repeat("\u2500", 44))
}

// Watching prints the watching-N-directories line.
func Watching(n int) {
	greenBold.Fprint(os.Stderr, "  \u2714  ")
	if n == 1 {
		hiWhite.Fprintln(os.Stderr, "watching 1 directory")
	} else {
		hiWhite.Fprintf(os.Stderr, "watching %d directories\n", n)
	}
}
