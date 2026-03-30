package php

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditResult_Fields(t *testing.T) {
	result := AuditResult{
		Tool:            "composer",
		Vulnerabilities: 2,
		Advisories: []AuditAdvisory{
			{Package: "vendor/pkg", Severity: "high", Title: "RCE", URL: "https://example.com/1", Identifiers: []string{"CVE-2025-0001"}},
			{Package: "vendor/other", Severity: "medium", Title: "XSS", URL: "https://example.com/2", Identifiers: []string{"CVE-2025-0002"}},
		},
	}

	assert.Equal(t, "composer", result.Tool)
	assert.Equal(t, 2, result.Vulnerabilities)
	assert.Len(t, result.Advisories, 2)
	assert.Equal(t, "vendor/pkg", result.Advisories[0].Package)
	assert.Equal(t, "high", result.Advisories[0].Severity)
	assert.Equal(t, "RCE", result.Advisories[0].Title)
	assert.Equal(t, "https://example.com/1", result.Advisories[0].URL)
	assert.Equal(t, []string{"CVE-2025-0001"}, result.Advisories[0].Identifiers)
}

func TestAuditAdvisory_Fields(t *testing.T) {
	adv := AuditAdvisory{
		Package:     "laravel/framework",
		Severity:    "critical",
		Title:       "SQL Injection",
		URL:         "https://example.com/advisory",
		Identifiers: []string{"CVE-2025-9999", "GHSA-xxxx"},
	}

	assert.Equal(t, "laravel/framework", adv.Package)
	assert.Equal(t, "critical", adv.Severity)
	assert.Equal(t, "SQL Injection", adv.Title)
	assert.Equal(t, "https://example.com/advisory", adv.URL)
	assert.Equal(t, []string{"CVE-2025-9999", "GHSA-xxxx"}, adv.Identifiers)
}

func TestSortAuditAdvisories_Good(t *testing.T) {
	advisories := []AuditAdvisory{
		{Package: "vendor/package-b", Title: "Zulu"},
		{Package: "vendor/package-a", Title: "Beta"},
		{Package: "vendor/package-b", Title: "Alpha"},
	}

	sortAuditAdvisories(advisories)

	require.Len(t, advisories, 3)
	assert.Equal(t, "vendor/package-a", advisories[0].Package)
	assert.Equal(t, "Beta", advisories[0].Title)
	assert.Equal(t, "vendor/package-b", advisories[1].Package)
	assert.Equal(t, "Alpha", advisories[1].Title)
	assert.Equal(t, "vendor/package-b", advisories[2].Package)
	assert.Equal(t, "Zulu", advisories[2].Title)
}

func TestRunComposerAudit_ParsesJSON(t *testing.T) {
	// Test the JSON parsing of composer audit output by verifying
	// the struct can be populated from JSON matching composer's format.
	composerOutput := `{
		"advisories": {
			"vendor/package-a": [
				{
					"title": "Remote Code Execution",
					"link": "https://example.com/advisory/1",
					"cve": "CVE-2025-1234",
					"affectedVersions": ">=1.0,<1.5"
				}
			],
			"vendor/package-b": [
				{
					"title": "Cross-Site Scripting",
					"link": "https://example.com/advisory/2",
					"cve": "CVE-2025-5678",
					"affectedVersions": ">=2.0,<2.3"
				},
				{
					"title": "Open Redirect",
					"link": "https://example.com/advisory/3",
					"cve": "CVE-2025-9012",
					"affectedVersions": ">=2.0,<2.1"
				}
			]
		}
	}`

	var auditData struct {
		Advisories map[string][]struct {
			Title          string `json:"title"`
			Link           string `json:"link"`
			CVE            string `json:"cve"`
			AffectedRanges string `json:"affectedVersions"`
		} `json:"advisories"`
	}

	err := json.Unmarshal([]byte(composerOutput), &auditData)
	require.NoError(t, err)

	// Simulate the same parsing logic as runComposerAudit
	result := AuditResult{Tool: "composer"}
	for pkg, advisories := range auditData.Advisories {
		for _, adv := range advisories {
			result.Advisories = append(result.Advisories, AuditAdvisory{
				Package:     pkg,
				Title:       adv.Title,
				URL:         adv.Link,
				Identifiers: []string{adv.CVE},
			})
		}
	}
	sortAuditAdvisories(result.Advisories)
	result.Vulnerabilities = len(result.Advisories)

	assert.Equal(t, "composer", result.Tool)
	assert.Equal(t, 3, result.Vulnerabilities)
	assert.Len(t, result.Advisories, 3)
	assert.Equal(t, "vendor/package-a", result.Advisories[0].Package)
	assert.Equal(t, "Remote Code Execution", result.Advisories[0].Title)
	assert.Equal(t, "https://example.com/advisory/1", result.Advisories[0].URL)
	assert.Equal(t, []string{"CVE-2025-1234"}, result.Advisories[0].Identifiers)
	assert.Equal(t, "vendor/package-b", result.Advisories[1].Package)
	assert.Equal(t, "Cross-Site Scripting", result.Advisories[1].Title)
	assert.Equal(t, "vendor/package-b", result.Advisories[2].Package)
	assert.Equal(t, "Open Redirect", result.Advisories[2].Title)
}

func TestNpmAuditJSON_ParsesCorrectly(t *testing.T) {
	// Test npm audit JSON parsing logic
	npmOutput := `{
		"metadata": {
			"vulnerabilities": {
				"total": 2
			}
		},
		"vulnerabilities": {
			"lodash": {
				"severity": "high",
				"via": ["prototype pollution"]
			},
			"minimist": {
				"severity": "low",
				"via": ["prototype pollution"]
			}
		}
	}`

	var auditData struct {
		Metadata struct {
			Vulnerabilities struct {
				Total int `json:"total"`
			} `json:"vulnerabilities"`
		} `json:"metadata"`
		Vulnerabilities map[string]struct {
			Severity string `json:"severity"`
			Via      []any  `json:"via"`
		} `json:"vulnerabilities"`
	}

	err := json.Unmarshal([]byte(npmOutput), &auditData)
	require.NoError(t, err)

	result := AuditResult{Tool: "npm"}
	result.Vulnerabilities = auditData.Metadata.Vulnerabilities.Total
	for pkg, vuln := range auditData.Vulnerabilities {
		result.Advisories = append(result.Advisories, AuditAdvisory{
			Package:  pkg,
			Severity: vuln.Severity,
		})
	}
	sortAuditAdvisories(result.Advisories)

	assert.Equal(t, "npm", result.Tool)
	assert.Equal(t, 2, result.Vulnerabilities)
	assert.Len(t, result.Advisories, 2)
	assert.Equal(t, "lodash", result.Advisories[0].Package)
	assert.Equal(t, "high", result.Advisories[0].Severity)
	assert.Equal(t, "minimist", result.Advisories[1].Package)
	assert.Equal(t, "low", result.Advisories[1].Severity)
}

func TestRunAudit_SkipsNpmWithoutPackageJSON(t *testing.T) {
	// Create a temp directory with no package.json
	dir := t.TempDir()

	// RunAudit should only return composer result (npm skipped)
	// Note: composer will fail since it's not installed in the test env,
	// but the important thing is npm audit is NOT run
	results, err := RunAudit(context.Background(), AuditOptions{
		Dir:    dir,
		Output: os.Stdout,
	})

	// No error from RunAudit itself (individual tool errors are in AuditResult.Error)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "should only have composer result when no package.json")
	assert.Equal(t, "composer", results[0].Tool)
}

func TestRunAudit_IncludesNpmWithPackageJSON(t *testing.T) {
	// Create a temp directory with a package.json
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	require.NoError(t, err)

	results, runErr := RunAudit(context.Background(), AuditOptions{
		Dir:    dir,
		Output: os.Stdout,
	})

	// No error from RunAudit itself
	assert.NoError(t, runErr)
	assert.Len(t, results, 2, "should have both composer and npm results when package.json exists")
	assert.Equal(t, "composer", results[0].Tool)
	assert.Equal(t, "npm", results[1].Tool)
}

func TestAuditOptions_Defaults(t *testing.T) {
	opts := AuditOptions{}
	assert.Empty(t, opts.Dir)
	assert.False(t, opts.JSON)
	assert.False(t, opts.Fix)
	assert.Nil(t, opts.Output)
}

func TestAuditResult_ZeroValue(t *testing.T) {
	result := AuditResult{}
	assert.Empty(t, result.Tool)
	assert.Equal(t, 0, result.Vulnerabilities)
	assert.Nil(t, result.Advisories)
	assert.NoError(t, result.Error)
}
