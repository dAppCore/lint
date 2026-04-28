package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

func TestNewQARunner(t *T) {
	runner := NewQARunner("/tmp/test", false)
	AssertNotNil(t, runner)
	AssertEqual(t, "/tmp/test", runner.dir)
}

func TestBuildSpecs_Audit(t *T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"audit"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "audit", specs[0].Name)
	AssertEqual(t, "composer", specs[0].Command)
	AssertContains(t, specs[0].Args, "--format=summary")
}

func TestBuildSpecs_Fmt_WithPint(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"fmt"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "fmt", specs[0].Name)
	AssertContains(t, specs[0].Args, "--test")
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Fmt_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true) // fix mode
	specs := runner.BuildSpecs([]string{"fmt"})
	RequireLen(t, specs, 1)
	AssertNotContains(t, specs[0].Args, "--test")
}

func TestBuildSpecs_Fmt_NoPint(t *T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"fmt"})
	AssertEmpty(t, specs)
}

func TestBuildSpecs_Stan_WithPHPStan(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpstan"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"stan"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "stan", specs[0].Name)
	AssertContains(t, specs[0].Args, "--no-progress")
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Stan_NotDetected(t *T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"stan"})
	AssertEmpty(t, specs)
}

func TestBuildSpecs_Psalm_WithPsalm(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"psalm"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "psalm", specs[0].Name)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Psalm_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"psalm"})
	RequireLen(t, specs, 1)
	AssertContains(t, specs[0].Args, "--alter")
}

func TestBuildSpecs_Test_Pest(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pest"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "test", specs[0].Name)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Test_PHPUnit(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpunit"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	RequireLen(t, specs, 1)
	AssertContains(t, specs[0].Command, "phpunit")
}

func TestBuildSpecs_Test_WithPsalmDep(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pest"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	RequireLen(t, specs, 1)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Test_NoRunner(t *T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	AssertEmpty(t, specs)
}

func TestBuildSpecs_Rector(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"rector"})
	RequireLen(t, specs, 1)
	AssertTrue(t, specs[0].AllowFailure)
	AssertContains(t, specs[0].Args, "--dry-run")
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Rector_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"rector"})
	RequireLen(t, specs, 1)
	AssertNotContains(t, specs[0].Args, "--dry-run")
}

func TestBuildSpecs_Infection(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "infection"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"infection"})
	RequireLen(t, specs, 1)
	AssertTrue(t, specs[0].AllowFailure)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Unknown(t *T) {
	runner := NewQARunner(t.TempDir(), false)
	specs := runner.BuildSpecs([]string{"unknown"})
	AssertEmpty(t, specs)
}

func TestBuildSpecs_Multiple(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "pint"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "phpstan"), []byte("#!/bin/sh"), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"audit", "fmt", "stan"})
	AssertLen(t, specs, 3)
	AssertEmpty(t, specs[0].After)
	AssertEqual(t, []string{"audit"}, specs[1].After)
	AssertEqual(t, []string{"fmt"}, specs[2].After)
}

func TestQACheckRunResult_GetIssueMessage(t *T) {
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
		t.Run(tt.name, func(t *T) {
			AssertEqual(t, tt.expected, tt.result.GetIssueMessage())
		})
	}
}

func TestQARunResult(t *T) {
	result := QARunResult{
		Passed:   true,
		Duration: "1.5s",
		Results: []QACheckRunResult{
			{Name: "audit", Passed: true},
			{Name: "fmt", Passed: true},
		},
		PassedCount: 2,
	}
	AssertTrue(t, result.Passed)
	AssertEqual(t, 2, result.PassedCount)
	AssertEqual(t, 0, result.FailedCount)
}
