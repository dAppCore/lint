package php

import (
	"context"
	. "dappco.re/go"
	"encoding/json"
	"os"
	"path/filepath"
)

func TestAuditResult_Fields(t *T) {
	result := AuditResult{
		Tool:            "composer",
		Vulnerabilities: 2,
		Advisories: []AuditAdvisory{
			{Package: "vendor/pkg", Severity: "high", Title: "RCE", URL: "https://example.com/1", Identifiers: []string{"CVE-2025-0001"}},
			{Package: "vendor/other", Severity: "medium", Title: "XSS", URL: "https://example.com/2", Identifiers: []string{"CVE-2025-0002"}},
		},
	}

	AssertEqual(t, "composer", result.Tool)
	AssertEqual(t, 2, result.Vulnerabilities)
	AssertLen(t, result.Advisories, 2)
	AssertEqual(t, "vendor/pkg", result.Advisories[0].Package)
	AssertEqual(t, "high", result.Advisories[0].Severity)
	AssertEqual(t, "RCE", result.Advisories[0].Title)
	AssertEqual(t, "https://example.com/1", result.Advisories[0].URL)
	AssertEqual(t, []string{"CVE-2025-0001"}, result.Advisories[0].Identifiers)
}

func TestAuditAdvisory_Fields(t *T) {
	adv := AuditAdvisory{
		Package:     "laravel/framework",
		Severity:    "critical",
		Title:       "SQL Injection",
		URL:         "https://example.com/advisory",
		Identifiers: []string{"CVE-2025-9999", "GHSA-xxxx"},
	}

	AssertEqual(t, "laravel/framework", adv.Package)
	AssertEqual(t, "critical", adv.Severity)
	AssertEqual(t, "SQL Injection", adv.Title)
	AssertEqual(t, "https://example.com/advisory", adv.URL)
	AssertEqual(t, []string{"CVE-2025-9999", "GHSA-xxxx"}, adv.Identifiers)
}

func TestSortAuditAdvisories_Good(t *T) {
	advisories := []AuditAdvisory{
		{Package: "vendor/package-b", Title: "Zulu"},
		{Package: "vendor/package-a", Title: "Beta"},
		{Package: "vendor/package-b", Title: "Alpha"},
	}

	sortAuditAdvisories(advisories)

	RequireLen(t, advisories, 3)
	AssertEqual(t, "vendor/package-a", advisories[0].Package)
	AssertEqual(t, "Beta", advisories[0].Title)
	AssertEqual(t, "vendor/package-b", advisories[1].Package)
	AssertEqual(t, "Alpha", advisories[1].Title)
	AssertEqual(t, "vendor/package-b", advisories[2].Package)
	AssertEqual(t, "Zulu", advisories[2].Title)
}

func TestRunComposerAudit_ParsesJSON(t *T) {
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
	RequireNoError(t, err)

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

	AssertEqual(t, "composer", result.Tool)
	AssertEqual(t, 3, result.Vulnerabilities)
	AssertLen(t, result.Advisories, 3)
	AssertEqual(t, "vendor/package-a", result.Advisories[0].Package)
	AssertEqual(t, "Remote Code Execution", result.Advisories[0].Title)
	AssertEqual(t, "https://example.com/advisory/1", result.Advisories[0].URL)
	AssertEqual(t, []string{"CVE-2025-1234"}, result.Advisories[0].Identifiers)
	AssertEqual(t, "vendor/package-b", result.Advisories[1].Package)
	AssertEqual(t, "Cross-Site Scripting", result.Advisories[1].Title)
	AssertEqual(t, "vendor/package-b", result.Advisories[2].Package)
	AssertEqual(t, "Open Redirect", result.Advisories[2].Title)
}

func TestNpmAuditJSON_ParsesCorrectly(t *T) {
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
	RequireNoError(t, err)

	result := AuditResult{Tool: "npm"}
	result.Vulnerabilities = auditData.Metadata.Vulnerabilities.Total
	for pkg, vuln := range auditData.Vulnerabilities {
		result.Advisories = append(result.Advisories, AuditAdvisory{
			Package:  pkg,
			Severity: vuln.Severity,
		})
	}
	sortAuditAdvisories(result.Advisories)

	AssertEqual(t, "npm", result.Tool)
	AssertEqual(t, 2, result.Vulnerabilities)
	AssertLen(t, result.Advisories, 2)
	AssertEqual(t, "lodash", result.Advisories[0].Package)
	AssertEqual(t, "high", result.Advisories[0].Severity)
	AssertEqual(t, "minimist", result.Advisories[1].Package)
	AssertEqual(t, "low", result.Advisories[1].Severity)
}

func TestRunAudit_SkipsNpmWithoutPackageJSON(t *T) {
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
	AssertNoError(t, err)
	AssertLen(t, results, 1, "should only have composer result when no package.json")
	AssertEqual(t, "composer", results[0].Tool)
}

func TestRunAudit_IncludesNpmWithPackageJSON(t *T) {
	// Create a temp directory with a package.json
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	RequireNoError(t, err)

	results, runErr := RunAudit(context.Background(), AuditOptions{
		Dir:    dir,
		Output: os.Stdout,
	})

	// No error from RunAudit itself
	AssertNoError(t, runErr)
	AssertLen(t, results, 2, "should have both composer and npm results when package.json exists")
	AssertEqual(t, "composer", results[0].Tool)
	AssertEqual(t, "npm", results[1].Tool)
}

func TestAuditOptions_Defaults(t *T) {
	opts := AuditOptions{}
	AssertEmpty(t, opts.Dir)
	AssertFalse(t, opts.JSON)
	AssertFalse(t, opts.Fix)
	AssertNil(t, opts.Output)
}

func TestAuditResult_ZeroValue(t *T) {
	result := AuditResult{}
	AssertEmpty(t, result.Tool)
	AssertEqual(t, 0, result.Vulnerabilities)
	AssertNil(t, result.Advisories)
	AssertNoError(t, result.Error)
}
