package qa

import (
	. "dappco.re/go"

	"dappco.re/go/lint/pkg/php"
)

// TestGetSeverityStyle_GoodBadUgly maps known severities to their dedicated
// styles and falls back to the dim style for anything unrecognised.
func TestGetSeverityStyle_GoodBadUgly(t *T) {
	AssertTrue(t, getSeverityStyle("critical") == criticalStyle)
	AssertTrue(t, getSeverityStyle("HIGH") == highStyle)
	AssertTrue(t, getSeverityStyle("medium") == mediumStyle)
	AssertTrue(t, getSeverityStyle("low") == lowStyle)
	AssertTrue(t, getSeverityStyle("unknown") == dimStyle)
}

// TestPrintPHPAuditResult_Good_NoVulns prints a clean line and reports no
// vulnerabilities for a zero-count audit result.
func TestPrintPHPAuditResult_Good_NoVulns(t *T) {
	var hasVulns bool
	output := captureStdout(t, func() {
		hasVulns = printPHPAuditResult(php.AuditResult{Tool: "composer", Vulnerabilities: 0})
	})
	AssertFalse(t, hasVulns)
	AssertContains(t, output, "no vulnerabilities found")
}

// TestPrintPHPAuditResult_Bad_VulnsListed prints each advisory and reports
// vulnerabilities present.
func TestPrintPHPAuditResult_Bad_VulnsListed(t *T) {
	var hasVulns bool
	output := captureStdout(t, func() {
		hasVulns = printPHPAuditResult(php.AuditResult{
			Tool:            "composer",
			Vulnerabilities: 1,
			Advisories: []php.AuditAdvisory{
				{Package: "acme/widget", Title: "RCE in widget"},
			},
		})
	})
	AssertTrue(t, hasVulns)
	AssertContains(t, output, "1 vulnerabilities found")
	AssertContains(t, output, "acme/widget")
	AssertContains(t, output, "RCE in widget")
}

// TestPrintPHPAuditResult_Ugly_ToolErrorWarns prints a warning and does not
// flag vulnerabilities when the audit tool itself errored.
func TestPrintPHPAuditResult_Ugly_ToolErrorWarns(t *T) {
	var hasVulns bool
	output := captureStdout(t, func() {
		hasVulns = printPHPAuditResult(php.AuditResult{Tool: "npm", Error: Errorf("tool missing")})
	})
	AssertFalse(t, hasVulns)
	AssertContains(t, output, "npm")
	AssertContains(t, output, "tool missing")
}

// TestPrintPHPAuditText_Bad_FailsWhenVulnsPresent aggregates results and fails
// when any tool reported vulnerabilities.
func TestPrintPHPAuditText_Bad_FailsWhenVulnsPresent(t *T) {
	captureStdout(t, func() {
		result := printPHPAuditText([]php.AuditResult{
			{Tool: "composer", Vulnerabilities: 0},
			{Tool: "npm", Vulnerabilities: 2, Advisories: []php.AuditAdvisory{{Package: "p", Title: "t"}}},
		})
		RequireResultError(t, result)
	})
}

// TestPrintPHPAuditText_Good_PassesWhenClean returns OK when no tool reported
// vulnerabilities.
func TestPrintPHPAuditText_Good_PassesWhenClean(t *T) {
	captureStdout(t, func() {
		result := printPHPAuditText([]php.AuditResult{{Tool: "composer", Vulnerabilities: 0}})
		RequireResultOK(t, result)
	})
}

// TestPrintPHPSecurityCheck_Good_PassedRendersName prints just the check name
// for a passing check.
func TestPrintPHPSecurityCheck_Good_PassedRendersName(t *T) {
	output := captureStdout(t, func() {
		printPHPSecurityCheck(php.SecurityCheck{Name: "csrf", Passed: true})
	})
	AssertContains(t, output, "csrf")
}

// TestPrintPHPSecurityCheck_Bad_FailedRendersSeverityMessageFix prints the
// severity label, message and fix for a failing check.
func TestPrintPHPSecurityCheck_Bad_FailedRendersSeverityMessageFix(t *T) {
	output := captureStdout(t, func() {
		printPHPSecurityCheck(php.SecurityCheck{
			Name:     "debug-mode",
			Severity: "high",
			Passed:   false,
			Message:  "APP_DEBUG is true",
			Fix:      "set APP_DEBUG=false",
		})
	})
	AssertContains(t, output, "debug-mode")
	AssertContains(t, output, "high")
	AssertContains(t, output, "APP_DEBUG is true")
	AssertContains(t, output, "set APP_DEBUG=false")
}

// TestPrintPHPSecurityText_Ugly_RendersChecksAndSummary prints each check and
// the trailing pass/total summary.
func TestPrintPHPSecurityText_Ugly_RendersChecksAndSummary(t *T) {
	output := captureStdout(t, func() {
		printPHPSecurityText(&php.SecurityResult{
			Checks: []php.SecurityCheck{
				{Name: "csrf", Passed: true},
				{Name: "debug", Severity: "critical", Passed: false, Message: "on"},
			},
			Summary: php.SecuritySummary{Total: 2, Passed: 1, Critical: 1},
		})
	})
	AssertContains(t, output, "csrf")
	AssertContains(t, output, "debug")
	AssertContains(t, output, "1/2 checks passed")
	AssertContains(t, output, "1 critical")
}
