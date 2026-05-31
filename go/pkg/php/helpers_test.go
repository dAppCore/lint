package php

import (
	. "dappco.re/go"
)

// TestSecuritySeverityRank_Good_OrdersBySeverity asserts the rank ordering runs
// critical < high < medium < low < info, all reported as known.
func TestSecuritySeverityRank_Good_OrdersBySeverity(t *T) {
	critical, ok := securitySeverityRank("critical")
	AssertTrue(t, ok)
	high, ok := securitySeverityRank("high")
	AssertTrue(t, ok)
	medium, ok := securitySeverityRank("medium")
	AssertTrue(t, ok)
	low, ok := securitySeverityRank("low")
	AssertTrue(t, ok)
	info, ok := securitySeverityRank("info")
	AssertTrue(t, ok)

	AssertTrue(t, critical < high)
	AssertTrue(t, high < medium)
	AssertTrue(t, medium < low)
	AssertTrue(t, low < info)
}

// TestSecuritySeverityRank_Ugly_NormalisesCaseAndWhitespace asserts mixed case
// and surrounding whitespace still resolve to a known rank.
func TestSecuritySeverityRank_Ugly_NormalisesCaseAndWhitespace(t *T) {
	rank, ok := securitySeverityRank("  HIGH  ")
	AssertTrue(t, ok)
	AssertEqual(t, 1, rank)
}

// TestSecuritySeverityRank_Bad_UnknownReportsNotOK asserts an unrecognised
// severity is reported as unknown.
func TestSecuritySeverityRank_Bad_UnknownReportsNotOK(t *T) {
	_, ok := securitySeverityRank("apocalyptic")
	AssertFalse(t, ok)
}

// TestFilterSecurityChecks_Good_KeepsAtOrAboveMinimum keeps only checks at or
// above the requested minimum severity.
func TestFilterSecurityChecks_Good_KeepsAtOrAboveMinimum(t *T) {
	checks := []SecurityCheck{
		{Name: "a", Severity: "critical"},
		{Name: "b", Severity: "medium"},
		{Name: "c", Severity: "low"},
	}
	result := filterSecurityChecks(checks, "medium")
	RequireResultOK(t, result)
	kept := result.Value.([]SecurityCheck)
	RequireLen(t, kept, 2)
}

// TestFilterSecurityChecks_Bad_InvalidMinimumFails fails when the minimum
// severity is not recognised.
func TestFilterSecurityChecks_Bad_InvalidMinimumFails(t *T) {
	result := filterSecurityChecks([]SecurityCheck{{Severity: "high"}}, "nonsense")
	AssertFalse(t, result.OK)
}

// TestFilterSecurityChecks_Ugly_EmptyMinimumPassesThrough returns all checks
// unfiltered when no minimum is supplied.
func TestFilterSecurityChecks_Ugly_EmptyMinimumPassesThrough(t *T) {
	checks := []SecurityCheck{{Severity: "low"}}
	result := filterSecurityChecks(checks, "")
	RequireResultOK(t, result)
	RequireLen(t, result.Value.([]SecurityCheck), 1)
}

// TestPhpCommandExitCode_GoodBadUgly maps nil, ExitCode-bearing and plain
// errors to 0, the carried code and -1 respectively.
func TestPhpCommandExitCode_GoodBadUgly(t *T) {
	AssertEqual(t, 0, phpCommandExitCode(nil))
	AssertEqual(t, 3, phpCommandExitCode(exitCodeError(3)))
	AssertEqual(t, -1, phpCommandExitCode(Errorf("plain error")))
}

// exitCodeError is a test error carrying an ExitCode method.
type exitCodeError int

func (e exitCodeError) Error() string { return "exit" }
func (e exitCodeError) ExitCode() int { return int(e) }

// TestFindPHPExecutable_Good_FindsOnPath resolves an executable placed on a
// scratch PATH.
func TestFindPHPExecutable_Good_FindsOnPath(t *T) {
	binDir := t.TempDir()
	toolPath := PathJoin(binDir, "phpfake")
	RequireResultOK(t, WriteFile(toolPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", binDir)

	found := findPHPExecutable("phpfake")
	RequireResultOK(t, found)
	AssertEqual(t, toolPath, found.Value.(string))
}

// TestFindPHPExecutable_Bad_EmptyNameFails rejects an empty executable name.
func TestFindPHPExecutable_Bad_EmptyNameFails(t *T) {
	result := findPHPExecutable("")
	AssertFalse(t, result.OK)
}

// TestFindPHPExecutable_Ugly_MissingNameFails fails when the name is absent
// from PATH.
func TestFindPHPExecutable_Ugly_MissingNameFails(t *T) {
	t.Setenv("PATH", t.TempDir())
	result := findPHPExecutable("definitely-not-a-real-tool-xyz")
	AssertFalse(t, result.OK)
}

// TestVendorBinOrDefault_Good_PrefersVendorBin returns the vendor/bin path when
// the binary exists there.
func TestVendorBinOrDefault_Good_PrefersVendorBin(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin", "pint")
	RequireResultOK(t, MkdirAll(PathDir(vendorBin), 0o755))
	RequireResultOK(t, WriteFile(vendorBin, []byte("#!/bin/sh\n"), 0o755))
	AssertEqual(t, vendorBin, vendorBinOrDefault(dir, "pint"))
}

// TestVendorBinOrDefault_Ugly_FallsBackToName returns the bare name when no
// vendor binary exists.
func TestVendorBinOrDefault_Ugly_FallsBackToName(t *T) {
	AssertEqual(t, "pint", vendorBinOrDefault(t.TempDir(), "pint"))
}

// TestJunitReportPath_GoodBadUgly returns the explicit path when set and the
// default file name otherwise.
func TestJunitReportPath_GoodBadUgly(t *T) {
	AssertEqual(t, "out.xml", junitReportPath(TestOptions{JUnitPath: "out.xml"}))
	AssertEqual(t, "test-results.xml", junitReportPath(TestOptions{}))
}

// TestEmitJUnitReport_Good_WritesAndTerminates copies the report to the writer
// and appends a trailing newline when missing.
func TestEmitJUnitReport_Good_WritesAndTerminates(t *T) {
	dir := t.TempDir()
	reportPath := PathJoin(dir, "report.xml")
	RequireResultOK(t, WriteFile(reportPath, []byte("<testsuite/>"), 0o644))

	buffer := NewBuffer()
	RequireResultOK(t, emitJUnitReport(buffer, reportPath))
	AssertEqual(t, "<testsuite/>\n", buffer.String())
}

// TestEmitJUnitReport_Ugly_AlreadyTerminatedNotDoubled does not append a second
// newline when the report already ends in one.
func TestEmitJUnitReport_Ugly_AlreadyTerminatedNotDoubled(t *T) {
	dir := t.TempDir()
	reportPath := PathJoin(dir, "report.xml")
	RequireResultOK(t, WriteFile(reportPath, []byte("<testsuite/>\n"), 0o644))

	buffer := NewBuffer()
	RequireResultOK(t, emitJUnitReport(buffer, reportPath))
	AssertEqual(t, "<testsuite/>\n", buffer.String())
}

// TestEmitJUnitReport_Bad_MissingFileFails fails when the report file is
// absent.
func TestEmitJUnitReport_Bad_MissingFileFails(t *T) {
	result := emitJUnitReport(NewBuffer(), PathJoin(t.TempDir(), "absent.xml"))
	AssertFalse(t, result.OK)
}
