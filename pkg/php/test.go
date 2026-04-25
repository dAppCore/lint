package php

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	coreerr "dappco.re/go/core/log"
)

// TestOptions configures PHP test execution.
type TestOptions struct {
	// Dir is the project directory (defaults to current working directory).
	Dir string

	// Filter filters tests by name pattern.
	Filter string

	// Parallel runs tests in parallel.
	Parallel bool

	// Coverage generates code coverage.
	Coverage bool

	// CoverageFormat is the coverage output format (text, html, clover).
	CoverageFormat string

	// Groups runs only tests in the specified groups.
	Groups []string

	// JUnit outputs results in JUnit XML format via --log-junit.
	JUnit bool

	// JUnitPath overrides the JUnit report path. Defaults to test-results.xml.
	JUnitPath string

	// Output is the writer for test output (defaults to os.Stdout).
	Output io.Writer
}

// TestRunner represents the detected test runner.
type TestRunner string

// Test runner type constants.
const (
	// TestRunnerPest indicates Pest testing framework.
	TestRunnerPest TestRunner = "pest"
	// TestRunnerPHPUnit indicates PHPUnit testing framework.
	TestRunnerPHPUnit TestRunner = "phpunit"
)

// DetectTestRunner detects which test runner is available in the project.
// Returns Pest if tests/Pest.php exists, otherwise PHPUnit.
func DetectTestRunner(dir string) TestRunner {
	pestFile := filepath.Join(dir, "tests", "Pest.php")
	if fileExists(pestFile) {
		return TestRunnerPest
	}

	return TestRunnerPHPUnit
}

// RunTests runs PHPUnit or Pest tests.
func RunTests(ctx context.Context, opts TestOptions) error {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return coreerr.E("php.RunTests", "get working directory", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	if opts.JUnit && opts.JUnitPath == "" {
		reportFile, err := os.CreateTemp("", "core-qa-junit-*.xml")
		if err != nil {
			return coreerr.E("php.RunTests", "create JUnit report file", err)
		}
		if closeErr := reportFile.Close(); closeErr != nil {
			return coreerr.E("php.RunTests", "close JUnit report file", closeErr)
		}
		opts.JUnitPath = reportFile.Name()
		defer os.Remove(opts.JUnitPath)
	}

	// Detect test runner
	runner := DetectTestRunner(opts.Dir)

	// Build command based on runner
	var cmdName string
	var args []string

	switch runner {
	case TestRunnerPest:
		cmdName, args = buildPestCommand(opts)
	default:
		cmdName, args = buildPHPUnitCommand(opts)
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = opts.Dir
	cmd.Stdin = os.Stdin

	// Set XDEBUG_MODE=coverage to avoid PHPUnit 11 warning
	cmd.Env = append(os.Environ(), "XDEBUG_MODE=coverage")

	if !opts.JUnit {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
		return cmd.Run()
	}

	var machineOutput bytes.Buffer
	cmd.Stdout = &machineOutput
	cmd.Stderr = &machineOutput

	runErr := cmd.Run()
	reportErr := emitJUnitReport(opts.Output, opts.JUnitPath)
	if runErr != nil {
		return runErr
	}
	return reportErr
}

// RunParallel runs tests in parallel using the appropriate runner.
func RunParallel(ctx context.Context, opts TestOptions) error {
	opts.Parallel = true
	return RunTests(ctx, opts)
}

// buildPestCommand builds the command for running Pest tests.
func buildPestCommand(opts TestOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "pest")
	cmdName := "pest"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	var args []string

	if opts.Filter != "" {
		args = append(args, "--filter", opts.Filter)
	}

	if opts.Parallel {
		args = append(args, "--parallel")
	}

	if opts.Coverage {
		switch opts.CoverageFormat {
		case "html":
			args = append(args, "--coverage-html", "coverage")
		case "clover":
			args = append(args, "--coverage-clover", "coverage.xml")
		default:
			args = append(args, "--coverage")
		}
	}

	for _, group := range opts.Groups {
		args = append(args, "--group", group)
	}

	if opts.JUnit {
		args = append(args, "--log-junit", junitReportPath(opts))
	}

	return cmdName, args
}

// buildPHPUnitCommand builds the command for running PHPUnit tests.
func buildPHPUnitCommand(opts TestOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "phpunit")
	cmdName := "phpunit"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	var args []string

	if opts.Filter != "" {
		args = append(args, "--filter", opts.Filter)
	}

	if opts.Parallel {
		// PHPUnit uses paratest for parallel execution
		paratestBin := filepath.Join(opts.Dir, "vendor", "bin", "paratest")
		if fileExists(paratestBin) {
			cmdName = paratestBin
		}
	}

	if opts.Coverage {
		switch opts.CoverageFormat {
		case "html":
			args = append(args, "--coverage-html", "coverage")
		case "clover":
			args = append(args, "--coverage-clover", "coverage.xml")
		default:
			args = append(args, "--coverage-text")
		}
	}

	for _, group := range opts.Groups {
		args = append(args, "--group", group)
	}

	if opts.JUnit {
		args = append(args, "--log-junit", junitReportPath(opts))
	}

	return cmdName, args
}

func junitReportPath(opts TestOptions) string {
	if opts.JUnitPath != "" {
		return opts.JUnitPath
	}
	return "test-results.xml"
}

func emitJUnitReport(output io.Writer, reportPath string) error {
	report, err := os.ReadFile(reportPath)
	if err != nil {
		return coreerr.E("php.emitJUnitReport", "read JUnit report", err)
	}

	if _, err := output.Write(report); err != nil {
		return coreerr.E("php.emitJUnitReport", "write JUnit report", err)
	}

	if len(report) == 0 || report[len(report)-1] != '\n' {
		if _, err := io.WriteString(output, "\n"); err != nil {
			return coreerr.E("php.emitJUnitReport", "terminate JUnit report", err)
		}
	}

	return nil
}
