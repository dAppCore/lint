package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DetectTestRunner
// =============================================================================

func TestDetectTestRunner_Good_Pest(t *testing.T) {
	dir := t.TempDir()

	// Create tests/Pest.php
	mkFile(t, filepath.Join(dir, "tests", "Pest.php"))

	runner := DetectTestRunner(dir)
	assert.Equal(t, TestRunnerPest, runner)
}

func TestDetectTestRunner_Good_PHPUnit(t *testing.T) {
	dir := t.TempDir()

	// No tests/Pest.php → defaults to PHPUnit
	runner := DetectTestRunner(dir)
	assert.Equal(t, TestRunnerPHPUnit, runner)
}

func TestDetectTestRunner_Good_PHPUnitWithTestsDir(t *testing.T) {
	dir := t.TempDir()

	// tests/ dir exists but no Pest.php
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0o755))

	runner := DetectTestRunner(dir)
	assert.Equal(t, TestRunnerPHPUnit, runner)
}

// =============================================================================
// buildPestCommand
// =============================================================================

func TestBuildPestCommand_Good_Defaults(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir}
	cmdName, args := buildPestCommand(opts)

	assert.Equal(t, "pest", cmdName)
	assert.Empty(t, args)
}

func TestBuildPestCommand_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "pest")
	mkFile(t, vendorBin)

	opts := TestOptions{Dir: dir}
	cmdName, _ := buildPestCommand(opts)

	assert.Equal(t, vendorBin, cmdName)
}

func TestBuildPestCommand_Good_Filter(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Filter: "TestLogin"}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--filter")
	assert.Contains(t, args, "TestLogin")
}

func TestBuildPestCommand_Good_Parallel(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Parallel: true}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--parallel")
}

func TestBuildPestCommand_Good_CoverageDefault(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--coverage")
}

func TestBuildPestCommand_Good_CoverageHTML(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "html"}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--coverage-html")
	assert.Contains(t, args, "coverage")
}

func TestBuildPestCommand_Good_CoverageClover(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "clover"}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--coverage-clover")
	assert.Contains(t, args, "coverage.xml")
}

func TestBuildPestCommand_Good_Groups(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Groups: []string{"unit", "integration"}}
	_, args := buildPestCommand(opts)

	// Should have --group unit --group integration
	groupCount := 0
	for _, a := range args {
		if a == "--group" {
			groupCount++
		}
	}
	assert.Equal(t, 2, groupCount)
	assert.Contains(t, args, "unit")
	assert.Contains(t, args, "integration")
}

func TestBuildPestCommand_Good_JUnit(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, JUnit: true}
	_, args := buildPestCommand(opts)

	assert.Contains(t, args, "--log-junit")
	assert.Contains(t, args, "test-results.xml")
}

func TestBuildPestCommand_Good_AllFlags(t *testing.T) {
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

	assert.Contains(t, args, "--filter")
	assert.Contains(t, args, "TestFoo")
	assert.Contains(t, args, "--parallel")
	assert.Contains(t, args, "--coverage-clover")
	assert.Contains(t, args, "--group")
	assert.Contains(t, args, "smoke")
	assert.Contains(t, args, "--log-junit")
}

// =============================================================================
// buildPHPUnitCommand
// =============================================================================

func TestBuildPHPUnitCommand_Good_Defaults(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir}
	cmdName, args := buildPHPUnitCommand(opts)

	assert.Equal(t, "phpunit", cmdName)
	assert.Empty(t, args)
}

func TestBuildPHPUnitCommand_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "phpunit")
	mkFile(t, vendorBin)

	opts := TestOptions{Dir: dir}
	cmdName, _ := buildPHPUnitCommand(opts)

	assert.Equal(t, vendorBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_Filter(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Filter: "TestCheckout"}
	_, args := buildPHPUnitCommand(opts)

	assert.Contains(t, args, "--filter")
	assert.Contains(t, args, "TestCheckout")
}

func TestBuildPHPUnitCommand_Good_Parallel_WithParatest(t *testing.T) {
	dir := t.TempDir()
	paratestBin := filepath.Join(dir, "vendor", "bin", "paratest")
	mkFile(t, paratestBin)

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	assert.Equal(t, paratestBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_Parallel_NoParatest(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	// Falls back to phpunit when paratest is not available
	assert.Equal(t, "phpunit", cmdName)
}

func TestBuildPHPUnitCommand_Good_Parallel_VendorPHPUnit_WithParatest(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpunit"))
	paratestBin := filepath.Join(dir, "vendor", "bin", "paratest")
	mkFile(t, paratestBin)

	opts := TestOptions{Dir: dir, Parallel: true}
	cmdName, _ := buildPHPUnitCommand(opts)

	// paratest takes precedence over phpunit when parallel is requested
	assert.Equal(t, paratestBin, cmdName)
}

func TestBuildPHPUnitCommand_Good_CoverageDefault(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true}
	_, args := buildPHPUnitCommand(opts)

	assert.Contains(t, args, "--coverage-text")
}

func TestBuildPHPUnitCommand_Good_CoverageHTML(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "html"}
	_, args := buildPHPUnitCommand(opts)

	assert.Contains(t, args, "--coverage-html")
	assert.Contains(t, args, "coverage")
}

func TestBuildPHPUnitCommand_Good_CoverageClover(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Coverage: true, CoverageFormat: "clover"}
	_, args := buildPHPUnitCommand(opts)

	assert.Contains(t, args, "--coverage-clover")
	assert.Contains(t, args, "coverage.xml")
}

func TestBuildPHPUnitCommand_Good_Groups(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, Groups: []string{"api", "slow"}}
	_, args := buildPHPUnitCommand(opts)

	groupCount := 0
	for _, a := range args {
		if a == "--group" {
			groupCount++
		}
	}
	assert.Equal(t, 2, groupCount)
	assert.Contains(t, args, "api")
	assert.Contains(t, args, "slow")
}

func TestBuildPHPUnitCommand_Good_JUnit(t *testing.T) {
	dir := t.TempDir()

	opts := TestOptions{Dir: dir, JUnit: true}
	_, args := buildPHPUnitCommand(opts)

	assert.Contains(t, args, "--log-junit")
	assert.Contains(t, args, "test-results.xml")
	assert.Contains(t, args, "--testdox")
}

func TestBuildPHPUnitCommand_Good_AllFlags(t *testing.T) {
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

	assert.Equal(t, filepath.Join(dir, "vendor", "bin", "paratest"), cmdName)
	assert.Contains(t, args, "--filter")
	assert.Contains(t, args, "TestBar")
	assert.Contains(t, args, "--coverage-html")
	assert.Contains(t, args, "--group")
	assert.Contains(t, args, "feature")
	assert.Contains(t, args, "--log-junit")
	assert.Contains(t, args, "--testdox")
}
