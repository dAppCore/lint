package php

import (
	"cmp"
	"context"
	"io"
	"slices"

	core "dappco.re/go"
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
func RunAudit(ctx context.Context, opts AuditOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.RunAudit", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	var results []AuditResult

	// Run composer audit
	composerResult := runComposerAudit(ctx, opts)
	results = append(results, composerResult)

	// Run npm audit if package.json exists
	if fileExists(core.PathJoin(opts.Dir, "package.json")) {
		npmResult := runNpmAudit(ctx, opts)
		results = append(results, npmResult)
	}

	return core.Ok(results)
}

func runComposerAudit(ctx context.Context, opts AuditOptions) AuditResult {
	result := AuditResult{Tool: "composer"}

	args := []string{"audit", "--format=json"}

	run := outputPHPCommand(ctx, opts.Dir, "composer", args)
	output := run.Stdout
	if run.Err != nil && run.Stderr != "" {
		output = core.Concat(output, run.Stderr)
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

	if jsonResult := core.JSONUnmarshal([]byte(output), &auditData); jsonResult.OK {
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
	} else if run.Err != nil {
		result.Error = run.Err
	}

	return result
}

func runNpmAudit(ctx context.Context, opts AuditOptions) AuditResult {
	result := AuditResult{Tool: "npm"}

	args := []string{"audit", "--json"}
	if opts.Fix {
		args = []string{"audit", "fix"}
	}

	run := outputPHPCommand(ctx, opts.Dir, "npm", args)
	output := run.Stdout
	if run.Err != nil && run.Stderr != "" {
		output = core.Concat(output, run.Stderr)
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

		if jsonResult := core.JSONUnmarshal([]byte(output), &auditData); jsonResult.OK {
			result.Vulnerabilities = auditData.Metadata.Vulnerabilities.Total
			for pkg, vuln := range auditData.Vulnerabilities {
				result.Advisories = append(result.Advisories, AuditAdvisory{
					Package:  pkg,
					Severity: vuln.Severity,
				})
			}
			sortAuditAdvisories(result.Advisories)
		} else if run.Err != nil {
			result.Error = run.Err
		}
	}

	return result
}

func sortAuditAdvisories(advisories []AuditAdvisory) {
	slices.SortFunc(advisories, func(a, b AuditAdvisory) int {
		return cmp.Or(
			cmp.Compare(a.Package, b.Package),
			cmp.Compare(a.Title, b.Title),
			cmp.Compare(a.Severity, b.Severity),
			cmp.Compare(a.URL, b.URL),
		)
	})
}
