package php

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// AuditOptions configures dependency security auditing.
type AuditOptions struct {
	Dir    string
	JSON   bool // Output in JSON format
	Fix    bool // Auto-fix vulnerabilities (npm only)
	Output io.Writer
}

// AuditResult holds the results of a security audit.
type AuditResult struct {
	Tool            string
	Vulnerabilities int
	Advisories      []AuditAdvisory
	Error           error
}

// AuditAdvisory represents a single security advisory.
type AuditAdvisory struct {
	Package     string
	Severity    string
	Title       string
	URL         string
	Identifiers []string
}

// RunAudit runs security audits on dependencies.
func RunAudit(ctx context.Context, opts AuditOptions) ([]AuditResult, error) {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	var results []AuditResult

	// Run composer audit
	composerResult := runComposerAudit(ctx, opts)
	results = append(results, composerResult)

	// Run npm audit if package.json exists
	if fileExists(filepath.Join(opts.Dir, "package.json")) {
		npmResult := runNpmAudit(ctx, opts)
		results = append(results, npmResult)
	}

	return results, nil
}

func runComposerAudit(ctx context.Context, opts AuditOptions) AuditResult {
	result := AuditResult{Tool: "composer"}

	args := []string{"audit", "--format=json"}

	cmd := exec.CommandContext(ctx, "composer", args...)
	cmd.Dir = opts.Dir

	output, err := cmd.Output()
	if err != nil {
		// composer audit returns non-zero if vulnerabilities found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = append(output, exitErr.Stderr...)
		}
	}

	// Parse JSON output
	var auditData struct {
		Advisories map[string][]struct {
			Title          string `json:"title"`
			Link           string `json:"link"`
			CVE            string `json:"cve"`
			AffectedRanges string `json:"affectedVersions"`
		} `json:"advisories"`
	}

	if jsonErr := json.Unmarshal(output, &auditData); jsonErr == nil {
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
		result.Vulnerabilities = len(result.Advisories)
	} else if err != nil {
		result.Error = err
	}

	return result
}

func runNpmAudit(ctx context.Context, opts AuditOptions) AuditResult {
	result := AuditResult{Tool: "npm"}

	args := []string{"audit", "--json"}
	if opts.Fix {
		args = []string{"audit", "fix"}
	}

	cmd := exec.CommandContext(ctx, "npm", args...)
	cmd.Dir = opts.Dir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = append(output, exitErr.Stderr...)
		}
	}

	if !opts.Fix {
		// Parse JSON output
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

		if jsonErr := json.Unmarshal(output, &auditData); jsonErr == nil {
			result.Vulnerabilities = auditData.Metadata.Vulnerabilities.Total
			for pkg, vuln := range auditData.Vulnerabilities {
				result.Advisories = append(result.Advisories, AuditAdvisory{
					Package:  pkg,
					Severity: vuln.Severity,
				})
			}
		} else if err != nil {
			result.Error = err
		}
	}

	return result
}
