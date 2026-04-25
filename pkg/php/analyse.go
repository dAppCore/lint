package php

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	coreerr "dappco.re/go/core/log"
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
	phpstanConfig := filepath.Join(dir, "phpstan.neon")
	phpstanDistConfig := filepath.Join(dir, "phpstan.neon.dist")

	hasConfig := fileExists(phpstanConfig) || fileExists(phpstanDistConfig)

	// Check for vendor binary
	phpstanBin := filepath.Join(dir, "vendor", "bin", "phpstan")
	hasBin := fileExists(phpstanBin)

	if hasConfig || hasBin {
		// Check if it's Larastan (Laravel-specific PHPStan)
		larastanPath := filepath.Join(dir, "vendor", "larastan", "larastan")
		if fileExists(larastanPath) {
			return AnalyserLarastan, true
		}
		// Also check nunomaduro/larastan
		larastanPath2 := filepath.Join(dir, "vendor", "nunomaduro", "larastan")
		if fileExists(larastanPath2) {
			return AnalyserLarastan, true
		}
		return AnalyserPHPStan, true
	}

	return "", false
}

// Analyse runs PHPStan or Larastan for static analysis.
func Analyse(ctx context.Context, opts AnalyseOptions) error {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return coreerr.E("php.Analyse", "get working directory", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Check if analyser is available
	analyser, found := DetectAnalyser(opts.Dir)
	if !found {
		return coreerr.E("php.Analyse", "no static analyser found (install PHPStan: composer require phpstan/phpstan --dev)", nil)
	}

	var cmdName string
	var args []string

	switch analyser {
	case AnalyserPHPStan, AnalyserLarastan:
		cmdName, args = buildPHPStanCommand(opts)
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = opts.Dir
	cmd.Stdout = opts.Output
	cmd.Stderr = opts.Output

	return cmd.Run()
}

// buildPHPStanCommand builds the command for running PHPStan.
func buildPHPStanCommand(opts AnalyseOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "phpstan")
	cmdName := "phpstan"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"analyse"}

	if opts.Level > 0 {
		args = append(args, "--level", fmt.Sprintf("%d", opts.Level))
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
	psalmConfig := filepath.Join(dir, "psalm.xml")
	psalmDistConfig := filepath.Join(dir, "psalm.xml.dist")

	hasConfig := fileExists(psalmConfig) || fileExists(psalmDistConfig)

	// Check for vendor binary
	psalmBin := filepath.Join(dir, "vendor", "bin", "psalm")
	if fileExists(psalmBin) {
		return PsalmStandard, true
	}

	if hasConfig {
		return PsalmStandard, true
	}

	return "", false
}

// RunPsalm runs Psalm static analysis.
func RunPsalm(ctx context.Context, opts PsalmOptions) error {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return coreerr.E("php.RunPsalm", "get working directory", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Build command
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "psalm")
	cmdName := "psalm"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	args := []string{"--no-progress"}

	if opts.Level > 0 && opts.Level <= 8 {
		args = append(args, fmt.Sprintf("--error-level=%d", opts.Level))
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

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = opts.Dir
	cmd.Stdout = opts.Output
	cmd.Stderr = opts.Output

	return cmd.Run()
}
