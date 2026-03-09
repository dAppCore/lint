package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQARunner(t *testing.T) {
	runner := NewQARunner("/tmp/test", false)
	assert.NotNil(t, runner)
}

func TestBuildSpecs_Audit(t *testing.T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"audit"})
	require.Len(t, specs, 1)
	assert.Equal(t, "audit", specs[0].Name)
	assert.Equal(t, "composer", specs[0].Command)
	assert.Contains(t, specs[0].Args, "--format=summary")
}

func TestBuildSpecs_Fmt_WithPint(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"fmt"})
	require.Len(t, specs, 1)
	assert.Equal(t, "fmt", specs[0].Name)
	assert.Contains(t, specs[0].Args, "--test")
	assert.Equal(t, []string{"audit"}, specs[0].After)
}

func TestBuildSpecs_Fmt_Fix(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true) // fix mode
	specs := runner.BuildSpecs([]string{"fmt"})
	require.Len(t, specs, 1)
	assert.NotContains(t, specs[0].Args, "--test")
}

func TestBuildSpecs_Fmt_NoPint(t *testing.T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"fmt"})
	assert.Empty(t, specs)
}

func TestBuildSpecs_Stan_WithPHPStan(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpstan"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"stan"})
	require.Len(t, specs, 1)
	assert.Equal(t, "stan", specs[0].Name)
	assert.Contains(t, specs[0].Args, "--no-progress")
	assert.Equal(t, []string{"fmt"}, specs[0].After)
}

func TestBuildSpecs_Stan_NotDetected(t *testing.T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"stan"})
	assert.Empty(t, specs)
}

func TestBuildSpecs_Psalm_WithPsalm(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"psalm"})
	require.Len(t, specs, 1)
	assert.Equal(t, "psalm", specs[0].Name)
	assert.Equal(t, []string{"stan"}, specs[0].After)
}

func TestBuildSpecs_Psalm_Fix(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"psalm"})
	require.Len(t, specs, 1)
	assert.Contains(t, specs[0].Args, "--alter")
}

func TestBuildSpecs_Test_Pest(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pest"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	require.Len(t, specs, 1)
	assert.Equal(t, "test", specs[0].Name)
	assert.Equal(t, []string{"stan"}, specs[0].After)
}

func TestBuildSpecs_Test_PHPUnit(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpunit"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	require.Len(t, specs, 1)
	assert.Contains(t, specs[0].Command, "phpunit")
}

func TestBuildSpecs_Test_WithPsalmDep(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pest"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	require.Len(t, specs, 1)
	assert.Equal(t, []string{"psalm"}, specs[0].After)
}

func TestBuildSpecs_Test_NoRunner(t *testing.T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	assert.Empty(t, specs)
}

func TestBuildSpecs_Rector(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"rector"})
	require.Len(t, specs, 1)
	assert.True(t, specs[0].AllowFailure)
	assert.Contains(t, specs[0].Args, "--dry-run")
	assert.Equal(t, []string{"test"}, specs[0].After)
}

func TestBuildSpecs_Rector_Fix(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"rector"})
	require.Len(t, specs, 1)
	assert.NotContains(t, specs[0].Args, "--dry-run")
}

func TestBuildSpecs_Infection(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "infection"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"infection"})
	require.Len(t, specs, 1)
	assert.True(t, specs[0].AllowFailure)
	assert.Equal(t, []string{"test"}, specs[0].After)
}

func TestBuildSpecs_Unknown(t *testing.T) {
	runner := NewQARunner(t.TempDir(), false)
	specs := runner.BuildSpecs([]string{"unknown"})
	assert.Empty(t, specs)
}

func TestBuildSpecs_Multiple(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpstan"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"audit", "fmt", "stan"})
	assert.Len(t, specs, 3)
}

func TestQACheckRunResult_GetIssueMessage(t *testing.T) {
	tests := []struct {
		name     string
		result   QACheckRunResult
		expected string
	}{
		{"passed returns empty", QACheckRunResult{Passed: true, Name: "audit"}, ""},
		{"skipped returns empty", QACheckRunResult{Skipped: true, Name: "audit"}, ""},
		{"audit", QACheckRunResult{Name: "audit"}, "found vulnerabilities"},
		{"fmt", QACheckRunResult{Name: "fmt"}, "found style issues"},
		{"stan", QACheckRunResult{Name: "stan"}, "found analysis errors"},
		{"psalm", QACheckRunResult{Name: "psalm"}, "found type errors"},
		{"test", QACheckRunResult{Name: "test"}, "tests failed"},
		{"rector", QACheckRunResult{Name: "rector"}, "found refactoring suggestions"},
		{"infection", QACheckRunResult{Name: "infection"}, "mutation testing did not pass"},
		{"unknown", QACheckRunResult{Name: "whatever"}, "found issues"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.GetIssueMessage())
		})
	}
}

func TestQARunResult(t *testing.T) {
	result := QARunResult{
		Passed:   true,
		Duration: "1.5s",
		Results: []QACheckRunResult{
			{Name: "audit", Passed: true},
			{Name: "fmt", Passed: true},
		},
		PassedCount: 2,
	}
	assert.True(t, result.Passed)
	assert.Equal(t, 2, result.PassedCount)
	assert.Equal(t, 0, result.FailedCount)
}
