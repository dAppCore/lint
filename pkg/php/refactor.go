package php

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	core "dappco.re/go"
)

// RectorOptions configures Rector code refactoring.
type RectorOptions struct {
	Dir        string
	Fix        bool // Apply changes (default is dry-run)
	Diff       bool // Show detailed diff
	ClearCache bool // Clear cache before running
	Output     io.Writer
}

// DetectRector checks if Rector is available in the project.
func DetectRector(dir string) bool {
	// Check for rector.php config
	rectorConfig := filepath.Join(dir, "rector.php")
	if fileExists(rectorConfig) {
		return true
	}

	// Check for vendor binary
	rectorBin := filepath.Join(dir, "vendor", "bin", "rector")
	if fileExists(rectorBin) {
		return true
	}

	return false
}

// RunRector runs Rector for automated code refactoring.
func RunRector(ctx context.Context, opts RectorOptions) error {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return core.E("php.RunRector", "get working directory", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Build command
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "rector")
	cmdName := "rector"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"process"}

	if !opts.Fix {
		args = append(args, "--dry-run")
	}

	if opts.Diff {
		args = append(args, "--output-format", "diff")
	}

	if opts.ClearCache {
		args = append(args, "--clear-cache")
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = opts.Dir
	cmd.Stdout = opts.Output
	cmd.Stderr = opts.Output

	return cmd.Run()
}

// buildRectorCommand builds the command for running Rector (exported for testing).
func buildRectorCommand(opts RectorOptions) (string, []string) {
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "rector")
	cmdName := "rector"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"process"}

	if !opts.Fix {
		args = append(args, "--dry-run")
	}

	if opts.Diff {
		args = append(args, "--output-format", "diff")
	}

	if opts.ClearCache {
		args = append(args, "--clear-cache")
	}

	return cmdName, args
}
