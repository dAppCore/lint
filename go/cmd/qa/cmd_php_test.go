package qa

import (
	. "dappco.re/go"

	"dappco.re/go/cli/pkg/cli"
)

const (
	cmdPhpTestAppDebugTrueAppKeyShortAppUrlHttbbebce = "APP_DEBUG=true\nAPP_KEY=short\nAPP_URL=http://example.com\n"
	cmdPhpTestBinShPrintfSNAdvisories73fca5          = "#!/bin/sh\nprintf '%s\\n' '{\"advisories\":{}}'\n"
	cmdPhpTestComposerJson089de2                     = "composer.json"
)

func TestPHPStanJSONOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"phpstan\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	phpStanJSON = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runPHPStan())
	})

	AssertEqual(t, "{\"tool\":\"phpstan\",\"status\":\"ok\"}\n", output)
	AssertNotContains(t, output, "Static analysis passed")
	AssertNotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmJSONOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"psalm\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	phpPsalmJSON = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runPHPPsalm())
	})

	AssertEqual(t, "{\"tool\":\"psalm\",\"status\":\"ok\"}\n", output)
	AssertNotContains(t, output, "Psalm analysis passed")
	AssertNotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPStanSARIFOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	phpStanSARIF = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runPHPStan())
	})

	AssertEqual(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	AssertNotContains(t, output, "Static analysis passed")
	AssertNotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmSARIFOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	phpPsalmSARIF = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runPHPPsalm())
	})

	AssertEqual(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	AssertNotContains(t, output, "Psalm analysis passed")
	AssertNotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPSecurityJSONOutput_UsesMachineFriendlyKeys(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeTestFile(t, PathJoin(dir, ".env"), cmdPhpTestAppDebugTrueAppKeyShortAppUrlHttbbebce)
	writeExecutable(t, PathJoin(dir, "bin", "composer"), cmdPhpTestBinShPrintfSNAdvisories73fca5)

	restoreWorkingDir(t, dir)
	prependPath(t, PathJoin(dir, "bin"))
	resetPHPSecurityFlags(t)

	phpSecurityJSON = true

	output := captureStdout(t, func() {
		RequireResultError(t, runPHPSecurity())
	})

	AssertContains(t, output, "\"checks\"")
	AssertContains(t, output, "\"summary\"")
	AssertContains(t, output, "\"app_key_set\"")
	AssertNotContains(t, output, "\"Checks\"")
	AssertNotContains(t, output, "Security Checks")
}

func TestPHPSecuritySARIFOutput_IsStructuredAndChromeFree(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeTestFile(t, PathJoin(dir, ".env"), cmdPhpTestAppDebugTrueAppKeyShortAppUrlHttbbebce)
	writeExecutable(t, PathJoin(dir, "bin", "composer"), cmdPhpTestBinShPrintfSNAdvisories73fca5)

	restoreWorkingDir(t, dir)
	prependPath(t, PathJoin(dir, "bin"))
	resetPHPSecurityFlags(t)

	phpSecuritySARIF = true

	output := captureStdout(t, func() {
		RequireResultError(t, runPHPSecurity())
	})

	var payload map[string]any
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertEqual(t, "2.1.0", payload["version"])
	AssertContains(t, output, "\"ruleId\": \"app_key_set\"")
	AssertNotContains(t, output, "Security Checks")
	AssertNotContains(t, output, "Summary:")
}

func TestPHPSecurityJSONOutput_RespectsSeverityFilter(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeTestFile(t, PathJoin(dir, ".env"), cmdPhpTestAppDebugTrueAppKeyShortAppUrlHttbbebce)
	writeExecutable(t, PathJoin(dir, "bin", "composer"), cmdPhpTestBinShPrintfSNAdvisories73fca5)

	restoreWorkingDir(t, dir)
	prependPath(t, PathJoin(dir, "bin"))
	resetPHPSecurityFlags(t)

	phpSecurityJSON = true
	phpSecuritySeverity = "critical"

	output := captureStdout(t, func() {
		RequireResultError(t, runPHPSecurity())
	})

	var payload struct {
		Checks []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
		} `json:"checks"`
		Summary struct {
			Total    int `json:"total"`
			Passed   int `json:"passed"`
			Critical int `json:"critical"`
			High     int `json:"high"`
		} `json:"summary"`
	}
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertEqual(t, 3, payload.Summary.Total)
	AssertEqual(t, 1, payload.Summary.Passed)
	AssertEqual(t, 2, payload.Summary.Critical)
	AssertEqual(t, 0, payload.Summary.High)
	RequireLen(t, payload.Checks, 3)
	AssertNotContains(t, output, "https_enforced")
}

func TestPHPAuditJSONOutput_UsesLowerCaseAdvisoryKeys(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "composer"), `#!/bin/sh
cat <<'JSON'
{
  "advisories": {
    "vendor/package-a": [
      {
        "title": "Remote Code Execution",
        "link": "https://example.com/advisory/1",
        "cve": "CVE-2025-1234",
        "affectedVersions": ">=1.0,<1.5"
      }
    ]
  }
}
JSON
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetPHPAuditFlags(t)

	phpAuditJSON = true

	var runResult Result
	output := captureStdout(t, func() {
		runResult = runPHPAudit()
	})

	RequireResultError(t, runResult)

	var payload struct {
		Results []struct {
			Tool       string `json:"tool"`
			Advisories []struct {
				Package string `json:"package"`
			} `json:"advisories"`
		} `json:"results"`
		HasVulnerabilities bool `json:"has_vulnerabilities"`
		Vulnerabilities    int  `json:"vulnerabilities"`
	}
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	RequireLen(t, payload.Results, 1)
	AssertEqual(t, "composer", payload.Results[0].Tool)
	RequireLen(t, payload.Results[0].Advisories, 1)
	AssertEqual(t, "vendor/package-a", payload.Results[0].Advisories[0].Package)
	AssertTrue(t, payload.HasVulnerabilities)
	AssertEqual(t, 1, payload.Vulnerabilities)
	AssertNotContains(t, output, "\"Package\"")
	AssertNotContains(t, output, "Dependency Audit")
}

func TestPHPTestJUnitOutput_PrintsOnlyXML(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdPhpTestComposerJson089de2), "{}")
	writeExecutable(t, PathJoin(dir, "vendor", "bin", "phpunit"), "#!/bin/sh\njunit=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--log-junit\" ]; then\n    shift\n    junit=\"$1\"\n  fi\n  shift\ndone\nprintf '%s\\n' 'human output should be suppressed'\nprintf '%s' '<testsuite tests=\"1\"></testsuite>' > \"$junit\"\n")

	restoreWorkingDir(t, dir)
	resetPHPTestFlags(t)

	phpTestJUnit = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runPHPTest())
	})

	AssertEqual(t, "<testsuite tests=\"1\"></testsuite>\n", output)
	AssertNotContains(t, output, "human output should be suppressed")
	AssertNotContains(t, output, "PHP Tests")
	AssertNotContains(t, output, "All tests passed")
}

func writeTestFile(t *T, path string, content string) {
	t.Helper()
	RequireResultOK(t, MkdirAll(PathDir(path), 0o755))
	RequireResultOK(t, WriteFile(path, []byte(content), 0o644))
}

func writeExecutable(t *T, path string, content string) {
	t.Helper()
	RequireResultOK(t, MkdirAll(PathDir(path), 0o755))
	RequireResultOK(t, WriteFile(path, []byte(content), 0o755))
}

func restoreWorkingDir(t *T, dir string) {
	t.Helper()
	wd := Getwd()
	RequireResultOK(t, wd)
	RequireResultOK(t, Chdir(dir))
	t.Cleanup(func() {
		RequireResultOK(t, Chdir(wd.Value.(string)))
	})
}

func resetPHPStanFlags(t *T) {
	t.Helper()
	oldLevel := phpStanLevel
	oldMemory := phpStanMemory
	oldJSON := phpStanJSON
	oldSARIF := phpStanSARIF
	phpStanLevel = 0
	phpStanMemory = ""
	phpStanJSON = false
	phpStanSARIF = false
	t.Cleanup(func() {
		phpStanLevel = oldLevel
		phpStanMemory = oldMemory
		phpStanJSON = oldJSON
		phpStanSARIF = oldSARIF
	})
}

func resetPHPPsalmFlags(t *T) {
	t.Helper()
	oldLevel := phpPsalmLevel
	oldFix := phpPsalmFix
	oldBaseline := phpPsalmBaseline
	oldShowInfo := phpPsalmShowInfo
	oldJSON := phpPsalmJSON
	oldSARIF := phpPsalmSARIF
	phpPsalmLevel = 0
	phpPsalmFix = false
	phpPsalmBaseline = false
	phpPsalmShowInfo = false
	phpPsalmJSON = false
	phpPsalmSARIF = false
	t.Cleanup(func() {
		phpPsalmLevel = oldLevel
		phpPsalmFix = oldFix
		phpPsalmBaseline = oldBaseline
		phpPsalmShowInfo = oldShowInfo
		phpPsalmJSON = oldJSON
		phpPsalmSARIF = oldSARIF
	})
}

func resetPHPSecurityFlags(t *T) {
	t.Helper()
	oldSeverity := phpSecuritySeverity
	oldJSON := phpSecurityJSON
	oldSARIF := phpSecuritySARIF
	oldURL := phpSecurityURL
	phpSecuritySeverity = ""
	phpSecurityJSON = false
	phpSecuritySARIF = false
	phpSecurityURL = ""
	t.Cleanup(func() {
		phpSecuritySeverity = oldSeverity
		phpSecurityJSON = oldJSON
		phpSecuritySARIF = oldSARIF
		phpSecurityURL = oldURL
	})
}

func resetPHPAuditFlags(t *T) {
	t.Helper()
	oldJSON := phpAuditJSON
	oldFix := phpAuditFix
	phpAuditJSON = false
	phpAuditFix = false
	t.Cleanup(func() {
		phpAuditJSON = oldJSON
		phpAuditFix = oldFix
	})
}

func resetPHPTestFlags(t *T) {
	t.Helper()
	oldParallel := phpTestParallel
	oldCoverage := phpTestCoverage
	oldFilter := phpTestFilter
	oldGroup := phpTestGroup
	oldJUnit := phpTestJUnit
	phpTestParallel = false
	phpTestCoverage = false
	phpTestFilter = ""
	phpTestGroup = ""
	phpTestJUnit = false
	t.Cleanup(func() {
		phpTestParallel = oldParallel
		phpTestCoverage = oldCoverage
		phpTestFilter = oldFilter
		phpTestGroup = oldGroup
		phpTestJUnit = oldJUnit
	})
}

func captureStdout(t *T, fn func()) string {
	t.Helper()
	out := NewBuilder()
	cli.SetStdout(out)
	defer cli.SetStdout(nil)
	fn()
	return out.String()
}

func prependPath(t *T, dir string) {
	t.Helper()
	oldPath := Getenv("PATH")
	RequireResultOK(t, Setenv("PATH", dir+string(PathListSeparator)+oldPath))
	t.Cleanup(func() {
		RequireResultOK(t, Setenv("PATH", oldPath))
	})
}
