package main

import (
	. "dappco.re/go"
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
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "input.go"), []byte(`package sample

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
	RequireResultOK(t, JSONUnmarshal([]byte(stdout), &report))
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

	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestCleanGob56c68), []byte(`package sample

func Clean() {}
`), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "ignored.go"), []byte(`package sample

func Run() {
	_ = helper()
}

func helper() error { return nil }
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", mainTestOutput231216, "json", "--files", mainTestCleanGob56c68, dir)
	AssertEqual(t, 0, exitCode, stderr)

	var report lintpkg.Report
	RequireResultOK(t, JSONUnmarshal([]byte(stdout), &report))
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

	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "root.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	RequireResultOK(t, MkdirAll(PathJoin(dir, "services"), 0o755))
	RequireResultOK(t, WriteFile(PathJoin(dir, "services", mainTestCleanGob56c68), []byte(`package sample

func Clean() {}
`), 0o644))
	RequireResultOK(t, MkdirAll(PathJoin(dir, ".core"), 0o755))
	RequireResultOK(t, WriteFile(PathJoin(dir, ".core", "lint.yaml"), []byte(`output: text
schedules:
  nightly:
    output: json
    paths:
      - services
`), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "run", "--schedule", "nightly", dir)
	AssertEqual(t, 0, exitCode, stderr)

	var report lintpkg.Report
	RequireResultOK(t, JSONUnmarshal([]byte(stdout), &report))
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
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "package.json"), []byte("{}\n"), 0o644))

	stdout, stderr, exitCode := runCLI(t, dir, "detect", mainTestOutput231216, "json", dir)
	AssertEqual(t, 0, exitCode, stderr)

	var languages []string
	RequireResultOK(t, JSONUnmarshal([]byte(stdout), &languages))
	AssertEqual(t, []string{"go", "js"}, languages)
}

func TestCLI_Init_WritesConfig(t *T) {
	dir := t.TempDir()

	stdout, stderr, exitCode := runCLI(t, dir, "init", dir)
	AssertEqual(t, 0, exitCode, stderr)
	AssertContains(t, stdout, ".core/lint.yaml")

	configPath := PathJoin(dir, ".core", "lint.yaml")
	content := ReadFile(configPath)
	RequireResultOK(t, content)
	config := string(content.Value.([]byte))
	AssertContains(t, config, "golangci-lint")
	AssertContains(t, config, "fail_on: error")
}

func TestCLI_Tools_TextIncludesMetadata(t *T) {
	buildCLI(t)

	binDir := t.TempDir()
	fakeToolPath := PathJoin(binDir, "gosec")
	RequireResultOK(t, WriteFile(fakeToolPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", binDir+string(PathListSeparator)+Getenv("PATH"))

	stdout, stderr, exitCode := runCoreLintCommand(t, t.TempDir(), buildCLI(t), "tools", "--lang", "go")
	AssertEqual(t, 0, exitCode, stderr)
	text := stdout + stderr
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
	RequireResultOK(t, JSONUnmarshal([]byte(stdout), &sarif))
	RequireEqual(t, "2.1.0", sarif.Version)
	RequireLen(t, sarif.Runs, 1)
	AssertEqual(t, "core-lint", sarif.Runs[0].Tool.Driver.Name)
	RequireLen(t, sarif.Runs[0].Results, 1)
	AssertEqual(t, "go-cor-003", sarif.Runs[0].Results[0].RuleID)
}

func TestCLI_HookInstallRemove(t *T) {
	if !(App{}).Find("git", "git").OK {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runCLIExpectSuccess(t, dir, "git", "init")
	runCLIExpectSuccess(t, dir, "git", "config", "user.email", "test@example.com")
	runCLIExpectSuccess(t, dir, "git", "config", "user.name", "Test User")

	_, stderr, exitCode := runCLI(t, dir, "hook", "install", dir)
	AssertEqual(t, 0, exitCode, stderr)

	hookPath := PathJoin(dir, ".git", "hooks", "pre-commit")
	hookContent := ReadFile(hookPath)
	RequireResultOK(t, hookContent)
	AssertContains(t, string(hookContent.Value.([]byte)), "core-lint run --hook")

	_, stderr, exitCode = runCLI(t, dir, "hook", "remove", dir)
	AssertEqual(t, 0, exitCode, stderr)

	removedContent := ReadFile(hookPath)
	if removedContent.OK {
		AssertNotContains(t, string(removedContent.Value.([]byte)), "core-lint run --hook")
	}
}

func runCLI(t *T, workdir string, args ...string) (string, string, int) {
	t.Helper()
	return runCoreLintCommand(t, workdir, buildCLI(t), args...)
}

func runCLIExpectSuccess(t *T, dir string, name string, args ...string) {
	t.Helper()

	stdout, stderr, exitCode := runCoreLintCommand(t, dir, name, args...)
	AssertEqual(t, 0, exitCode, stdout+stderr)
}

func buildCLI(t *T) string {
	t.Helper()

	buildBinaryOnce.Do(func() {
		repoRoot := repoRoot(t)
		binDir := MkdirTemp("", "core-lint-bin-*")
		if !binDir.OK {
			buildBinaryErr = binDir.Value.(error)
			return
		}

		builtBinaryPath = PathJoin(binDir.Value.(string), "core-lint")
		stdout, stderr, exitCode := runCoreLintCommand(t, repoRoot, "go", "build", "-o", builtBinaryPath, "./cmd/core-lint")
		if exitCode != 0 {
			buildBinaryErr = Errorf("build core-lint: exit %d: %s", exitCode, Trim(stdout+stderr))
		}
	})

	RequireNoError(t, buildBinaryErr)
	return builtBinaryPath
}

func repoRoot(t *T) string {
	t.Helper()

	root := PathAbs(PathJoin(".", "..", ".."))
	RequireResultOK(t, root)
	return root.Value.(string)
}

func runCoreLintCommand(t *T, dir string, name string, args ...string) (string, string, int) {
	t.Helper()
	path := name
	if found := (App{}).Find(name, name); found.OK {
		path = found.Value.(*App).Path
	}
	stdout := NewBuffer()
	stderr := NewBuffer()
	command := &Cmd{
		Path:   path,
		Args:   append([]string{name}, args...),
		Dir:    dir,
		Stdout: stdout,
		Stderr: stderr,
	}
	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	return stdout.String(), stderr.String(), -1
}
