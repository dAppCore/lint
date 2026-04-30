package php

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	. "dappco.re/go"
)

const (
	ax7TestExit05c15a7            = "exit 0"
	ax7TestInfectionJsona68fc9    = "infection.json"
	ax7TestPestPhpcefea2          = "Pest.php"
	ax7TestPhp96d681              = "<?php\n"
	ax7TestPintJson7e00da         = "pint.json"
	ax7TestPrintfAdvisories19e053 = "printf '{\"advisories\":{}}'"
	ax7TestRectorPhp92466b        = "rector.php"
)

func ax7PHPProject(t *T) string {
	t.Helper()
	dir := t.TempDir()
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "vendor", "bin"), 0o755))
	return dir
}

func ax7PHPExecutable(t *T, dir string, name string, body string) string {
	t.Helper()
	path := filepath.Join(dir, "vendor", "bin", name)
	RequireNoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	RequireNoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	return path
}

func ax7PATHExecutable(t *T, name string, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	RequireNoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	t.Setenv("PATH", dir)
	return path
}

func TestPHP_DetectFormatter_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestPintJson7e00da), []byte(`{}`), 0o644))
	formatter, ok := DetectFormatter(dir)
	AssertTrue(t, ok)
	AssertEqual(t, FormatterPint, formatter)
}

func TestPHP_DetectFormatter_Bad(t *T) {
	dir := t.TempDir()
	formatter, ok := DetectFormatter(dir)
	AssertFalse(t, ok)
	AssertEqual(t, FormatterType(""), formatter)
}

func TestPHP_DetectFormatter_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "pint", ax7TestExit05c15a7)
	formatter, ok := DetectFormatter(dir)
	AssertTrue(t, ok)
	AssertEqual(t, FormatterPint, formatter)
}

func TestPHP_Format_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestPintJson7e00da), []byte(`{}`), 0o644))
	ax7PHPExecutable(t, dir, "pint", ax7TestExit05c15a7)
	var output bytes.Buffer
	err := Format(context.Background(), FormatOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "", output.String())
}

func TestPHP_Format_Bad(t *T) {
	dir := t.TempDir()
	err := Format(context.Background(), FormatOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "no formatter")
}

func TestPHP_Format_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestPintJson7e00da), []byte(`{}`), 0o644))
	ax7PHPExecutable(t, dir, "pint", "printf formatted")
	var output bytes.Buffer
	err := Format(context.Background(), FormatOptions{Dir: dir, Fix: true, Diff: true, JSON: true, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "formatted", output.String())
}

func TestPHP_DetectAnalyser_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "phpstan.neon"), []byte("parameters: {}\n"), 0o644))
	analyser, ok := DetectAnalyser(dir)
	AssertTrue(t, ok)
	AssertEqual(t, AnalyserPHPStan, analyser)
}

func TestPHP_DetectAnalyser_Bad(t *T) {
	dir := t.TempDir()
	analyser, ok := DetectAnalyser(dir)
	AssertFalse(t, ok)
	AssertEqual(t, AnalyserType(""), analyser)
}

func TestPHP_DetectAnalyser_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "vendor", "larastan", "larastan"), 0o755))
	ax7PHPExecutable(t, dir, "phpstan", ax7TestExit05c15a7)
	analyser, ok := DetectAnalyser(dir)
	AssertTrue(t, ok)
	AssertEqual(t, AnalyserLarastan, analyser)
}

func TestPHP_Analyse_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "phpstan.neon"), []byte("parameters: {}\n"), 0o644))
	ax7PHPExecutable(t, dir, "phpstan", ax7TestExit05c15a7)
	var output bytes.Buffer
	err := Analyse(context.Background(), AnalyseOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "", output.String())
}

func TestPHP_Analyse_Bad(t *T) {
	dir := t.TempDir()
	err := Analyse(context.Background(), AnalyseOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "no static analyser")
}

func TestPHP_Analyse_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "phpstan", "printf analysed")
	var output bytes.Buffer
	err := Analyse(context.Background(), AnalyseOptions{Dir: dir, Level: 9, Memory: "1G", JSON: true, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "analysed", output.String())
}

func TestPHP_DetectPsalm_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "psalm.xml"), []byte("<psalm/>"), 0o644))
	psalm, ok := DetectPsalm(dir)
	AssertTrue(t, ok)
	AssertEqual(t, PsalmStandard, psalm)
}

func TestPHP_DetectPsalm_Bad(t *T) {
	dir := t.TempDir()
	psalm, ok := DetectPsalm(dir)
	AssertFalse(t, ok)
	AssertEqual(t, PsalmType(""), psalm)
}

func TestPHP_DetectPsalm_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "psalm", ax7TestExit05c15a7)
	psalm, ok := DetectPsalm(dir)
	AssertTrue(t, ok)
	AssertEqual(t, PsalmStandard, psalm)
}

func TestPHP_RunPsalm_Good(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "psalm", ax7TestExit05c15a7)
	var output bytes.Buffer
	err := RunPsalm(context.Background(), PsalmOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "", output.String())
}

func TestPHP_RunPsalm_Bad(t *T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := RunPsalm(context.Background(), PsalmOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "psalm")
}

func TestPHP_RunPsalm_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "psalm", "printf psalm")
	var output bytes.Buffer
	err := RunPsalm(context.Background(), PsalmOptions{Dir: dir, Level: 99, Fix: true, Baseline: true, ShowInfo: true, SARIF: true, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "psalm", output.String())
}

func TestPHP_DetectTestRunner_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "tests", ax7TestPestPhpcefea2), []byte(ax7TestPhp96d681), 0o644))
	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPest, runner)
	AssertNotEqual(t, TestRunnerPHPUnit, runner)
}

func TestPHP_DetectTestRunner_Bad(t *T) {
	dir := t.TempDir()
	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPHPUnit, runner)
	AssertNotEqual(t, TestRunnerPest, runner)
}

func TestPHP_DetectTestRunner_Ugly(t *T) {
	dir := filepath.Join(t.TempDir(), "missing")
	runner := DetectTestRunner(dir)
	AssertEqual(t, TestRunnerPHPUnit, runner)
	AssertFalse(t, fileExists(filepath.Join(dir, "tests", ax7TestPestPhpcefea2)))
}

func TestPHP_RunTests_Good(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "phpunit", "printf tests")
	var output bytes.Buffer
	err := RunTests(context.Background(), TestOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "tests", output.String())
}

func TestPHP_RunTests_Bad(t *T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := RunTests(context.Background(), TestOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "phpunit")
}

func TestPHP_RunTests_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "tests", ax7TestPestPhpcefea2), []byte(ax7TestPhp96d681), 0o644))
	ax7PHPExecutable(t, dir, "pest", "printf pest")
	var output bytes.Buffer
	err := RunTests(context.Background(), TestOptions{Dir: dir, Parallel: true, Coverage: true, CoverageFormat: "clover", Groups: []string{"unit"}, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "pest", output.String())
}

func TestPHP_RunParallel_Good(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "phpunit", "printf parallel")
	var output bytes.Buffer
	err := RunParallel(context.Background(), TestOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "parallel", output.String())
}

func TestPHP_RunParallel_Bad(t *T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := RunParallel(context.Background(), TestOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "phpunit")
}

func TestPHP_RunParallel_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "tests"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "tests", ax7TestPestPhpcefea2), []byte(ax7TestPhp96d681), 0o644))
	ax7PHPExecutable(t, dir, "pest", "printf parallel-pest")
	var output bytes.Buffer
	err := RunParallel(context.Background(), TestOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "parallel-pest", output.String())
}

func TestPHP_DetectRector_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestRectorPhp92466b), []byte(ax7TestPhp96d681), 0o644))
	got := DetectRector(dir)
	AssertTrue(t, got)
	AssertTrue(t, fileExists(filepath.Join(dir, ax7TestRectorPhp92466b)))
}

func TestPHP_DetectRector_Bad(t *T) {
	dir := t.TempDir()
	got := DetectRector(dir)
	AssertFalse(t, got)
	AssertFalse(t, fileExists(filepath.Join(dir, ax7TestRectorPhp92466b)))
}

func TestPHP_DetectRector_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "rector", ax7TestExit05c15a7)
	got := DetectRector(dir)
	AssertTrue(t, got)
	AssertTrue(t, fileExists(filepath.Join(dir, "vendor", "bin", "rector")))
}

func TestPHP_RunRector_Good(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "rector", "printf rector")
	var output bytes.Buffer
	err := RunRector(context.Background(), RectorOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "rector", output.String())
}

func TestPHP_RunRector_Bad(t *T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := RunRector(context.Background(), RectorOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "rector")
}

func TestPHP_RunRector_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "rector", "printf rector")
	var output bytes.Buffer
	err := RunRector(context.Background(), RectorOptions{Dir: dir, Fix: true, Diff: true, ClearCache: true, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "rector", output.String())
}

func TestPHP_DetectInfection_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestInfectionJsona68fc9), []byte(`{}`), 0o644))
	got := DetectInfection(dir)
	AssertTrue(t, got)
	AssertTrue(t, fileExists(filepath.Join(dir, ax7TestInfectionJsona68fc9)))
}

func TestPHP_DetectInfection_Bad(t *T) {
	dir := t.TempDir()
	got := DetectInfection(dir)
	AssertFalse(t, got)
	AssertFalse(t, fileExists(filepath.Join(dir, ax7TestInfectionJsona68fc9)))
}

func TestPHP_DetectInfection_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "infection", ax7TestExit05c15a7)
	got := DetectInfection(dir)
	AssertTrue(t, got)
	AssertTrue(t, fileExists(filepath.Join(dir, "vendor", "bin", "infection")))
}

func TestPHP_RunInfection_Good(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "infection", "printf infection")
	var output bytes.Buffer
	err := RunInfection(context.Background(), InfectionOptions{Dir: dir, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "infection", output.String())
}

func TestPHP_RunInfection_Bad(t *T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := RunInfection(context.Background(), InfectionOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "infection")
}

func TestPHP_RunInfection_Ugly(t *T) {
	dir := ax7PHPProject(t)
	ax7PHPExecutable(t, dir, "infection", "printf infection")
	var output bytes.Buffer
	err := RunInfection(context.Background(), InfectionOptions{Dir: dir, MinMSI: 1, MinCoveredMSI: 2, Threads: 3, Filter: "src", OnlyCovered: true, Output: &output})
	AssertNoError(t, err)
	AssertEqual(t, "infection", output.String())
}

func TestPHP_RunAudit_Good(t *T) {
	ax7PATHExecutable(t, "composer", ax7TestPrintfAdvisories19e053)
	dir := t.TempDir()
	results, err := RunAudit(context.Background(), AuditOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertNoError(t, err)
	AssertLen(t, results, 1)
	AssertEqual(t, 0, results[0].Vulnerabilities)
}

func TestPHP_RunAudit_Bad(t *T) {
	ax7PATHExecutable(t, "composer", "printf not-json; exit 1")
	dir := t.TempDir()
	results, err := RunAudit(context.Background(), AuditOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertNoError(t, err)
	AssertLen(t, results, 1)
	AssertError(t, results[0].Error)
}

func TestPHP_RunAudit_Ugly(t *T) {
	dir := t.TempDir()
	binDir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(binDir, "composer"), []byte("#!/bin/sh\nprintf '{\"advisories\":{}}'\n"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(binDir, "npm"), []byte("#!/bin/sh\nprintf '{\"metadata\":{\"vulnerabilities\":{\"total\":1}},\"vulnerabilities\":{\"left-pad\":{\"severity\":\"high\",\"via\":[]}}}'\n"), 0o755))
	t.Setenv("PATH", binDir)
	results, err := RunAudit(context.Background(), AuditOptions{Dir: dir, Output: &bytes.Buffer{}})
	AssertNoError(t, err)
	AssertLen(t, results, 2)
	AssertEqual(t, "npm", results[1].Tool)
}

func TestPHP_RunSecurityChecks_Good(t *T) {
	ax7PATHExecutable(t, "composer", ax7TestPrintfAdvisories19e053)
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_DEBUG=false\nAPP_KEY=12345678901234567890123456789012\nAPP_URL=https://example.test\n"), 0o644))
	result, err := RunSecurityChecks(context.Background(), SecurityOptions{Dir: dir})
	AssertNoError(t, err)
	AssertNotNil(t, result)
	AssertEqual(t, result.Summary.Total, result.Summary.Passed)
}

func TestPHP_RunSecurityChecks_Bad(t *T) {
	ax7PATHExecutable(t, "composer", ax7TestPrintfAdvisories19e053)
	dir := t.TempDir()
	result, err := RunSecurityChecks(context.Background(), SecurityOptions{Dir: dir, Severity: "impossible"})
	AssertError(t, err)
	AssertNil(t, result)
}

func TestPHP_RunSecurityChecks_Ugly(t *T) {
	ax7PATHExecutable(t, "composer", ax7TestPrintfAdvisories19e053)
	dir := t.TempDir()
	result, err := RunSecurityChecks(context.Background(), SecurityOptions{Dir: dir, URL: "://bad-url"})
	AssertNoError(t, err)
	AssertNotNil(t, result)
	AssertTrue(t, result.Summary.Total > 0)
}

func TestPHP_GetQAStages_Good(t *T) {
	stages := GetQAStages(QAOptions{})
	AssertEqual(t, []QAStage{QAStageQuick, QAStageStandard}, stages)
	AssertLen(t, stages, 2)
}

func TestPHP_GetQAStages_Bad(t *T) {
	stages := GetQAStages(QAOptions{Quick: true, Full: true})
	AssertEqual(t, []QAStage{QAStageQuick}, stages)
	AssertLen(t, stages, 1)
}

func TestPHP_GetQAStages_Ugly(t *T) {
	stages := GetQAStages(QAOptions{Full: true})
	AssertEqual(t, []QAStage{QAStageQuick, QAStageStandard, QAStageFull}, stages)
	AssertLen(t, stages, 3)
}

func TestPHP_GetQAChecks_Good(t *T) {
	checks := GetQAChecks(t.TempDir(), QAStageQuick)
	AssertEqual(t, []string{"audit", "fmt", "stan"}, checks)
	AssertLen(t, checks, 3)
}

func TestPHP_GetQAChecks_Bad(t *T) {
	checks := GetQAChecks(t.TempDir(), QAStage("unknown"))
	AssertNil(t, checks)
	AssertNotEqual(t, []string{}, checks)
}

func TestPHP_GetQAChecks_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestRectorPhp92466b), []byte(ax7TestPhp96d681), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestInfectionJsona68fc9), []byte(`{}`), 0o644))
	checks := GetQAChecks(dir, QAStageFull)
	AssertEqual(t, []string{"rector", "infection"}, checks)
	AssertLen(t, checks, 2)
}

func TestPHP_NewQARunner_Good(t *T) {
	runner := NewQARunner("/repo", true)
	AssertNotNil(t, runner)
	AssertEqual(t, "/repo", runner.dir)
	AssertTrue(t, runner.fix)
}

func TestPHP_NewQARunner_Bad(t *T) {
	runner := NewQARunner("", false)
	AssertNotNil(t, runner)
	AssertEqual(t, "", runner.dir)
	AssertFalse(t, runner.fix)
}

func TestPHP_NewQARunner_Ugly(t *T) {
	runner := NewQARunner(filepath.Join("a", "..", "repo"), false)
	AssertNotNil(t, runner)
	AssertContains(t, runner.dir, "repo")
	AssertFalse(t, runner.fix)
}

func TestPHP_QARunner_BuildSpecs_Good(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestPintJson7e00da), []byte(`{}`), 0o644))
	specs := NewQARunner(dir, false).BuildSpecs([]string{"audit", "fmt"})
	AssertLen(t, specs, 2)
	AssertEqual(t, "fmt", specs[1].Name)
}

func TestPHP_QARunner_BuildSpecs_Bad(t *T) {
	specs := NewQARunner(t.TempDir(), false).BuildSpecs([]string{"unknown"})
	AssertEmpty(t, specs)
	AssertNotNil(t, specs)
}

func TestPHP_QARunner_BuildSpecs_Ugly(t *T) {
	dir := ax7PHPProject(t)
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestRectorPhp92466b), []byte(ax7TestPhp96d681), 0o644))
	specs := NewQARunner(dir, true).BuildSpecs([]string{"rector"})
	AssertLen(t, specs, 1)
	AssertTrue(t, specs[0].AllowFailure)
}

func TestPHP_QACheckRunResult_GetIssueMessage_Good(t *T) {
	result := QACheckRunResult{Name: "audit"}
	message := result.GetIssueMessage()
	AssertEqual(t, "found vulnerabilities", message)
	AssertNotEqual(t, "", message)
}

func TestPHP_QACheckRunResult_GetIssueMessage_Bad(t *T) {
	result := QACheckRunResult{Name: "audit", Passed: true}
	message := result.GetIssueMessage()
	AssertEqual(t, "", message)
	AssertTrue(t, result.Passed)
}

func TestPHP_QACheckRunResult_GetIssueMessage_Ugly(t *T) {
	result := QACheckRunResult{Name: "unknown"}
	message := result.GetIssueMessage()
	AssertEqual(t, "found issues", message)
	AssertNotEqual(t, "", message)
}
