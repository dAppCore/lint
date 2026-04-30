package php

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"time"

	core "dappco.re/go"
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
	return core.Upper(s[:1]) + s[1:]
}

// securitySeverityRank maps severities to a sortable rank.
// Lower numbers are more severe.
func securitySeverityRank(severity string) (int, bool) {
	switch core.Lower(core.Trim(severity)) {
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
func filterSecurityChecks(checks []SecurityCheck, minimum string) core.Result {
	if core.Trim(minimum) == "" {
		return core.Ok(checks)
	}

	minRank, ok := securitySeverityRank(minimum)
	if !ok {
		return core.Fail(core.E("filterSecurityChecks", "invalid security severity "+minimum, nil))
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

	return core.Ok(filtered)
}

// RunSecurityChecks runs security checks on the project.
func RunSecurityChecks(ctx context.Context, opts SecurityOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("RunSecurityChecks", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	result := &SecurityResult{}

	// Run composer audit
	auditResults := RunAudit(ctx, AuditOptions{Dir: opts.Dir})
	if auditResults.OK {
		for _, audit := range auditResults.Value.([]AuditResult) {
			check := SecurityCheck{
				ID:          audit.Tool + "_audit",
				Name:        capitalise(audit.Tool) + " Security Audit",
				Description: "Check " + audit.Tool + " dependencies for vulnerabilities",
				Severity:    "critical",
				Passed:      audit.Vulnerabilities == 0 && audit.Error == nil,
				CWE:         "CWE-1395",
			}
			if !check.Passed {
				check.Message = core.Sprintf("Found %d vulnerabilities", audit.Vulnerabilities)
			}
			result.Checks = append(result.Checks, check)
		}
	}

	// Check .env file for security issues
	envChecks := runEnvSecurityChecks(opts.Dir)
	result.Checks = append(result.Checks, envChecks...)

	// Check filesystem security
	fsChecks := runFilesystemSecurityChecks(opts.Dir)
	result.Checks = append(result.Checks, fsChecks...)

	// Check HTTP security headers when a URL is supplied.
	result.Checks = append(result.Checks, runHTTPSecurityHeaderChecks(ctx, opts.URL)...)

	filteredChecks := filterSecurityChecks(result.Checks, opts.Severity)
	if !filteredChecks.OK {
		return filteredChecks
	}
	result.Checks = filteredChecks.Value.([]SecurityCheck)

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

	return core.Ok(result)
}

func runHTTPSecurityHeaderChecks(ctx context.Context, rawURL string) []SecurityCheck {
	if core.Trim(rawURL) == "" {
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
	if copied := core.Copy(core.Discard, resp.Body); !copied.OK {
		check.Message = copied.Error()
		check.Fix = "Ensure the URL response body can be read"
		return []SecurityCheck{check}
	}

	requiredHeaders := []string{
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
	}
	if core.Lower(parsedURL.Scheme) == "https" {
		requiredHeaders = append(requiredHeaders, "Strict-Transport-Security")
	}

	var missing []string
	for _, header := range requiredHeaders {
		if core.Trim(resp.Header.Get(header)) == "" {
			missing = append(missing, header)
		}
	}

	if len(missing) == 0 {
		check.Passed = true
		check.Message = "Common security headers are present"
		return []SecurityCheck{check}
	}

	check.Message = core.Sprintf("Missing headers: %s", core.Join(", ", missing...))
	check.Fix = "Add the missing security headers to the response"
	return []SecurityCheck{check}
}

func runEnvSecurityChecks(dir string) []SecurityCheck {
	envMap := readEnvMap(core.PathJoin(dir, ".env"))
	if !envMap.OK {
		return nil
	}
	return envSecurityChecks(envMap.Value.(map[string]string))
}

func readEnvMap(path string) core.Result {
	read := core.ReadFile(path)
	if !read.OK {
		return read
	}
	envMap := make(map[string]string)
	for _, line := range core.Split(string(read.Value.([]byte)), "\n") {
		addEnvLine(envMap, line)
	}
	return core.Ok(envMap)
}

func addEnvLine(envMap map[string]string, line string) {
	line = core.Trim(line)
	if line == "" || core.HasPrefix(line, "#") {
		return
	}
	parts := core.SplitN(line, "=", 2)
	if len(parts) == 2 {
		envMap[parts[0]] = parts[1]
	}
}

func envSecurityChecks(envMap map[string]string) []SecurityCheck {
	var checks []SecurityCheck
	if debug, ok := envMap["APP_DEBUG"]; ok {
		checks = append(checks, appDebugCheck(debug))
	}
	if key, ok := envMap["APP_KEY"]; ok {
		checks = append(checks, appKeyCheck(key))
	}
	if url, ok := envMap["APP_URL"]; ok {
		checks = append(checks, appURLCheck(url))
	}
	return checks
}

func appDebugCheck(debug string) SecurityCheck {
	check := SecurityCheck{
		ID:          "debug_mode",
		Name:        "Debug Mode Disabled",
		Description: "APP_DEBUG should be false in production",
		Severity:    "critical",
		Passed:      core.Lower(debug) != "true",
		CWE:         "CWE-215",
	}
	if !check.Passed {
		check.Message = "Debug mode exposes sensitive information"
		check.Fix = "Set APP_DEBUG=false in .env"
	}
	return check
}

func appKeyCheck(key string) SecurityCheck {
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
	return check
}

func appURLCheck(url string) SecurityCheck {
	check := SecurityCheck{
		ID:          "https_enforced",
		Name:        "HTTPS Enforced",
		Description: "APP_URL should use HTTPS in production",
		Severity:    "high",
		Passed:      core.HasPrefix(url, "https://"),
		CWE:         "CWE-319",
	}
	if !check.Passed {
		check.Message = "Application not using HTTPS"
		check.Fix = "Update APP_URL to use https://"
	}
	return check
}

func runFilesystemSecurityChecks(dir string) []SecurityCheck {
	var checks []SecurityCheck

	// Check .env not in public
	publicEnvPaths := []string{"public/.env", "public_html/.env"}
	for _, path := range publicEnvPaths {
		fullPath := core.PathJoin(dir, path)
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
		fullPath := core.PathJoin(dir, path)
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
