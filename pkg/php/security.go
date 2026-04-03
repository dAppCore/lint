package php

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

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

// securitySeverityRank maps severities to a sortable rank.
// Lower numbers are more severe.
func securitySeverityRank(severity string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 0, true
	case "high":
		return 1, true
	case "medium":
		return 2, true
	case "low":
		return 3, true
	case "info":
		return 4, true
	default:
		return 0, false
	}
}

// filterSecurityChecks returns checks at or above the requested severity.
func filterSecurityChecks(checks []SecurityCheck, minimum string) ([]SecurityCheck, error) {
	if strings.TrimSpace(minimum) == "" {
		return checks, nil
	}

	minRank, ok := securitySeverityRank(minimum)
	if !ok {
		return nil, coreerr.E("filterSecurityChecks", "invalid security severity "+minimum, nil)
	}

	filtered := make([]SecurityCheck, 0, len(checks))
	for _, check := range checks {
		rank, ok := securitySeverityRank(check.Severity)
		if !ok {
			continue
		}
		if rank <= minRank {
			filtered = append(filtered, check)
		}
	}

	return filtered, nil
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

	// Check HTTP security headers when a URL is supplied.
	result.Checks = append(result.Checks, runHTTPSecurityHeaderChecks(ctx, opts.URL)...)

	filteredChecks, err := filterSecurityChecks(result.Checks, opts.Severity)
	if err != nil {
		return nil, err
	}
	result.Checks = filteredChecks

	// Keep the check order stable for callers that consume the package result
	// directly instead of going through the CLI layer.
	slices.SortFunc(result.Checks, func(a, b SecurityCheck) int {
		return cmp.Compare(a.ID, b.ID)
	})

	// Calculate summary after any severity filtering has been applied.
	for _, check := range result.Checks {
		result.Summary.Total++
		if check.Passed {
			result.Summary.Passed++
			continue
		}

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

	return result, nil
}

func runHTTPSecurityHeaderChecks(ctx context.Context, rawURL string) []SecurityCheck {
	if strings.TrimSpace(rawURL) == "" {
		return nil
	}

	check := SecurityCheck{
		ID:          "http_security_headers",
		Name:        "HTTP Security Headers",
		Description: "Check for common security headers on the supplied URL",
		Severity:    "high",
		CWE:         "CWE-693",
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		check.Message = "Invalid URL"
		check.Fix = "Provide a valid http:// or https:// URL"
		return []SecurityCheck{check}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		check.Message = err.Error()
		check.Fix = "Provide a reachable URL"
		return []SecurityCheck{check}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		check.Message = err.Error()
		check.Fix = "Ensure the URL is reachable"
		return []SecurityCheck{check}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	requiredHeaders := []string{
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
	}
	if strings.EqualFold(parsedURL.Scheme, "https") {
		requiredHeaders = append(requiredHeaders, "Strict-Transport-Security")
	}

	var missing []string
	for _, header := range requiredHeaders {
		if strings.TrimSpace(resp.Header.Get(header)) == "" {
			missing = append(missing, header)
		}
	}

	if len(missing) == 0 {
		check.Passed = true
		check.Message = "Common security headers are present"
		return []SecurityCheck{check}
	}

	check.Message = fmt.Sprintf("Missing headers: %s", strings.Join(missing, ", "))
	check.Fix = "Add the missing security headers to the response"
	return []SecurityCheck{check}
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
