package php

import (
	"context"
	. "dappco.re/go"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

func TestSecurityCheck_Fields(t *T) {
	check := SecurityCheck{
		ID:          "debug_mode",
		Name:        "Debug Mode Disabled",
		Description: "APP_DEBUG should be false in production",
		Severity:    "critical",
		Passed:      false,
		Message:     "Debug mode exposes sensitive information",
		Fix:         "Set APP_DEBUG=false in .env",
		CWE:         "CWE-215",
	}

	AssertEqual(t, "debug_mode", check.ID)
	AssertEqual(t, "Debug Mode Disabled", check.Name)
	AssertEqual(t, "critical", check.Severity)
	AssertFalse(t, check.Passed)
	AssertEqual(t, "CWE-215", check.CWE)
	AssertEqual(t, "Set APP_DEBUG=false in .env", check.Fix)
}

func TestSecuritySummary_Fields(t *T) {
	summary := SecuritySummary{
		Total:    10,
		Passed:   6,
		Critical: 2,
		High:     1,
		Medium:   1,
		Low:      0,
	}

	AssertEqual(t, 10, summary.Total)
	AssertEqual(t, 6, summary.Passed)
	AssertEqual(t, 2, summary.Critical)
	AssertEqual(t, 1, summary.High)
	AssertEqual(t, 1, summary.Medium)
	AssertEqual(t, 0, summary.Low)
}

func TestRunEnvSecurityChecks_DebugTrue(t *T) {
	dir := t.TempDir()
	envContent := "APP_DEBUG=true\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	RequireNoError(t, err)

	checks := runEnvSecurityChecks(dir)

	RequireLen(t, checks, 1)
	AssertEqual(t, "debug_mode", checks[0].ID)
	AssertFalse(t, checks[0].Passed)
	AssertEqual(t, "critical", checks[0].Severity)
	AssertEqual(t, "Debug mode exposes sensitive information", checks[0].Message)
	AssertEqual(t, "Set APP_DEBUG=false in .env", checks[0].Fix)
}

func TestRunEnvSecurityChecks_AllPass(t *T) {
	dir := t.TempDir()
	envContent := "APP_DEBUG=false\nAPP_KEY=base64:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nAPP_URL=https://example.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	RequireNoError(t, err)

	checks := runEnvSecurityChecks(dir)

	RequireLen(t, checks, 3)

	// Build a map by ID for deterministic assertions
	byID := make(map[string]SecurityCheck)
	for _, c := range checks {
		byID[c.ID] = c
	}

	AssertTrue(t, byID["debug_mode"].Passed)
	AssertTrue(t, byID["app_key_set"].Passed)
	AssertTrue(t, byID["https_enforced"].Passed)
}

func TestRunEnvSecurityChecks_WeakKey(t *T) {
	dir := t.TempDir()
	envContent := "APP_KEY=short\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	RequireNoError(t, err)

	checks := runEnvSecurityChecks(dir)

	RequireLen(t, checks, 1)
	AssertEqual(t, "app_key_set", checks[0].ID)
	AssertFalse(t, checks[0].Passed)
	AssertEqual(t, "Missing or weak encryption key", checks[0].Message)
}

func TestRunEnvSecurityChecks_HttpUrl(t *T) {
	dir := t.TempDir()
	envContent := "APP_URL=http://example.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	RequireNoError(t, err)

	checks := runEnvSecurityChecks(dir)

	RequireLen(t, checks, 1)
	AssertEqual(t, "https_enforced", checks[0].ID)
	AssertFalse(t, checks[0].Passed)
	AssertEqual(t, "high", checks[0].Severity)
	AssertEqual(t, "Application not using HTTPS", checks[0].Message)
}

func TestRunEnvSecurityChecks_NoEnvFile(t *T) {
	dir := t.TempDir()

	checks := runEnvSecurityChecks(dir)
	AssertEmpty(t, checks)
}

func TestRunFilesystemSecurityChecks_EnvInPublic(t *T) {
	dir := t.TempDir()

	// Create public/.env
	publicDir := filepath.Join(dir, "public")
	err := os.Mkdir(publicDir, 0755)
	RequireNoError(t, err)
	err = os.WriteFile(filepath.Join(publicDir, ".env"), []byte("SECRET=leaked"), 0644)
	RequireNoError(t, err)

	checks := runFilesystemSecurityChecks(dir)

	RequireLen(t, checks, 1)
	AssertEqual(t, "env_not_public", checks[0].ID)
	AssertFalse(t, checks[0].Passed)
	AssertEqual(t, "critical", checks[0].Severity)
	AssertContains(t, checks[0].Message, "public/.env")
}

func TestRunFilesystemSecurityChecks_GitInPublic(t *T) {
	dir := t.TempDir()

	// Create public/.git directory
	gitDir := filepath.Join(dir, "public", ".git")
	err := os.MkdirAll(gitDir, 0755)
	RequireNoError(t, err)

	checks := runFilesystemSecurityChecks(dir)

	RequireLen(t, checks, 1)
	AssertEqual(t, "git_not_public", checks[0].ID)
	AssertFalse(t, checks[0].Passed)
	AssertContains(t, checks[0].Message, "source code leak")
}

func TestRunFilesystemSecurityChecks_EmptyDir(t *T) {
	dir := t.TempDir()

	checks := runFilesystemSecurityChecks(dir)
	AssertEmpty(t, checks)
}

func TestRunSecurityChecks_Summary(t *T) {
	dir := t.TempDir()

	// Create .env with debug=true (critical fail) and http URL (high fail)
	envContent := "APP_DEBUG=true\nAPP_KEY=base64:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nAPP_URL=http://insecure.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	RequireNoError(t, err)

	result, err := RunSecurityChecks(context.Background(), SecurityOptions{Dir: dir})
	RequireNoError(t, err)

	// Find the env-related checks by ID
	byID := make(map[string]SecurityCheck)
	for _, c := range result.Checks {
		byID[c.ID] = c
	}

	// debug_mode should fail (critical)
	AssertFalse(t, byID["debug_mode"].Passed)

	// app_key_set should pass
	AssertTrue(t, byID["app_key_set"].Passed)

	// https_enforced should fail (high)
	AssertFalse(t, byID["https_enforced"].Passed)

	// Summary should have totals
	AssertGreater(t, result.Summary.Total, 0)
	AssertGreater(t, result.Summary.Critical, 0) // at least debug_mode fails
	AssertGreater(t, result.Summary.High, 0)     // at least https_enforced fails
}

func TestRunSecurityChecks_DefaultsDir(t *T) {
	// Test that empty Dir defaults to cwd (should not error)
	result, err := RunSecurityChecks(context.Background(), SecurityOptions{})
	RequireNoError(t, err)
	AssertNotNil(t, result)
}

func TestRunSecurityChecks_SeverityFilterCritical(t *T) {
	dir := t.TempDir()
	setupSecurityFixture(t, dir, "APP_DEBUG=true\nAPP_KEY=short\nAPP_URL=http://example.com\n")

	result, err := RunSecurityChecks(context.Background(), SecurityOptions{
		Dir:      dir,
		Severity: "critical",
	})
	RequireNoError(t, err)

	RequireLen(t, result.Checks, 3)
	AssertEqual(t, 3, result.Summary.Total)
	AssertEqual(t, 1, result.Summary.Passed)
	AssertEqual(t, 2, result.Summary.Critical)
	AssertEqual(t, 0, result.Summary.High)

	for _, check := range result.Checks {
		AssertEqual(t, "critical", check.Severity)
	}

	byID := make(map[string]SecurityCheck)
	for _, check := range result.Checks {
		byID[check.ID] = check
	}

	AssertNotContains(t, byID, "https_enforced")
	AssertContains(t, byID, "app_key_set")
	AssertContains(t, byID, "composer_audit")
	AssertContains(t, byID, "debug_mode")
}

func TestRunSecurityChecks_URLAddsHeaderCheck(t *T) {
	dir := t.TempDir()
	setupSecurityFixture(t, dir, "APP_DEBUG=false\nAPP_KEY=base64:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nAPP_URL=https://example.com\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	result, err := RunSecurityChecks(context.Background(), SecurityOptions{
		Dir: dir,
		URL: server.URL,
	})
	RequireNoError(t, err)

	byID := make(map[string]SecurityCheck)
	for _, check := range result.Checks {
		byID[check.ID] = check
	}

	headerCheck, ok := byID["http_security_headers"]
	RequireTrue(t, ok)
	AssertFalse(t, headerCheck.Passed)
	AssertEqual(t, "high", headerCheck.Severity)
	AssertTrue(t, strings.Contains(headerCheck.Message, "Missing headers"))
	AssertNotEmpty(t, headerCheck.Fix)

	AssertEqual(t, 5, result.Summary.Total)
	AssertEqual(t, 4, result.Summary.Passed)
	AssertEqual(t, 1, result.Summary.High)
}

func TestRunSecurityChecks_InvalidSeverity(t *T) {
	dir := t.TempDir()

	_, err := RunSecurityChecks(context.Background(), SecurityOptions{
		Dir:      dir,
		Severity: "banana",
	})
	RequireError(t, err)
	AssertContains(t, err.Error(), "invalid security severity")
}

func TestCapitalise(t *T) {
	AssertEqual(t, "Composer", capitalise("composer"))
	AssertEqual(t, "Npm", capitalise("npm"))
	AssertEqual(t, "", capitalise(""))
	AssertEqual(t, "A", capitalise("a"))
}

func setupSecurityFixture(t *T, dir string, envContent string) {
	t.Helper()

	RequireNoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0o644))

	composerBin := filepath.Join(dir, "composer")
	RequireNoError(t, os.WriteFile(composerBin, []byte("#!/bin/sh\ncat <<'JSON'\n{\"advisories\":{}}\nJSON\n"), 0o755))

	oldPath := os.Getenv("PATH")
	RequireNoError(t, os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath))
	t.Cleanup(func() {
		RequireNoError(t, os.Setenv("PATH", oldPath))
	})
}
