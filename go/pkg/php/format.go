// Package php provides linting and quality tools for PHP projects.
package php

import (
	"context"
	"io"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

// fileExists reports whether the named file or directory exists.
func fileExists(path string) bool {
	return coreio.Local.Exists(path)
}

// FormatOptions configures PHP code formatting.
type FormatOptions struct {
	// Dir is the project directory (defaults to current working directory).
	Dir string

	// Fix automatically fixes formatting issues.
	Fix bool

	// Diff shows a diff of changes instead of modifying files.
	Diff bool

	// JSON outputs results in JSON format.
	JSON bool

	// Paths limits formatting to specific paths.
	Paths []string

	// Output is the writer for output (defaults to os.Stdout).
	Output io.Writer
}

// FormatterType represents the detected formatter.
type FormatterType string

// Formatter type constants.
const (
	// FormatterPint indicates Laravel Pint code formatter.
	FormatterPint FormatterType = "pint"
)

// DetectFormatter detects which formatter is available in the project.
func DetectFormatter(dir string) (FormatterType, bool) {
	// Check for Pint config
	pintConfig := core.PathJoin(dir, "pint.json")
	if fileExists(pintConfig) {
		return FormatterPint, true
	}

	// Check for vendor binary
	pintBin := core.PathJoin(dir, "vendor", "bin", "pint")
	if fileExists(pintBin) {
		return FormatterPint, true
	}

	return "", false
}

// Format runs Laravel Pint to format PHP code.
func Format(ctx context.Context, opts FormatOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.Format", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	// Check if formatter is available
	formatter, found := DetectFormatter(opts.Dir)
	if !found {
		return core.Fail(core.E("php.Format", "no formatter found (install Laravel Pint: composer require laravel/pint --dev)", nil))
	}

	var cmdName string
	var args []string

	switch formatter {
	case FormatterPint:
		cmdName, args = buildPintCommand(opts)
	}

	return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, nil)
}

// buildPintCommand builds the command for running Laravel Pint.
func buildPintCommand(opts FormatOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "pint")
	cmdName := "pint"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	var args []string

	if !opts.Fix {
		args = append(args, "--test")
	}

	if opts.Diff {
		args = append(args, "--diff")
	}

	if opts.JSON {
		args = append(args, "--format=json")
	}

	// Add specific paths if provided
	args = append(args, opts.Paths...)

	return cmdName, args
}
