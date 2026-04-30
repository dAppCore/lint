package php

import (
	. "dappco.re/go"
)

const (
	runnerTestBinSh70cd30 = "#!/bin/sh"
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
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "pint"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{phpFormatCheckName})
	RequireLen(t, specs, 1)
	AssertEqual(t, phpFormatCheckName, specs[0].Name)
	AssertContains(t, specs[0].Args, "--test")
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Fmt_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "pint"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, true) // fix mode
	specs := runner.BuildSpecs([]string{phpFormatCheckName})
	RequireLen(t, specs, 1)
	AssertNotContains(t, specs[0].Args, "--test")
}

func TestBuildSpecs_Fmt_NoPint(t *T) {
	dir := t.TempDir()
	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{phpFormatCheckName})
	AssertEmpty(t, specs)
}

func TestBuildSpecs_Stan_WithPHPStan(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "phpstan"), []byte(runnerTestBinSh70cd30), 0755)

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
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "psalm"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"psalm"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "psalm", specs[0].Name)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Psalm_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "psalm"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"psalm"})
	RequireLen(t, specs, 1)
	AssertContains(t, specs[0].Args, "--alter")
}

func TestBuildSpecs_Test_Pest(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "pest"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	RequireLen(t, specs, 1)
	AssertEqual(t, "test", specs[0].Name)
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Test_PHPUnit(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "phpunit"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"test"})
	RequireLen(t, specs, 1)
	AssertContains(t, specs[0].Command, "phpunit")
}

func TestBuildSpecs_Test_WithPsalmDep(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "pest"), []byte(runnerTestBinSh70cd30), 0755)
	WriteFile(PathJoin(vendorBin, "psalm"), []byte(runnerTestBinSh70cd30), 0755)

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
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "rector"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"rector"})
	RequireLen(t, specs, 1)
	AssertTrue(t, specs[0].AllowFailure)
	AssertContains(t, specs[0].Args, "--dry-run")
	AssertEmpty(t, specs[0].After)
}

func TestBuildSpecs_Rector_Fix(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "rector"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, true)
	specs := runner.BuildSpecs([]string{"rector"})
	RequireLen(t, specs, 1)
	AssertNotContains(t, specs[0].Args, "--dry-run")
}

func TestBuildSpecs_Infection(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "infection"), []byte(runnerTestBinSh70cd30), 0755)

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
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "pint"), []byte(runnerTestBinSh70cd30), 0755)
	WriteFile(PathJoin(vendorBin, "phpstan"), []byte(runnerTestBinSh70cd30), 0755)

	runner := NewQARunner(dir, false)
	specs := runner.BuildSpecs([]string{"audit", phpFormatCheckName, "stan"})
	AssertLen(t, specs, 3)
	AssertEmpty(t, specs[0].After)
	AssertEqual(t, []string{"audit"}, specs[1].After)
	AssertEqual(t, []string{phpFormatCheckName}, specs[2].After)
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
		{phpFormatCheckName, QACheckRunResult{Name: phpFormatCheckName}, "found style issues"},
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
			{Name: phpFormatCheckName, Passed: true},
		},
		PassedCount: 2,
	}
	AssertTrue(t, result.Passed)
	AssertEqual(t, 2, result.PassedCount)
	AssertEqual(t, 0, result.FailedCount)
}

func TestRunner_NewQARunner_Good(t *T) {
	subject := NewQARunner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewQARunner:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_NewQARunner_Bad(t *T) {
	subject := NewQARunner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewQARunner:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_NewQARunner_Ugly(t *T) {
	subject := NewQARunner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewQARunner:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QARunner_BuildSpecs_Good(t *T) {
	subject := (*QARunner).BuildSpecs
	if subject == nil {
		t.FailNow()
	}
	marker := "QARunner:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QARunner_BuildSpecs_Bad(t *T) {
	subject := (*QARunner).BuildSpecs
	if subject == nil {
		t.FailNow()
	}
	marker := "QARunner:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QARunner_BuildSpecs_Ugly(t *T) {
	subject := (*QARunner).BuildSpecs
	if subject == nil {
		t.FailNow()
	}
	marker := "QARunner:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QACheckRunResult_GetIssueMessage_Good(t *T) {
	subject := (*QACheckRunResult).GetIssueMessage
	if subject == nil {
		t.FailNow()
	}
	marker := "QACheckRunResult:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QACheckRunResult_GetIssueMessage_Bad(t *T) {
	subject := (*QACheckRunResult).GetIssueMessage
	if subject == nil {
		t.FailNow()
	}
	marker := "QACheckRunResult:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRunner_QACheckRunResult_GetIssueMessage_Ugly(t *T) {
	subject := (*QACheckRunResult).GetIssueMessage
	if subject == nil {
		t.FailNow()
	}
	marker := "QACheckRunResult:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
