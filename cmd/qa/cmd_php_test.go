package qa

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"dappco.re/go/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPHPStanJSONOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"phpstan\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPStanCommand(parent)
	command := findSubcommand(t, parent, "stan")
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"tool\":\"phpstan\",\"status\":\"ok\"}\n", output)
	assert.NotContains(t, output, "Static analysis passed")
	assert.NotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmJSONOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"psalm\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPPsalmCommand(parent)
	command := findSubcommand(t, parent, "psalm")
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"tool\":\"psalm\",\"status\":\"ok\"}\n", output)
	assert.NotContains(t, output, "Psalm analysis passed")
	assert.NotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPStanSARIFOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPStanCommand(parent)
	command := findSubcommand(t, parent, "stan")
	require.NoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	assert.NotContains(t, output, "Static analysis passed")
	assert.NotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmSARIFOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"version\":\"2.1.0\",\"runs\":[]}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPPsalmCommand(parent)
	command := findSubcommand(t, parent, "psalm")
	require.NoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n", output)
	assert.NotContains(t, output, "Psalm analysis passed")
	assert.NotContains(t, output, "PHP Psalm Analysis")
}

func TestPHPSecurityJSONOutput_UsesMachineFriendlyKeys(t *testing.T) {
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
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.Error(t, command.RunE(command, nil))
	})

	assert.Contains(t, output, "\"checks\"")
	assert.Contains(t, output, "\"summary\"")
	assert.Contains(t, output, "\"app_key_set\"")
	assert.NotContains(t, output, "\"Checks\"")
	assert.NotContains(t, output, "Security Checks")
}

func TestPHPSecuritySARIFOutput_IsStructuredAndChromeFree(t *testing.T) {
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
	require.NoError(t, command.Flags().Set("sarif", "true"))

	output := captureStdout(t, func() {
		require.Error(t, command.RunE(command, nil))
	})

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Equal(t, "2.1.0", payload["version"])
	assert.Contains(t, output, "\"ruleId\": \"app_key_set\"")
	assert.NotContains(t, output, "Security Checks")
	assert.NotContains(t, output, "Summary:")
}

func TestPHPSecurityJSONOutput_RespectsSeverityFilter(t *testing.T) {
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
	require.NoError(t, command.Flags().Set("json", "true"))
	require.NoError(t, command.Flags().Set("severity", "critical"))

	output := captureStdout(t, func() {
		require.Error(t, command.RunE(command, nil))
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
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Equal(t, 3, payload.Summary.Total)
	assert.Equal(t, 1, payload.Summary.Passed)
	assert.Equal(t, 2, payload.Summary.Critical)
	assert.Zero(t, payload.Summary.High)
	require.Len(t, payload.Checks, 3)
	assert.NotContains(t, output, "https_enforced")
}

func TestPHPAuditJSONOutput_UsesLowerCaseAdvisoryKeys(t *testing.T) {
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
	require.NoError(t, command.Flags().Set("json", "true"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	require.Error(t, runErr)

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
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	require.Len(t, payload.Results, 1)
	assert.Equal(t, "composer", payload.Results[0].Tool)
	require.Len(t, payload.Results[0].Advisories, 1)
	assert.Equal(t, "vendor/package-a", payload.Results[0].Advisories[0].Package)
	assert.True(t, payload.HasVulnerabilities)
	assert.Equal(t, 1, payload.Vulnerabilities)
	assert.NotContains(t, output, "\"Package\"")
	assert.NotContains(t, output, "Dependency Audit")
}

func TestPHPTestJUnitOutput_PrintsOnlyXML(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpunit"), "#!/bin/sh\njunit=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--log-junit\" ]; then\n    shift\n    junit=\"$1\"\n  fi\n  shift\ndone\nprintf '%s\\n' 'human output should be suppressed'\nprintf '%s' '<testsuite tests=\"1\"></testsuite>' > \"$junit\"\n")

	restoreWorkingDir(t, dir)
	resetPHPTestFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPTestCommand(parent)
	command := findSubcommand(t, parent, "test")
	require.NoError(t, command.Flags().Set("junit", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "<testsuite tests=\"1\"></testsuite>\n", output)
	assert.NotContains(t, output, "human output should be suppressed")
	assert.NotContains(t, output, "PHP Tests")
	assert.NotContains(t, output, "All tests passed")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func restoreWorkingDir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})
}

func resetPHPStanFlags(t *testing.T) {
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

func resetPHPPsalmFlags(t *testing.T) {
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

func resetPHPSecurityFlags(t *testing.T) {
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

func resetPHPAuditFlags(t *testing.T) {
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

func resetPHPTestFlags(t *testing.T) {
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

func findSubcommand(t *testing.T, parent *cli.Command, name string) *cli.Command {
	t.Helper()
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()
	defer func() {
		require.NoError(t, reader.Close())
	}()

	fn()

	require.NoError(t, writer.Close())

	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(output)
}

func prependPath(t *testing.T, dir string) {
	t.Helper()
	oldPath := os.Getenv("PATH")
	require.NoError(t, os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath))
	t.Cleanup(func() {
		require.NoError(t, os.Setenv("PATH", oldPath))
	})
}
