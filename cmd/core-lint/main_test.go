package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	lintpkg "forge.lthn.ai/core/lint/pkg/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	buildBinaryOnce sync.Once
	builtBinaryPath string
	buildBinaryErr  error
)

func TestCLI_Run_JSON(t *testing.T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", "--output", "json", "--fail-on", "warning", dir)
	assert.Equal(t, 1, exitCode, stderr)

	var report lintpkg.Report
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	require.Len(t, report.Findings, 1)
	assert.Equal(t, "go-cor-003", report.Findings[0].Code)
	assert.Equal(t, 1, report.Summary.Total)
	assert.False(t, report.Summary.Passed)
}

func TestCLI_Run_FilesFlagLimitsScanning(t *testing.T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clean.go"), []byte(`package sample

func Clean() {}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.go"), []byte(`package sample

func Run() {
	_ = helper()
}

func helper() error { return nil }
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", "--output", "json", "--files", "clean.go", dir)
	assert.Equal(t, 0, exitCode, stderr)

	var report lintpkg.Report
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	assert.Empty(t, report.Findings)
	assert.Equal(t, 0, report.Summary.Total)
	assert.True(t, report.Summary.Passed)
}

func TestCLI_Run_ScheduleAppliesPreset(t *testing.T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "services"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "services", "clean.go"), []byte(`package sample

func Clean() {}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`output: text
schedules:
  nightly:
    output: json
    paths:
      - services
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", "--schedule", "nightly", dir)
	assert.Equal(t, 0, exitCode, stderr)

	var report lintpkg.Report
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	assert.Empty(t, report.Findings)
	assert.Equal(t, 0, report.Summary.Total)
	assert.True(t, report.Summary.Passed)
}

func TestCLI_Detect_JSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "detect", "--output", "json", dir)
	assert.Equal(t, 0, exitCode, stderr)

	var languages []string
	require.NoError(t, json.Unmarshal([]byte(stdout), &languages))
	assert.Equal(t, []string{"go", "js"}, languages)
}

func TestCLI_Init_WritesConfig(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, exitCode := runCLI(t, dir, "init", dir)
	assert.Equal(t, 0, exitCode, stderr)
	assert.Contains(t, stdout, ".core/lint.yaml")

	configPath := filepath.Join(dir, ".core", "lint.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "golangci-lint")
	assert.Contains(t, string(content), "fail_on: error")
}

func TestCLI_Tools_TextIncludesMetadata(t *testing.T) {
	buildCLI(t)

	binDir := t.TempDir()
	fakeToolPath := filepath.Join(binDir, "gosec")
	require.NoError(t, os.WriteFile(fakeToolPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	command := exec.Command(buildCLI(t), "tools", "--lang", "go")
	command.Dir = t.TempDir()
	command.Env = os.Environ()

	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))

	text := string(output)
	assert.Contains(t, text, "gosec")
	assert.Contains(t, text, "langs=go")
	assert.Contains(t, text, "entitlement=lint.security")
}

func TestCLI_LintCheck_SARIF(t *testing.T) {
	buildCLI(t)

	repoRoot := repoRoot(t)
	stdout, stderr, exitCode := runCLI(t, repoRoot, "lint", "check", "--format", "sarif", "tests/cli/lint/check/fixtures")
	assert.Equal(t, 0, exitCode, stderr)

	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name string `json:"name"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &sarif))
	require.Equal(t, "2.1.0", sarif.Version)
	require.Len(t, sarif.Runs, 1)
	assert.Equal(t, "core-lint", sarif.Runs[0].Tool.Driver.Name)
	require.Len(t, sarif.Runs[0].Results, 1)
	assert.Equal(t, "go-cor-003", sarif.Runs[0].Results[0].RuleID)
}

func TestCLI_HookInstallRemove(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runCLIExpectSuccess(t, dir, "git", "init")
	runCLIExpectSuccess(t, dir, "git", "config", "user.email", "test@example.com")
	runCLIExpectSuccess(t, dir, "git", "config", "user.name", "Test User")

	_, stderr, exitCode := runCLI(t, dir, "hook", "install", dir)
	assert.Equal(t, 0, exitCode, stderr)

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	hookContent, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(hookContent), "core-lint run --hook")

	_, stderr, exitCode = runCLI(t, dir, "hook", "remove", dir)
	assert.Equal(t, 0, exitCode, stderr)

	removedContent, err := os.ReadFile(hookPath)
	if err == nil {
		assert.NotContains(t, string(removedContent), "core-lint run --hook")
	}
}

func runCLI(t *testing.T, workdir string, args ...string) (string, string, int) {
	t.Helper()

	command := exec.Command(buildCLI(t), args...)
	command.Dir = workdir
	command.Env = os.Environ()
	stdout, err := command.Output()
	if err == nil {
		return string(stdout), "", 0
	}

	exitCode := -1
	stderr := ""
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		stderr = string(exitErr.Stderr)
	}

	return string(stdout), stderr, exitCode
}

func runCLIExpectSuccess(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func buildCLI(t *testing.T) string {
	t.Helper()

	buildBinaryOnce.Do(func() {
		repoRoot := repoRoot(t)
		binDir, err := os.MkdirTemp("", "core-lint-bin-*")
		if err != nil {
			buildBinaryErr = err
			return
		}

		builtBinaryPath = filepath.Join(binDir, "core-lint")
		command := exec.Command("go", "build", "-o", builtBinaryPath, "./cmd/core-lint")
		command.Dir = repoRoot
		output, err := command.CombinedOutput()
		if err != nil {
			buildBinaryErr = fmt.Errorf("build core-lint: %w: %s", err, strings.TrimSpace(string(output)))
		}
	})

	require.NoError(t, buildBinaryErr)
	return builtBinaryPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join(".", "..", ".."))
	require.NoError(t, err)
	return root
}
