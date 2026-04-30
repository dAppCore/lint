package php

import (
	"context"
	"io"

	core "dappco.re/go"
)

// AnalyseOptions configures PHP static analysis.
type AnalyseOptions struct {
	// Dir is the project directory (defaults to current working directory).
	Dir string

	// Level is the PHPStan analysis level (0-9).
	Level int

	// Paths limits analysis to specific paths.
	Paths []string

	// Memory is the memory limit for analysis (e.g., "2G").
	Memory string

	// JSON outputs results in JSON format.
	JSON bool

	// SARIF outputs results in SARIF format for GitHub Security tab.
	SARIF bool

	// Output is the writer for output (defaults to os.Stdout).
	Output io.Writer
}

// AnalyserType represents the detected static analyser.
type AnalyserType string

// Static analyser type constants.
const (
	// AnalyserPHPStan indicates standard PHPStan analyser.
	AnalyserPHPStan AnalyserType = "phpstan"
	// AnalyserLarastan indicates Laravel-specific Larastan analyser.
	AnalyserLarastan AnalyserType = "larastan"
)

// DetectAnalyser detects which static analyser is available in the project.
func DetectAnalyser(dir string) (AnalyserType, bool) {
	// Check for PHPStan config
	phpstanConfig := core.PathJoin(dir, "phpstan.neon")
	phpstanDistConfig := core.PathJoin(dir, "phpstan.neon.dist")

	hasConfig := fileExists(phpstanConfig) || fileExists(phpstanDistConfig)

	// Check for vendor binary
	phpstanBin := core.PathJoin(dir, "vendor", "bin", "phpstan")
	hasBin := fileExists(phpstanBin)

	if hasConfig || hasBin {
		// Check if it's Larastan (Laravel-specific PHPStan)
		larastanPath := core.PathJoin(dir, "vendor", "larastan", "larastan")
		if fileExists(larastanPath) {
			return AnalyserLarastan, true
		}
		// Also check nunomaduro/larastan
		larastanPath2 := core.PathJoin(dir, "vendor", "nunomaduro", "larastan")
		if fileExists(larastanPath2) {
			return AnalyserLarastan, true
		}
		return AnalyserPHPStan, true
	}

	return "", false
}

// Analyse runs PHPStan or Larastan for static analysis.
func Analyse(ctx context.Context, opts AnalyseOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.Analyse", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	// Check if analyser is available
	analyser, found := DetectAnalyser(opts.Dir)
	if !found {
		return core.Fail(core.E("php.Analyse", "no static analyser found (install PHPStan: composer require phpstan/phpstan --dev)", nil))
	}

	var cmdName string
	var args []string

	switch analyser {
	case AnalyserPHPStan, AnalyserLarastan:
		cmdName, args = buildPHPStanCommand(opts)
	}

	return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, nil)
}

// buildPHPStanCommand builds the command for running PHPStan.
func buildPHPStanCommand(opts AnalyseOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "phpstan")
	cmdName := "phpstan"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"analyse"}

	if opts.Level > 0 {
		args = append(args, "--level", core.Sprintf("%d", opts.Level))
	}

	if opts.Memory != "" {
		args = append(args, "--memory-limit", opts.Memory)
	}

	// Output format - SARIF takes precedence over JSON
	if opts.SARIF {
		args = append(args, "--error-format=sarif")
	} else if opts.JSON {
		args = append(args, "--error-format=json")
	}

	// Add specific paths if provided
	args = append(args, opts.Paths...)

	return cmdName, args
}

// =============================================================================
// Psalm Static Analysis
// =============================================================================

// PsalmOptions configures Psalm static analysis.
type PsalmOptions struct {
	Dir      string
	Level    int  // Error level (1=strictest, 8=most lenient)
	Fix      bool // Auto-fix issues where possible
	Baseline bool // Generate/update baseline file
	ShowInfo bool // Show info-level issues
	JSON     bool // Output in JSON format
	SARIF    bool // Output in SARIF format for GitHub Security tab
	Output   io.Writer
}

// PsalmType represents the detected Psalm configuration.
type PsalmType string

// Psalm configuration type constants.
const (
	// PsalmStandard indicates standard Psalm configuration.
	PsalmStandard PsalmType = "psalm"
)

// DetectPsalm checks if Psalm is available in the project.
func DetectPsalm(dir string) (PsalmType, bool) {
	// Check for psalm.xml config
	psalmConfig := core.PathJoin(dir, "psalm.xml")
	psalmDistConfig := core.PathJoin(dir, "psalm.xml.dist")

	hasConfig := fileExists(psalmConfig) || fileExists(psalmDistConfig)

	// Check for vendor binary
	psalmBin := core.PathJoin(dir, "vendor", "bin", "psalm")
	if fileExists(psalmBin) {
		return PsalmStandard, true
	}

	if hasConfig {
		return PsalmStandard, true
	}

	return "", false
}

// RunPsalm runs Psalm static analysis.
func RunPsalm(ctx context.Context, opts PsalmOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.RunPsalm", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	// Build command
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "psalm")
	cmdName := "psalm"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"--no-progress"}

	if opts.Level > 0 && opts.Level <= 8 {
		args = append(args, core.Sprintf("--error-level=%d", opts.Level))
	}

	if opts.Fix {
		args = append(args, "--alter", "--issues=all")
	}

	if opts.Baseline {
		args = append(args, "--set-baseline=psalm-baseline.xml")
	}

	if opts.ShowInfo {
		args = append(args, "--show-info=true")
	}

	// Output format - SARIF takes precedence over JSON
	if opts.SARIF {
		args = append(args, "--output-format=sarif")
	} else if opts.JSON {
		args = append(args, "--output-format=json")
	}

	return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, nil)
}
