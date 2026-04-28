package qa

import (
	. "dappco.re/go"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"dappco.re/go/cli/pkg/cli"
)

func TestPHPStanJSONOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"phpstan\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPStanCommand(parent)
	command := findSubcommand(t, parent, "stan")
	RequireNoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	AssertEqual(t, "{\"tool\":\"phpstan\",\"status\":\"ok\"}\n", output)
	AssertNotContains(t, output, "Static analysis passed")
	AssertNotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmJSONOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"psalm\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPPsalmCommand(parent)
	command := findSubcommand(t, parent, "psalm")
	RequireNoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	AssertEqual(t, "{\"tool\":\"psalm\",\"status\":\"ok\"}\n", output)
	AssertNotContains(t, output, "Psalm analysis passed")
	AssertNotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPStanSARIFOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPStanCommand(parent)
	command := findSubcommand(t, parent, "stan")
	RequireNoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	AssertEqual(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	AssertNotContains(t, output, "Static analysis passed")
	AssertNotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmSARIFOutput_DoesNotAppendSuccessBanner(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPPsalmCommand(parent)
	command := findSubcommand(t, parent, "psalm")
	RequireNoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	AssertEqual(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	AssertNotContains(t, output, "Psalm analysis passed")
	AssertNotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPSecurityJSONOutput_UsesMachineFriendlyKeys(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeTestFile(t, filepath.Join(dir, ".env"), "APP_DEBUG=true\nAPP_KEY=short\nAPP_URL=http://example.com\n")
	writeExecutable(t, filepath.Join(dir, "bin", "composer"), "#!/bin/sh\nprintf '%s\\n' '{\"advisories\":{}}'\n")

	restoreWorkingDir(t, dir)
	prependPath(t, filepath.Join(dir, "bin"))
	resetPHPSecurityFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPSecurityCommand(parent)
	command := findSubcommand(t, parent, "security")
	RequireNoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		RequireError(t, command.RunE(command, nil))
	})

	AssertContains(t, output, "\"checks\"")
	AssertContains(t, output, "\"summary\"")
	AssertContains(t, output, "\"app_key_set\"")
	AssertNotContains(t, output, "\"Checks\"")
	AssertNotContains(t, output, "Security Checks")
}

func TestPHPSecuritySARIFOutput_IsStructuredAndChromeFree(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeTestFile(t, filepath.Join(dir, ".env"), "APP_DEBUG=true\nAPP_KEY=short\nAPP_URL=http://example.com\n")
	writeExecutable(t, filepath.Join(dir, "bin", "composer"), "#!/bin/sh\nprintf '%s\\n' '{\"advisories\":{}}'\n")

	restoreWorkingDir(t, dir)
	prependPath(t, filepath.Join(dir, "bin"))
	resetPHPSecurityFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPSecurityCommand(parent)
	command := findSubcommand(t, parent, "security")
	RequireNoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		RequireError(t, command.RunE(command, nil))
	})

	var payload map[string]any
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
	AssertEqual(t, "2.1.0", payload["version"])
	AssertContains(t, output, "\"ruleId\": \"app_key_set\"")
	AssertNotContains(t, output, "Security Checks")
	AssertNotContains(t, output, "Summary:")
}

func TestPHPSecurityJSONOutput_RespectsSeverityFilter(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeTestFile(t, filepath.Join(dir, ".env"), "APP_DEBUG=true\nAPP_KEY=short\nAPP_URL=http://example.com\n")
	writeExecutable(t, filepath.Join(dir, "bin", "composer"), "#!/bin/sh\nprintf '%s\\n' '{\"advisories\":{}}'\n")

	restoreWorkingDir(t, dir)
	prependPath(t, filepath.Join(dir, "bin"))
	resetPHPSecurityFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPSecurityCommand(parent)
	command := findSubcommand(t, parent, "security")
	RequireNoError(t, command.Flags().Set("json", "true"))
	RequireNoError(t, command.Flags().Set("severity", "critical"))

	output := captureStdout(t, func() {
		RequireError(t, command.RunE(command, nil))
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
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
	AssertEqual(t, 3, payload.Summary.Total)
	AssertEqual(t, 1, payload.Summary.Passed)
	AssertEqual(t, 2, payload.Summary.Critical)
	AssertEqual(t, 0, payload.Summary.High)
	RequireLen(t, payload.Checks, 3)
	AssertNotContains(t, output, "https_enforced")
}

func TestPHPAuditJSONOutput_UsesLowerCaseAdvisoryKeys(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "composer"), `#!/bin/sh
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

	parent := &cli.Command{Use: "qa"}
	addPHPAuditCommand(parent)
	command := findSubcommand(t, parent, "audit")
	RequireNoError(t, command.Flags().Set("json", "true"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	RequireError(t, runErr)

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
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
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
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpunit"), "#!/bin/sh\njunit=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--log-junit\" ]; then\n    shift\n    junit=\"$1\"\n  fi\n  shift\ndone\nprintf '%s\\n' 'human output should be suppressed'\nprintf '%s' '<testsuite tests=\"1\"></testsuite>' > \"$junit\"\n")

	restoreWorkingDir(t, dir)
	resetPHPTestFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPTestCommand(parent)
	command := findSubcommand(t, parent, "test")
	RequireNoError(t, command.Flags().Set("junit", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	AssertEqual(t, "<testsuite tests=\"1\"></testsuite>\n", output)
	AssertNotContains(t, output, "human output should be suppressed")
	AssertNotContains(t, output, "PHP Tests")
	AssertNotContains(t, output, "All tests passed")
}

func writeTestFile(t *T, path string, content string) {
	t.Helper()
	RequireNoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	RequireNoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeExecutable(t *T, path string, content string) {
	t.Helper()
	RequireNoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	RequireNoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func restoreWorkingDir(t *T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	RequireNoError(t, err)
	RequireNoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		RequireNoError(t, os.Chdir(wd))
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

func findSubcommand(t *T, parent *cli.Command, name string) *cli.Command {
	t.Helper()
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func captureStdout(t *T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	RequireNoError(t, err)
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()
	defer func() {
		RequireNoError(t, reader.Close())
	}()

	fn()

	RequireNoError(t, writer.Close())

	output, err := io.ReadAll(reader)
	RequireNoError(t, err)
	return string(output)
}

func prependPath(t *T, dir string) {
	t.Helper()
	oldPath := os.Getenv("PATH")
	RequireNoError(t, os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath))
	t.Cleanup(func() {
		RequireNoError(t, os.Setenv("PATH", oldPath))
	})
}
