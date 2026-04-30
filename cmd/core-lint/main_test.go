package main

import (
	. "dappco.re/go"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	lintpkg "dappco.re/go/lint/pkg/lint"
)

const (
	mainTestCleanGob56c68              = "clean.go"
	mainTestGoMod3af676                = "go.mod"
	mainTestMissingTool5b8a4a          = "missing-tool"
	mainTestModuleExampleComTeste3feb4 = "module example.com/test\n"
	mainTestOutput231216               = "--output"
)

var (
	buildBinaryOnce sync.Once
	builtBinaryPath string
	buildBinaryErr  error
)

func TestCLI_Run_JSON(t *T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())
	RequireNoError(t, os.WriteFile(filepath.Join(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", mainTestOutput231216, "json", "--fail-on", "warning", dir)
	AssertEqual(t, 1, exitCode, stderr)
	AssertContains(t, stderr, "lint failed (fail-on=warning)")

	var report lintpkg.Report
	RequireNoError(t, json.Unmarshal([]byte(stdout), &report))
	RequireNotEmpty(t, report.Findings)
	AssertGreaterOrEqual(t, report.Summary.Total, 2)
	AssertGreater(t, report.Summary.Info, 0)
	AssertContains(t, report.Summary.BySeverity, "info")
	AssertContains(t, report.Summary.BySeverity, "warning")

	var hasCatalogFinding bool
	var hasMissingToolFinding bool
	for _, finding := range report.Findings {
		switch finding.Code {
		case "go-cor-003":
			hasCatalogFinding = true
		case mainTestMissingTool5b8a4a:
			hasMissingToolFinding = true
		}
	}
	AssertTrue(t, hasCatalogFinding)
	AssertTrue(t, hasMissingToolFinding)
	AssertFalse(t, report.Summary.Passed)
}

func TestCLI_Run_FilesFlagLimitsScanning(t *T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())

	RequireNoError(t, os.WriteFile(filepath.Join(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, mainTestCleanGob56c68), []byte(`package sample

func Clean() {}
`), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "ignored.go"), []byte(`package sample

func Run() {
	_ = helper()
}

func helper() error { return nil }
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", mainTestOutput231216, "json", "--files", mainTestCleanGob56c68, dir)
	AssertEqual(t, 0, exitCode, stderr)

	var report lintpkg.Report
	RequireNoError(t, json.Unmarshal([]byte(stdout), &report))
	RequireNotEmpty(t, report.Findings)
	infoCount := 0
	for _, finding := range report.Findings {
		AssertEqual(t, mainTestMissingTool5b8a4a, finding.Code)
		AssertEqual(t, "info", finding.Severity)
		if finding.Severity == "info" {
			infoCount++
		}
	}
	AssertEqual(t, len(report.Findings), report.Summary.Total)
	AssertEqual(t, infoCount, report.Summary.Info)
	AssertTrue(t, report.Summary.Passed)
}

func TestCLI_Run_ScheduleAppliesPreset(t *T) {
	dir := t.TempDir()
	buildCLI(t)
	t.Setenv("PATH", t.TempDir())

	RequireNoError(t, os.WriteFile(filepath.Join(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "root.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, "services"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "services", mainTestCleanGob56c68), []byte(`package sample

func Clean() {}
`), 0o644))
	RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`output: text
schedules:
  nightly:
    output: json
    paths:
      - services
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", "--schedule", "nightly", dir)
	AssertEqual(t, 0, exitCode, stderr)

	var report lintpkg.Report
	RequireNoError(t, json.Unmarshal([]byte(stdout), &report))
	RequireNotEmpty(t, report.Findings)
	infoCount := 0
	for _, finding := range report.Findings {
		AssertEqual(t, mainTestMissingTool5b8a4a, finding.Code)
		AssertEqual(t, "info", finding.Severity)
		if finding.Severity == "info" {
			infoCount++
		}
	}
	AssertEqual(t, len(report.Findings), report.Summary.Total)
	AssertEqual(t, infoCount, report.Summary.Info)
	AssertTrue(t, report.Summary.Passed)
}

func TestCLI_Detect_JSON(t *T) {
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "detect", mainTestOutput231216, "json", dir)
	AssertEqual(t, 0, exitCode, stderr)

	var languages []string
	RequireNoError(t, json.Unmarshal([]byte(stdout), &languages))
	AssertEqual(t, []string{"go", "js"}, languages)
}

func TestCLI_Init_WritesConfig(t *T) {
	dir := t.TempDir()

	stdout, stderr, exitCode := runCLI(t, dir, "init", dir)
	AssertEqual(t, 0, exitCode, stderr)
	AssertContains(t, stdout, ".core/lint.yaml")

	configPath := filepath.Join(dir, ".core", "lint.yaml")
	content, err := os.ReadFile(configPath)
	RequireNoError(t, err)
	AssertContains(t, string(content), "golangci-lint")
	AssertContains(t, string(content), "fail_on: error")
}

func TestCLI_Tools_TextIncludesMetadata(t *T) {
	buildCLI(t)

	binDir := t.TempDir()
	fakeToolPath := filepath.Join(binDir, "gosec")
	RequireNoError(t, os.WriteFile(fakeToolPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	command := exec.Command(buildCLI(t), "tools", "--lang", "go")
	command.Dir = t.TempDir()
	command.Env = os.Environ()

	output, err := command.CombinedOutput()
	RequireNoError(t, err, string(output))

	text := string(output)
	AssertContains(t, text, "gosec")
	AssertContains(t, text, "langs=go")
	AssertContains(t, text, "entitlement=lint.security")
}

func TestCLI_LintCheck_SARIF(t *T) {
	buildCLI(t)

	repoRoot := repoRoot(t)
	stdout, stderr, exitCode := runCLI(t, repoRoot, "lint", "check", "--format", "sarif", "tests/cli/lint/check/fixtures")
	AssertEqual(t, 0, exitCode, stderr)

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
	RequireNoError(t, json.Unmarshal([]byte(stdout), &sarif))
	RequireEqual(t, "2.1.0", sarif.Version)
	RequireLen(t, sarif.Runs, 1)
	AssertEqual(t, "core-lint", sarif.Runs[0].Tool.Driver.Name)
	RequireLen(t, sarif.Runs[0].Results, 1)
	AssertEqual(t, "go-cor-003", sarif.Runs[0].Results[0].RuleID)
}

func TestCLI_HookInstallRemove(t *T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runCLIExpectSuccess(t, dir, "git", "init")
	runCLIExpectSuccess(t, dir, "git", "config", "user.email", "test@example.com")
	runCLIExpectSuccess(t, dir, "git", "config", "user.name", "Test User")

	_, stderr, exitCode := runCLI(t, dir, "hook", "install", dir)
	AssertEqual(t, 0, exitCode, stderr)

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	hookContent, err := os.ReadFile(hookPath)
	RequireNoError(t, err)
	AssertContains(t, string(hookContent), "core-lint run --hook")

	_, stderr, exitCode = runCLI(t, dir, "hook", "remove", dir)
	AssertEqual(t, 0, exitCode, stderr)

	removedContent, err := os.ReadFile(hookPath)
	if err == nil {
		AssertNotContains(t, string(removedContent), "core-lint run --hook")
	}
}

func runCLI(t *T, workdir string, args ...string) (string, string, int) {
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

func runCLIExpectSuccess(t *T, dir string, name string, args ...string) {
	t.Helper()

	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	RequireNoError(t, err, string(output))
}

func buildCLI(t *T) string {
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

	RequireNoError(t, buildBinaryErr)
	return builtBinaryPath
}

func repoRoot(t *T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join(".", "..", ".."))
	RequireNoError(t, err)
	return root
}
