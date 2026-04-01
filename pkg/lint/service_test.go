package lint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRun_Good_CatalogFindings(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

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
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "warning", report.Findings[0].Severity)
	assert.Equal(t, "catalog", report.Findings[0].Tool)
	assert.Equal(t, "go-cor-003", report.Findings[0].Code)
	assert.Equal(t, "correctness", report.Findings[0].Category)
	assert.Equal(t, 1, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Warnings)
	assert.False(t, report.Summary.Passed)
	assert.Contains(t, report.Languages, "go")
	require.NotEmpty(t, report.Tools)
	assert.Equal(t, "catalog", report.Tools[0].Name)
}

func TestServiceRun_Good_UsesConfiguredPaths(t *testing.T) {
	dir := t.TempDir()
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "services", "scoped.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("paths:\n  - services\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "services/scoped.go", report.Findings[0].File)
	assert.Equal(t, 1, report.Summary.Total)
	assert.False(t, report.Summary.Passed)
}

func TestServiceRun_Good_ExplicitEmptyFilesSkipsScanning(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.go"), []byte(`package sample

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
	require.NoError(t, err)

	assert.Empty(t, report.Languages)
	assert.Empty(t, report.Tools)
	assert.Empty(t, report.Findings)
	assert.True(t, report.Summary.Passed)
}

func TestServiceRun_Good_UsesConfiguredExclude(t *testing.T) {
	dir := t.TempDir()
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "services", "scoped.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("exclude:\n  - services\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "root.go", report.Findings[0].File)
	assert.Equal(t, 1, report.Summary.Total)
	assert.False(t, report.Summary.Passed)
}

func TestServiceRun_Good_SkipsHiddenConfiguredRootDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden", "scoped.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("paths:\n  - .hidden\n"), 0o644))

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	require.NoError(t, err)

	assert.Empty(t, report.Findings)
	assert.Empty(t, report.Tools)
	assert.True(t, report.Summary.Passed)
}

func TestServiceRun_Good_UsesNamedSchedule(t *testing.T) {
	dir := t.TempDir()
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "services", "scoped.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("scoped")
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`schedules:
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
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "services/scoped.go", report.Findings[0].File)
	assert.Equal(t, 1, report.Summary.Total)
	assert.False(t, report.Summary.Passed)
}

func TestServiceRun_Good_LanguageShortcutIgnoresCiAndSbomGroups(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`lint:
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
	require.NoError(t, err)

	require.Len(t, report.Tools, 2)
	assert.Equal(t, []string{"catalog", "go-tool"}, []string{report.Tools[0].Name, report.Tools[1].Name})
}

func TestServiceRun_Good_HookModeUsesStagedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")
	runTestCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runTestCommand(t, dir, "git", "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "staged.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unstaged.go"), []byte(`package sample

func run2() {
	panic("boom")
}
`), 0o644))

	runTestCommand(t, dir, "git", "add", "go.mod", "staged.go")

	svc := &Service{adapters: []Adapter{newCatalogAdapter()}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		Hook:   true,
		FailOn: "warning",
	})
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "staged.go", report.Findings[0].File)
	assert.Equal(t, "go-cor-003", report.Findings[0].Code)
	assert.False(t, report.Summary.Passed)
}

func TestServiceRun_Good_HookModeWithNoStagedFilesSkipsScanning(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")
	runTestCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runTestCommand(t, dir, "git", "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unstaged.go"), []byte(`package sample

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
	require.NoError(t, err)

	assert.Empty(t, report.Languages)
	assert.Empty(t, report.Tools)
	assert.Empty(t, report.Findings)
	assert.True(t, report.Summary.Passed)
}

func TestServiceRemoveHook_PreservesExistingHookContent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runTestCommand(t, dir, "git", "init")

	original := "\n# custom hook\nprintf 'keep'"
	hookDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hookDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hookDir, "pre-commit"), []byte(original), 0o755))

	svc := NewService()
	require.NoError(t, svc.InstallHook(dir))
	require.NoError(t, svc.RemoveHook(dir))

	restored, err := os.ReadFile(filepath.Join(hookDir, "pre-commit"))
	require.NoError(t, err)
	assert.Equal(t, original, string(restored))
}

func TestServiceRun_JS_PrettierFindings(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\n  \"name\": \"example\"\n}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("const value = 1;\n"), 0o644))

	setupMockCmdExit(t, "prettier", "index.js\n", "", 1)

	svc := &Service{adapters: []Adapter{
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, pathArgs("--list-different"), parsePrettierDiagnostics),
	}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	require.NoError(t, err)

	require.Len(t, report.Findings, 1)
	require.Len(t, report.Tools, 1)
	assert.Equal(t, "prettier", report.Findings[0].Tool)
	assert.Equal(t, "index.js", report.Findings[0].File)
	assert.Equal(t, "prettier-format", report.Findings[0].Code)
	assert.Equal(t, "warning", report.Findings[0].Severity)
	assert.False(t, report.Summary.Passed)
	assert.Equal(t, "prettier", report.Tools[0].Name)
	assert.Equal(t, "failed", report.Tools[0].Status)
	assert.Equal(t, 1, report.Tools[0].Findings)
}

func TestServiceRun_CapturesToolVersion(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\n  \"name\": \"example\"\n}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("const value = 1;\n"), 0o644))

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
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	svc := &Service{adapters: []Adapter{
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, pathArgs("--list-different"), parsePrettierDiagnostics),
	}}
	report, err := svc.Run(context.Background(), RunInput{
		Path:   dir,
		FailOn: "warning",
	})
	require.NoError(t, err)

	require.Len(t, report.Tools, 1)
	assert.Equal(t, "prettier", report.Tools[0].Name)
	assert.Equal(t, "prettier 3.2.1", report.Tools[0].Version)
}

func TestServiceRun_Good_DeduplicatesMergedFindings(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("lint:\n  go:\n    - dup\n"), 0o644))

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
	require.NoError(t, err)

	require.Len(t, report.Tools, 3)
	require.Len(t, report.Findings, 1)
	assert.Equal(t, "duplicate-finding", report.Findings[0].Code)
	assert.Equal(t, 1, report.Summary.Total)
}

func TestServiceTools_EmptyInventoryReturnsEmptySlice(t *testing.T) {
	tools := (&Service{}).Tools(nil)
	require.NotNil(t, tools)
	assert.Empty(t, tools)
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

func runTestCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
