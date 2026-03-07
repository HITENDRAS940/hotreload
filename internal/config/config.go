package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the validated CLI arguments for hotreload.
type Config struct {
	Root     string // Absolute path to the project root directory
	BuildCmd string // Shell command to build the project
	ExecCmd  string // Shell command to run the built server
}

// Parse reads CLI flags, validates them, and returns a Config.
// It exits with code 1 and prints usage on invalid input.
func Parse() Config {
	var cfg Config

	flag.StringVar(&cfg.Root, "root", ".", "root directory of the project to watch")
	flag.StringVar(&cfg.BuildCmd, "build", "", "build command (e.g. \"go build -o ./bin/server ./cmd/server\")")
	flag.StringVar(&cfg.ExecCmd, "exec", "", "exec command to run after build (e.g. \"./bin/server\")")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hotreload [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  hotreload --root ./myproject --build \"go build -o ./bin/server ./cmd/server\" --exec \"./bin/server\"\n")
	}

	flag.Parse()

	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func (c *Config) validate() error {
	if c.BuildCmd == "" {
		return fmt.Errorf("--build flag is required")
	}
	if c.ExecCmd == "" {
		return fmt.Errorf("--exec flag is required")
	}

	abs, err := filepath.Abs(c.Root)
	if err != nil {
		return fmt.Errorf("cannot resolve root path %q: %w", c.Root, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("root directory does not exist: %s", abs)
		}
		return fmt.Errorf("cannot access root directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", abs)
	}

	c.Root = abs
	return nil
}
