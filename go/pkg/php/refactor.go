package php

import (
	"context"
	"io"

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
	rectorConfig := core.PathJoin(dir, "rector.php")
	if fileExists(rectorConfig) {
		return true
	}

	// Check for vendor binary
	rectorBin := core.PathJoin(dir, "vendor", "bin", "rector")
	if fileExists(rectorBin) {
		return true
	}

	return false
}

// RunRector runs Rector for automated code refactoring.
func RunRector(ctx context.Context, opts RectorOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.RunRector", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	// Build command
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "rector")
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

	return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, nil)
}

// buildRectorCommand builds the command for running Rector (exported for testing).
func buildRectorCommand(opts RectorOptions) (string, []string) {
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "rector")
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
