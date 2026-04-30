package lint

import (
	"context"
	core "dappco.re/go"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	serviceTestComposerJson84d91a         = "composer.json"
	serviceTestGitNotAvailable3b9d08      = "git not available"
	serviceTestGoMod12bb12                = "go.mod"
	serviceTestHidden35c0dc               = ".hidden"
	serviceTestIndexJs22f9ba              = "index.js"
	serviceTestLintYamle8fcb1             = "lint.yaml"
	serviceTestModuleExampleComTesta4210c = "module example.com/test\n"
	serviceTestNameExampleTestb3f36f      = "{\n  \"name\": \"example/test\"\n}\n"
	serviceTestRootGo4c7d7a               = "root.go"
	serviceTestScopedGod6adbc             = "scoped.go"
	serviceTestStagedGo033be3             = "staged.go"
)

func TestServiceRun_Good_CatalogFindings(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, "warning", report.Findings[0].Severity)
	core.AssertEqual(t, "catalog", report.Findings[0].Tool)
	core.AssertEqual(t, "go-cor-003", report.Findings[0].Code)
	core.AssertEqual(t, "correctness", report.Findings[0].Category)
	core.AssertEqual(t, 1, report.Summary.Total)
	core.AssertEqual(t, 1, report.Summary.Warnings)
	core.AssertFalse(t, report.Summary.Passed)
	core.AssertContains(t, report.Languages, "go")
	core.RequireNotEmpty(t, report.Tools)
	core.AssertEqual(t, "catalog", report.Tools[0].Name)
}

func TestServiceRun_Good_UsesConfiguredPaths(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestRootGo4c7d7a), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "services"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "services", serviceTestScopedGod6adbc), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("paths:\n  - services\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, "services/scoped.go", report.Findings[0].File)
	core.AssertEqual(t, 1, report.Summary.Total)
	core.AssertFalse(t, report.Summary.Passed)
}

func TestServiceRun_Good_ExplicitEmptyFilesSkipsScanning(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestRootGo4c7d7a), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Files:  []string{},
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	core.AssertEmpty(t, report.Languages)
	core.AssertEmpty(t, report.Tools)
	core.AssertEmpty(t, report.Findings)
	core.AssertTrue(t, report.Summary.Passed)
}

func TestServiceRun_Good_UsesConfiguredExclude(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestRootGo4c7d7a), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "services"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "services", serviceTestScopedGod6adbc), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("exclude:\n  - services\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, serviceTestRootGo4c7d7a, report.Findings[0].File)
	core.AssertEqual(t, 1, report.Summary.Total)
	core.AssertFalse(t, report.Summary.Passed)
}

func TestServiceRun_Good_SkipsHiddenConfiguredRootDirectory(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, serviceTestHidden35c0dc), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestHidden35c0dc, serviceTestScopedGod6adbc), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("paths:\n  - .hidden\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	core.AssertEmpty(t, report.Findings)
	core.AssertEmpty(t, report.Tools)
	core.AssertTrue(t, report.Summary.Passed)
}

func TestServiceRun_Good_SkipsHiddenConfiguredFilePath(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestRootGo4c7d7a), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, serviceTestHidden35c0dc), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestHidden35c0dc, serviceTestScopedGod6adbc), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("hidden")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("paths:\n  - root.go\n  - .hidden/scoped.go\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, serviceTestRootGo4c7d7a, report.Findings[0].File)
	core.AssertEqual(t, 1, report.Summary.Total)
	core.AssertFalse(t, report.Summary.Passed)
}

func TestServiceRun_Good_UsesNamedSchedule(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestRootGo4c7d7a), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("root")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "services"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "services", serviceTestScopedGod6adbc), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte(`schedules:
  nightly:
    fail_on: warning
    paths:
      - services
`), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, "services/scoped.go", report.Findings[0].File)
	core.AssertEqual(t, 1, report.Summary.Total)
	core.AssertFalse(t, report.Summary.Passed)
}

func TestServiceRun_Good_LanguageShortcutIgnoresCiAndSbomGroups(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte(`lint:
  go:
    - catalog
    - go-tool
  security:
    - security-tool
  compliance:
    - compliance-tool
`), 0o644))

	svc := &Service{adapters: []Adapter{
		shortcutAdapter{name: "go-tool", category: "correctness"},
		shortcutAdapter{name: "security-tool", category: "security"},
		shortcutAdapter{name: "compliance-tool", category: "compliance"},
	}}

	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Lang:   "go",
		CI:     true,
		SBOM:   true,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Tools, 2)
	core.AssertEqual(t, []string{"catalog", "go-tool"}, []string{report.Tools[0].Name, report.Tools[1].Name})
}

func TestServiceRun_Good_LanguageShortcutExcludesInfraGroup(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestComposerJson84d91a), []byte(serviceTestNameExampleTestb3f36f), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte(`lint:
  php:
    - php-tool
  infra:
    - shell-tool
`), 0o644))

	svc := &Service{adapters: []Adapter{
		shortcutAdapter{name: "php-tool", category: "correctness"},
		shortcutAdapter{name: "shell-tool", category: "correctness"},
	}}

	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Lang:   "php",
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Tools, 1)
	core.AssertEqual(t, "php-tool", report.Tools[0].Name)
}

func TestServiceRun_Good_HookModeUsesStagedFiles(t *core.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip(serviceTestGitNotAvailable3b9d08)
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")
	runTestCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runTestCommand(t, dir, "git", "config", "user.name", "Test User")
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestStagedGo033be3), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "unstaged.go"), []byte(`package sample

func run2() {
	panic("boom")
}
`), 0o644))

	runTestCommand(t, dir, "git", "add", serviceTestGoMod12bb12, serviceTestStagedGo033be3)

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Hook:   true,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, serviceTestStagedGo033be3, report.Findings[0].File)
	core.AssertEqual(t, "go-cor-003", report.Findings[0].Code)
	core.AssertFalse(t, report.Summary.Passed)
}

func TestServiceRun_Good_HookModeWithNoStagedFilesSkipsScanning(t *core.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip(serviceTestGitNotAvailable3b9d08)
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")
	runTestCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runTestCommand(t, dir, "git", "config", "user.name", "Test User")
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "unstaged.go"), []byte(`package sample

func run() {
	panic("boom")
}
`), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Hook:   true,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	core.AssertEmpty(t, report.Languages)
	core.AssertEmpty(t, report.Tools)
	core.AssertEmpty(t, report.Findings)
	core.AssertTrue(t, report.Summary.Passed)
}

func TestServiceRemoveHook_PreservesExistingHookContent(t *core.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip(serviceTestGitNotAvailable3b9d08)
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")

	original := "\n# custom hook\nprintf 'keep'"
	hookDir := filepath.Join(dir, ".git", "hooks")
	core.RequireNoError(t, os.MkdirAll(hookDir, 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(hookDir, "pre-commit"), []byte(original), 0o755))

	svc := NewService()
	core.RequireNoError(t, svc.InstallHook(dir))
	core.RequireNoError(t, svc.RemoveHook(dir))

	restored, err := os.ReadFile(filepath.Join(hookDir, "pre-commit"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, original, string(restored))
}

func TestServiceRun_JS_PrettierFindings(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\n  \"name\": \"example\"\n}\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestIndexJs22f9ba), []byte("const value = 1;\n"), 0o644))

	setupMockCmdExit(t, "prettier", "index.js\n", "", 1)

	svc := &Service{adapters: []Adapter{
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, projectPathArguments("--list-different"), parsePrettierDiagnostics),
	}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Findings, 1)
	RequireLen(t, report.Tools, 1)
	core.AssertEqual(t, "prettier", report.Findings[0].Tool)
	core.AssertEqual(t, serviceTestIndexJs22f9ba, report.Findings[0].File)
	core.AssertEqual(t, "prettier-format", report.Findings[0].Code)
	core.AssertEqual(t, "warning", report.Findings[0].Severity)
	core.AssertFalse(t, report.Summary.Passed)
	core.AssertEqual(t, "prettier", report.Tools[0].Name)
	core.AssertEqual(t, "failed", report.Tools[0].Status)
	core.AssertEqual(t, 1, report.Tools[0].Findings)
}

func TestServiceRun_CapturesToolVersion(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\n  \"name\": \"example\"\n}\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestIndexJs22f9ba), []byte("const value = 1;\n"), 0o644))

	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "prettier")
	script := `#!/bin/sh
case "$1" in
  --version)
    echo "prettier 3.2.1"
    exit 0
    ;;
  --list-different)
    echo "index.js"
    exit 1
    ;;
esac
echo "unexpected args: $*" >&2
exit 0
`
	core.RequireNoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	svc := &Service{adapters: []Adapter{
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, projectPathArguments("--list-different"), parsePrettierDiagnostics),
	}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Tools, 1)
	core.AssertEqual(t, "prettier", report.Tools[0].Name)
	core.AssertEqual(t, "prettier 3.2.1", report.Tools[0].Version)
}

func TestServiceRun_Good_ReportsMissingToolAsInfoFinding(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestComposerJson84d91a), []byte(serviceTestNameExampleTestb3f36f), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "index.php"), []byte("<?php\n"), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("lint:\n  php:\n    - missing-tool\n"), 0o644))

	svc := &Service{adapters: []Adapter{
		newCommandAdapter("missing-tool", []string{"definitely-not-installed-xyz"}, []string{"php"}, "correctness", "", false, true, projectPathArguments(), parseTextDiagnostics),
	}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Tools, 1)
	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, "skipped", report.Tools[0].Status)
	core.AssertEqual(t, "info", report.Findings[0].Severity)
	core.AssertEqual(t, "missing-tool", report.Findings[0].Code)
	core.AssertEqual(t, "definitely-not-installed-xyz is not installed", report.Findings[0].Message)
	core.AssertEqual(t, 1, report.Summary.Info)
	core.AssertTrue(t, report.Summary.Passed)
}

func TestServiceRun_Good_DeduplicatesMergedFindings(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestGoMod12bb12), []byte(serviceTestModuleExampleComTesta4210c), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte("lint:\n  go:\n    - dup\n"), 0o644))

	finding := Finding{
		Tool:     "dup",
		File:     filepath.Join(dir, "input.go"),
		Line:     12,
		Column:   3,
		Severity: "warning",
		Code:     "duplicate-finding",
		Message:  "same finding",
	}

	svc := &Service{adapters: []Adapter{
		duplicateAdapter{name: "dup", finding: finding},
		duplicateAdapter{name: "dup", finding: finding},
	}}

	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	core.RequireNoError(t, err)

	RequireLen(t, report.Tools, 3)
	RequireLen(t, report.Findings, 1)
	core.AssertEqual(t, "duplicate-finding", report.Findings[0].Code)
	core.AssertEqual(t, 1, report.Summary.Total)
}

func TestServiceTools_EmptyInventoryReturnsEmptySlice(t *core.T) {
	tools := (&Service{}).Tools(nil)
	RequireNotNil(t, tools)
	core.AssertEmpty(t, tools)
}

func TestServiceRun_Good_StopsDispatchingAfterContextCancel(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, serviceTestComposerJson84d91a), []byte(serviceTestNameExampleTestb3f36f), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", serviceTestLintYamle8fcb1), []byte(`lint:
  php:
    - first
    - second
`), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var secondRan bool
	svc := &Service{adapters: []Adapter{
		cancellingAdapter{name: "first", cancel: cancel},
		recordingAdapter{name: "second", ran: &secondRan},
	}}

	report, err := svc.Run(ctx, RunInput{
		Path:   dir,
		Lang:   "php",
		FailOn: "warning",
	})
	RequireErrorIs(t, err, context.Canceled)
	core.AssertFalse(t, secondRan)
	core.AssertEmpty(t, report.Tools)
	core.AssertEmpty(t, report.Findings)
}

type shortcutAdapter struct {
	name     string
	category string
}

func (adapter shortcutAdapter) Name() string { return adapter.name }

func (adapter shortcutAdapter) Available() bool { return true }

func (adapter shortcutAdapter) Languages() []string { return []string{"*"} }

func (adapter shortcutAdapter) Command() string { return adapter.name }

func (adapter shortcutAdapter) Entitlement() string { return "" }

func (adapter shortcutAdapter) RequiresEntitlement() bool { return false }

func (adapter shortcutAdapter) MatchesLanguage(languages []string) bool { return true }

func (adapter shortcutAdapter) Category() string { return adapter.category }

func (adapter shortcutAdapter) Fast() bool { return true }

func (adapter shortcutAdapter) Run(_ context.Context, _ RunInput, _ []string) AdapterResult {
	return AdapterResult{
		Tool: ToolRun{
			Name:     adapter.name,
			Status:   "passed",
			Duration: "0s",
		},
	}
}

type recordingAdapter struct {
	name string
	ran  *bool
}

func (adapter recordingAdapter) Name() string { return adapter.name }

func (adapter recordingAdapter) Available() bool { return true }

func (adapter recordingAdapter) Languages() []string { return []string{"php"} }

func (adapter recordingAdapter) Command() string { return adapter.name }

func (adapter recordingAdapter) Entitlement() string { return "" }

func (adapter recordingAdapter) RequiresEntitlement() bool { return false }

func (adapter recordingAdapter) MatchesLanguage(languages []string) bool {
	for _, language := range languages {
		if language == "php" {
			return true
		}
	}
	return false
}

func (adapter recordingAdapter) Category() string { return "correctness" }

func (adapter recordingAdapter) Fast() bool { return true }

func (adapter recordingAdapter) Run(_ context.Context, _ RunInput, _ []string) AdapterResult {
	if adapter.ran != nil {
		*adapter.ran = true
	}
	return AdapterResult{
		Tool: ToolRun{
			Name:     adapter.name,
			Status:   "passed",
			Duration: "0s",
		},
	}
}

type cancellingAdapter struct {
	name   string
	cancel context.CancelFunc
}

func (adapter cancellingAdapter) Name() string { return adapter.name }

func (adapter cancellingAdapter) Available() bool { return true }

func (adapter cancellingAdapter) Languages() []string { return []string{"php"} }

func (adapter cancellingAdapter) Command() string { return adapter.name }

func (adapter cancellingAdapter) Entitlement() string { return "" }

func (adapter cancellingAdapter) RequiresEntitlement() bool { return false }

func (adapter cancellingAdapter) MatchesLanguage(languages []string) bool {
	for _, language := range languages {
		if language == "php" {
			return true
		}
	}
	return false
}

func (adapter cancellingAdapter) Category() string { return "correctness" }

func (adapter cancellingAdapter) Fast() bool { return true }

func (adapter cancellingAdapter) Run(_ context.Context, _ RunInput, _ []string) AdapterResult {
	if adapter.cancel != nil {
		adapter.cancel()
	}
	return AdapterResult{
		Tool: ToolRun{
			Name:     adapter.name,
			Status:   "passed",
			Duration: "0s",
		},
	}
}

type duplicateAdapter struct {
	name    string
	finding Finding
}

func (adapter duplicateAdapter) Name() string { return adapter.name }

func (adapter duplicateAdapter) Available() bool { return true }

func (adapter duplicateAdapter) Languages() []string { return []string{"go"} }

func (adapter duplicateAdapter) Command() string { return adapter.name }

func (adapter duplicateAdapter) Entitlement() string { return "" }

func (adapter duplicateAdapter) RequiresEntitlement() bool { return false }

func (adapter duplicateAdapter) MatchesLanguage(languages []string) bool {
	for _, language := range languages {
		if language == "go" {
			return true
		}
	}
	return false
}

func (adapter duplicateAdapter) Category() string { return "correctness" }

func (adapter duplicateAdapter) Fast() bool { return true }

func (adapter duplicateAdapter) Run(_ context.Context, _ RunInput, _ []string) AdapterResult {
	return AdapterResult{
		Tool: ToolRun{
			Name:     adapter.name,
			Status:   "passed",
			Duration: "0s",
		},
		Findings: []Finding{adapter.finding},
	}
}

func runTestCommand(t *core.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	core.RequireNoError(t, err, string(output))
}
