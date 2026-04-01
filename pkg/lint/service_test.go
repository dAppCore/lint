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

	svc := NewService()
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

	svc := NewService()
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

func runTestCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
