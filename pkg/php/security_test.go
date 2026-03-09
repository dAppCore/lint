package php

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityCheck_Fields(t *testing.T) {
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

	assert.Equal(t, "debug_mode", check.ID)
	assert.Equal(t, "Debug Mode Disabled", check.Name)
	assert.Equal(t, "critical", check.Severity)
	assert.False(t, check.Passed)
	assert.Equal(t, "CWE-215", check.CWE)
	assert.Equal(t, "Set APP_DEBUG=false in .env", check.Fix)
}

func TestSecuritySummary_Fields(t *testing.T) {
	summary := SecuritySummary{
		Total:    10,
		Passed:   6,
		Critical: 2,
		High:     1,
		Medium:   1,
		Low:      0,
	}

	assert.Equal(t, 10, summary.Total)
	assert.Equal(t, 6, summary.Passed)
	assert.Equal(t, 2, summary.Critical)
	assert.Equal(t, 1, summary.High)
	assert.Equal(t, 1, summary.Medium)
	assert.Equal(t, 0, summary.Low)
}

func TestRunEnvSecurityChecks_DebugTrue(t *testing.T) {
	dir := t.TempDir()
	envContent := "APP_DEBUG=true\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	checks := runEnvSecurityChecks(dir)

	require.Len(t, checks, 1)
	assert.Equal(t, "debug_mode", checks[0].ID)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "critical", checks[0].Severity)
	assert.Equal(t, "Debug mode exposes sensitive information", checks[0].Message)
	assert.Equal(t, "Set APP_DEBUG=false in .env", checks[0].Fix)
}

func TestRunEnvSecurityChecks_AllPass(t *testing.T) {
	dir := t.TempDir()
	envContent := "APP_DEBUG=false\nAPP_KEY=base64:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nAPP_URL=https://example.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	checks := runEnvSecurityChecks(dir)

	require.Len(t, checks, 3)

	// Build a map by ID for deterministic assertions
	byID := make(map[string]SecurityCheck)
	for _, c := range checks {
		byID[c.ID] = c
	}

	assert.True(t, byID["debug_mode"].Passed)
	assert.True(t, byID["app_key_set"].Passed)
	assert.True(t, byID["https_enforced"].Passed)
}

func TestRunEnvSecurityChecks_WeakKey(t *testing.T) {
	dir := t.TempDir()
	envContent := "APP_KEY=short\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	checks := runEnvSecurityChecks(dir)

	require.Len(t, checks, 1)
	assert.Equal(t, "app_key_set", checks[0].ID)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "Missing or weak encryption key", checks[0].Message)
}

func TestRunEnvSecurityChecks_HttpUrl(t *testing.T) {
	dir := t.TempDir()
	envContent := "APP_URL=http://example.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	checks := runEnvSecurityChecks(dir)

	require.Len(t, checks, 1)
	assert.Equal(t, "https_enforced", checks[0].ID)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "high", checks[0].Severity)
	assert.Equal(t, "Application not using HTTPS", checks[0].Message)
}

func TestRunEnvSecurityChecks_NoEnvFile(t *testing.T) {
	dir := t.TempDir()

	checks := runEnvSecurityChecks(dir)
	assert.Empty(t, checks)
}

func TestRunFilesystemSecurityChecks_EnvInPublic(t *testing.T) {
	dir := t.TempDir()

	// Create public/.env
	publicDir := filepath.Join(dir, "public")
	err := os.Mkdir(publicDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(publicDir, ".env"), []byte("SECRET=leaked"), 0644)
	require.NoError(t, err)

	checks := runFilesystemSecurityChecks(dir)

	require.Len(t, checks, 1)
	assert.Equal(t, "env_not_public", checks[0].ID)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "critical", checks[0].Severity)
	assert.Contains(t, checks[0].Message, "public/.env")
}

func TestRunFilesystemSecurityChecks_GitInPublic(t *testing.T) {
	dir := t.TempDir()

	// Create public/.git directory
	gitDir := filepath.Join(dir, "public", ".git")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	checks := runFilesystemSecurityChecks(dir)

	require.Len(t, checks, 1)
	assert.Equal(t, "git_not_public", checks[0].ID)
	assert.False(t, checks[0].Passed)
	assert.Contains(t, checks[0].Message, "source code leak")
}

func TestRunFilesystemSecurityChecks_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	checks := runFilesystemSecurityChecks(dir)
	assert.Empty(t, checks)
}

func TestRunSecurityChecks_Summary(t *testing.T) {
	dir := t.TempDir()

	// Create .env with debug=true (critical fail) and http URL (high fail)
	envContent := "APP_DEBUG=true\nAPP_KEY=base64:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nAPP_URL=http://insecure.com\n"
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	result, err := RunSecurityChecks(context.Background(), SecurityOptions{Dir: dir})
	require.NoError(t, err)

	// Find the env-related checks by ID
	byID := make(map[string]SecurityCheck)
	for _, c := range result.Checks {
		byID[c.ID] = c
	}

	// debug_mode should fail (critical)
	assert.False(t, byID["debug_mode"].Passed)

	// app_key_set should pass
	assert.True(t, byID["app_key_set"].Passed)

	// https_enforced should fail (high)
	assert.False(t, byID["https_enforced"].Passed)

	// Summary should have totals
	assert.Greater(t, result.Summary.Total, 0)
	assert.Greater(t, result.Summary.Critical, 0) // at least debug_mode fails
	assert.Greater(t, result.Summary.High, 0)      // at least https_enforced fails
}

func TestRunSecurityChecks_DefaultsDir(t *testing.T) {
	// Test that empty Dir defaults to cwd (should not error)
	result, err := RunSecurityChecks(context.Background(), SecurityOptions{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCapitalise(t *testing.T) {
	assert.Equal(t, "Composer", capitalise("composer"))
	assert.Equal(t, "Npm", capitalise("npm"))
	assert.Equal(t, "", capitalise(""))
	assert.Equal(t, "A", capitalise("a"))
}
