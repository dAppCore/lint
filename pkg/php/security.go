package php

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

// SecurityOptions configures security scanning.
type SecurityOptions struct {
	Dir      string
	Severity string // Minimum severity (critical, high, medium, low)
	JSON     bool   // Output in JSON format
	SARIF    bool   // Output in SARIF format
	URL      string // URL to check HTTP headers (optional)
}

// SecurityResult holds the results of security scanning.
type SecurityResult struct {
	Checks  []SecurityCheck `json:"checks"`
	Summary SecuritySummary `json:"summary"`
}

// SecurityCheck represents a single security check result.
type SecurityCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message,omitempty"`
	Fix         string `json:"fix,omitempty"`
	CWE         string `json:"cwe,omitempty"`
}

// SecuritySummary summarises security check results.
type SecuritySummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// capitalise returns s with the first letter upper-cased.
func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RunSecurityChecks runs security checks on the project.
func RunSecurityChecks(ctx context.Context, opts SecurityOptions) (*SecurityResult, error) {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, coreerr.E("RunSecurityChecks", "get working directory", err)
		}
		opts.Dir = cwd
	}

	result := &SecurityResult{}

	// Run composer audit
	auditResults, _ := RunAudit(ctx, AuditOptions{Dir: opts.Dir})
	for _, audit := range auditResults {
		check := SecurityCheck{
			ID:          audit.Tool + "_audit",
			Name:        capitalise(audit.Tool) + " Security Audit",
			Description: "Check " + audit.Tool + " dependencies for vulnerabilities",
			Severity:    "critical",
			Passed:      audit.Vulnerabilities == 0 && audit.Error == nil,
			CWE:         "CWE-1395",
		}
		if !check.Passed {
			check.Message = fmt.Sprintf("Found %d vulnerabilities", audit.Vulnerabilities)
		}
		result.Checks = append(result.Checks, check)
	}

	// Check .env file for security issues
	envChecks := runEnvSecurityChecks(opts.Dir)
	result.Checks = append(result.Checks, envChecks...)

	// Check filesystem security
	fsChecks := runFilesystemSecurityChecks(opts.Dir)
	result.Checks = append(result.Checks, fsChecks...)

	// Calculate summary
	for _, check := range result.Checks {
		result.Summary.Total++
		if check.Passed {
			result.Summary.Passed++
		} else {
			switch check.Severity {
			case "critical":
				result.Summary.Critical++
			case "high":
				result.Summary.High++
			case "medium":
				result.Summary.Medium++
			case "low":
				result.Summary.Low++
			}
		}
	}

	return result, nil
}

func runEnvSecurityChecks(dir string) []SecurityCheck {
	var checks []SecurityCheck

	envPath := filepath.Join(dir, ".env")
	envContent, err := coreio.Local.Read(envPath)
	if err != nil {
		return checks
	}
	envLines := strings.Split(envContent, "\n")
	envMap := make(map[string]string)
	for _, line := range envLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check APP_DEBUG
	if debug, ok := envMap["APP_DEBUG"]; ok {
		check := SecurityCheck{
			ID:          "debug_mode",
			Name:        "Debug Mode Disabled",
			Description: "APP_DEBUG should be false in production",
			Severity:    "critical",
			Passed:      strings.ToLower(debug) != "true",
			CWE:         "CWE-215",
		}
		if !check.Passed {
			check.Message = "Debug mode exposes sensitive information"
			check.Fix = "Set APP_DEBUG=false in .env"
		}
		checks = append(checks, check)
	}

	// Check APP_KEY
	if key, ok := envMap["APP_KEY"]; ok {
		check := SecurityCheck{
			ID:          "app_key_set",
			Name:        "Application Key Set",
			Description: "APP_KEY must be set and valid",
			Severity:    "critical",
			Passed:      len(key) >= 32,
			CWE:         "CWE-321",
		}
		if !check.Passed {
			check.Message = "Missing or weak encryption key"
			check.Fix = "Run: php artisan key:generate"
		}
		checks = append(checks, check)
	}

	// Check APP_URL for HTTPS
	if url, ok := envMap["APP_URL"]; ok {
		check := SecurityCheck{
			ID:          "https_enforced",
			Name:        "HTTPS Enforced",
			Description: "APP_URL should use HTTPS in production",
			Severity:    "high",
			Passed:      strings.HasPrefix(url, "https://"),
			CWE:         "CWE-319",
		}
		if !check.Passed {
			check.Message = "Application not using HTTPS"
			check.Fix = "Update APP_URL to use https://"
		}
		checks = append(checks, check)
	}

	return checks
}

func runFilesystemSecurityChecks(dir string) []SecurityCheck {
	var checks []SecurityCheck

	// Check .env not in public
	publicEnvPaths := []string{"public/.env", "public_html/.env"}
	for _, path := range publicEnvPaths {
		fullPath := filepath.Join(dir, path)
		if fileExists(fullPath) {
			checks = append(checks, SecurityCheck{
				ID:          "env_not_public",
				Name:        ".env Not Publicly Accessible",
				Description: ".env file should not be in public directory",
				Severity:    "critical",
				Passed:      false,
				Message:     "Environment file exposed to web at " + path,
				CWE:         "CWE-538",
			})
		}
	}

	// Check .git not in public
	publicGitPaths := []string{"public/.git", "public_html/.git"}
	for _, path := range publicGitPaths {
		fullPath := filepath.Join(dir, path)
		if fileExists(fullPath) {
			checks = append(checks, SecurityCheck{
				ID:          "git_not_public",
				Name:        ".git Not Publicly Accessible",
				Description: ".git directory should not be in public",
				Severity:    "critical",
				Passed:      false,
				Message:     "Git repository exposed to web (source code leak)",
				CWE:         "CWE-538",
			})
		}
	}

	return checks
}
