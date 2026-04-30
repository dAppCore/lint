package php

import (
	"context"
	"io"

	core "dappco.re/go"
)

const (
	testPhpEmitjunitreportf76608 = "php.emitJUnitReport"
	testPhpRuntests8829d3        = "php.RunTests"
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
	pestFile := core.PathJoin(dir, "tests", "Pest.php")
	if fileExists(pestFile) {
		return TestRunnerPest
	}

	return TestRunnerPHPUnit
}

// RunTests runs PHPUnit or Pest tests.
func RunTests(ctx context.Context, opts TestOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E(testPhpRuntests8829d3, "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	if opts.JUnit && opts.JUnitPath == "" {
		reportDir := core.MkdirTemp("", "core-qa-junit-*")
		if !reportDir.OK {
			err, _ := reportDir.Value.(error)
			return core.Fail(core.E(testPhpRuntests8829d3, "create JUnit report directory", err))
		}
		opts.JUnitPath = core.PathJoin(reportDir.Value.(string), "test-results.xml")
		defer core.RemoveAll(reportDir.Value.(string))
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

	// Set XDEBUG_MODE=coverage to avoid PHPUnit 11 warning
	env := []string{"XDEBUG_MODE=coverage"}

	if !opts.JUnit {
		return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, env)
	}

	machineOutput := core.NewBuffer()
	runResult := runPHPCommand(ctx, opts.Dir, cmdName, args, machineOutput, env)
	reportResult := emitJUnitReport(opts.Output, opts.JUnitPath)
	if !runResult.OK {
		return runResult
	}
	return reportResult
}

// RunParallel runs tests in parallel using the appropriate runner.
func RunParallel(ctx context.Context, opts TestOptions) core.Result {
	opts.Parallel = true
	return RunTests(ctx, opts)
}

// buildPestCommand builds the command for running Pest tests.
func buildPestCommand(opts TestOptions) (string, []string) {
	// Check for vendor binary first
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "pest")
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
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "phpunit")
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
		paratestBin := core.PathJoin(opts.Dir, "vendor", "bin", "paratest")
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

func emitJUnitReport(output io.Writer, reportPath string) core.Result {
	read := core.ReadFile(reportPath)
	if !read.OK {
		err, _ := read.Value.(error)
		return core.Fail(core.E(testPhpEmitjunitreportf76608, "read JUnit report", err))
	}
	report := read.Value.([]byte)

	if _, err := output.Write(report); err != nil {
		return core.Fail(core.E(testPhpEmitjunitreportf76608, "write JUnit report", err))
	}

	if len(report) == 0 || report[len(report)-1] != '\n' {
		if write := core.WriteString(output, "\n"); !write.OK {
			err, _ := write.Value.(error)
			return core.Fail(core.E(testPhpEmitjunitreportf76608, "terminate JUnit report", err))
		}
	}

	return core.Ok(nil)
}
