package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

const (
	testTestCoverageClover2083f2 = "--coverage-clover"
	testTestCoverageHtml06c6e2   = "--coverage-html"
	testTestFilter946eb5         = "--filter"
	testTestGroup389856          = "--group"
	testTestLogJunit792b50       = "--log-junit"
)

// =============================================================================
// DetectTestRunner
// =============================================================================

func TestDetectTestRunner_Good_Pest(t *T) {
	dir := t.TempDir()

	// Create tests/Pest.php
	mkFile(t, filepath.Join(dir, "tests", "Pest.php"))

	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPest, runner)
}

func TestDetectTestRunner_Good_PHPUnit(t *T) {
	dir := t.TempDir()

	// No tests/Pest.php → defaults to PHPUnit
	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPHPUnit, runner)
}

func TestDetectTestRunner_Good_PHPUnitWithTestsDir(t *T) {
	dir := t.TempDir()

	// tests/ dir exists but no Pest.php
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0o755))

	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPHPUnit, runner)
}

// =============================================================================
// buildPestCommand
// =============================================================================

func TestBuildPestCommand_Good_Defaults(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir}
	cmdName, args := buildPestCommand(opts)

	AssertEqual(t, "pest", cmdName)
	AssertEmpty(t, args)
}

func TestBuildPestCommand_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "pest")
	mkFile(t, vendorBin)

	opts := TestOptions{Dir: dir}
	cmdName, _ := buildPestCommand(opts)

	AssertEqual(t, vendorBin, cmdName)
}

func TestBuildPestCommand_Good_Filter(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Filter: "TestLogin"}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, testTestFilter946eb5)
	AssertContains(t, args, "TestLogin")
}

func TestBuildPestCommand_Good_Parallel(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Parallel: true}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, "--parallel")
}

func TestBuildPestCommand_Good_CoverageDefault(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, "--coverage")
}

func TestBuildPestCommand_Good_CoverageHTML(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "html"}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, testTestCoverageHtml06c6e2)
	AssertContains(t, args, "coverage")
}

func TestBuildPestCommand_Good_CoverageClover(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "clover"}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, testTestCoverageClover2083f2)
	AssertContains(t, args, "coverage.xml")
}

func TestBuildPestCommand_Good_Groups(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Groups: []string{"unit", "integration"}}
	_, args := buildPestCommand(opts)

	// Should have --group unit --group integration
	groupCount := 0
	for _, a := range args {
		if a == testTestGroup389856 {
			groupCount++
		}
	}
	AssertEqual(t, 2, groupCount)
	AssertContains(t, args, "unit")
	AssertContains(t, args, "integration")
}

func TestBuildPestCommand_Good_JUnit(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, JUnit: true}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, testTestLogJunit792b50)
	AssertContains(t, args, "test-results.xml")
}

func TestBuildPestCommand_Good_AllFlags(t *T) {
	dir := t.TempDir()

	opts := TestOptions{
		Dir:            dir,
		Filter:         "TestFoo",
		Parallel:       true,
		Coverage:       true,
		CoverageFormat: "clover",
		Groups:         []string{"smoke"},
		JUnit:          true,
	}
	_, args := buildPestCommand(opts)

	AssertContains(t, args, testTestFilter946eb5)
	AssertContains(t, args, "TestFoo")
	AssertContains(t, args, "--parallel")
	AssertContains(t, args, testTestCoverageClover2083f2)
	AssertContains(t, args, testTestGroup389856)
	AssertContains(t, args, "smoke")
	AssertContains(t, args, testTestLogJunit792b50)
}

// =============================================================================
// buildPHPUnitCommand
// =============================================================================

func TestBuildPHPUnitCommand_Good_Defaults(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir}
	cmdName, args := buildPHPUnitCommand(opts)

	AssertEqual(t, "phpunit", cmdName)
	AssertEmpty(t, args)
}

func TestBuildPHPUnitCommand_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "phpunit")
	mkFile(t, vendorBin)

	opts := TestOptions{Dir: dir}
	cmdName, _ := buildPHPUnitCommand(opts)

	AssertEqual(t, vendorBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_Filter(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Filter: "TestCheckout"}
	_, args := buildPHPUnitCommand(opts)

	AssertContains(t, args, testTestFilter946eb5)
	AssertContains(t, args, "TestCheckout")
}

func TestBuildPHPUnitCommand_Good_Parallel_WithParatest(t *T) {
	dir := t.TempDir()
	paratestBin := filepath.Join(dir, "vendor", "bin", "paratest")
	mkFile(t, paratestBin)

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	AssertEqual(t, paratestBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_Parallel_NoParatest(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	// Falls back to phpunit when paratest is not available
	AssertEqual(t, "phpunit", cmdName)
}

func TestBuildPHPUnitCommand_Good_Parallel_VendorPHPUnit_WithParatest(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpunit"))
	paratestBin := filepath.Join(dir, "vendor", "bin", "paratest")
	mkFile(t, paratestBin)

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	// paratest takes precedence over phpunit when parallel is requested
	AssertEqual(t, paratestBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_CoverageDefault(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true}
	_, args := buildPHPUnitCommand(opts)

	AssertContains(t, args, "--coverage-text")
}

func TestBuildPHPUnitCommand_Good_CoverageHTML(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "html"}
	_, args := buildPHPUnitCommand(opts)

	AssertContains(t, args, testTestCoverageHtml06c6e2)
	AssertContains(t, args, "coverage")
}

func TestBuildPHPUnitCommand_Good_CoverageClover(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "clover"}
	_, args := buildPHPUnitCommand(opts)

	AssertContains(t, args, testTestCoverageClover2083f2)
	AssertContains(t, args, "coverage.xml")
}

func TestBuildPHPUnitCommand_Good_Groups(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Groups: []string{"api", "slow"}}
	_, args := buildPHPUnitCommand(opts)

	groupCount := 0
	for _, a := range args {
		if a == testTestGroup389856 {
			groupCount++
		}
	}
	AssertEqual(t, 2, groupCount)
	AssertContains(t, args, "api")
	AssertContains(t, args, "slow")
}

func TestBuildPHPUnitCommand_Good_JUnit(t *T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, JUnit: true}
	_, args := buildPHPUnitCommand(opts)

	AssertContains(t, args, testTestLogJunit792b50)
	AssertContains(t, args, "test-results.xml")
	AssertNotContains(t, args, "--testdox")
}

func TestBuildPHPUnitCommand_Good_AllFlags(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "paratest"))

	opts := TestOptions{
		Dir:            dir,
		Filter:         "TestBar",
		Parallel:       true,
		Coverage:       true,
		CoverageFormat: "html",
		Groups:         []string{"feature"},
		JUnit:          true,
	}
	cmdName, args := buildPHPUnitCommand(opts)

	AssertEqual(t, filepath.Join(dir, "vendor", "bin", "paratest"), cmdName)
	AssertContains(t, args, testTestFilter946eb5)
	AssertContains(t, args, "TestBar")
	AssertContains(t, args, testTestCoverageHtml06c6e2)
	AssertContains(t, args, testTestGroup389856)
	AssertContains(t, args, "feature")
	AssertContains(t, args, testTestLogJunit792b50)
	AssertNotContains(t, args, "--testdox")
}
