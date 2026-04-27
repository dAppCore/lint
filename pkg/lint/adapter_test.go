package lint

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adapterTestCommandEnv = "LINT_ADAPTER_TEST_COMMAND"
	adapterTestProcessEnv = "LINT_ADAPTER_TEST_PROCESS"
)

func TestMain(m *testing.M) {
	if os.Getenv(adapterTestProcessEnv) == "1" {
		os.Exit(runAdapterTestCommand(os.Getenv(adapterTestCommandEnv), os.Args[1:]))
	}
	os.Exit(m.Run())
}

func TestAdapter_CommandAdapter_Good(t *testing.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, "demo-tool")
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		"demo-tool",
		[]string{"demo-tool"},
		[]string{"go"},
		"security",
		"lint.security",
		true,
		true,
		func(_ string, _ []string) []string {
			return []string{"scan"}
		},
		nil,
	).(CommandAdapter)

	langs := adapter.Languages()
	assert.Equal(t, "demo-tool", adapter.Name())
	assert.True(t, adapter.Available())
	assert.Equal(t, "demo-tool", adapter.Command())
	assert.Equal(t, "lint.security", adapter.Entitlement())
	assert.True(t, adapter.RequiresEntitlement())
	assert.True(t, adapter.MatchesLanguage([]string{"go"}))
	assert.True(t, adapter.MatchesLanguage([]string{"security"}))
	assert.True(t, adapter.MatchesLanguage(nil))
	assert.False(t, adapter.MatchesLanguage([]string{"rust"}))
	assert.Equal(t, "security", adapter.Category())
	assert.True(t, adapter.Fast())
	assert.Equal(t, []string{"go"}, langs)

	langs[0] = "mutated"
	assert.Equal(t, []string{"go"}, adapter.Languages())

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	require.Equal(t, "passed", result.Tool.Status)
	assert.Equal(t, "demo-tool version 1.2.3", result.Tool.Version)
	assert.Equal(t, 0, result.Tool.Findings)
	assert.Empty(t, result.Findings)
}

func TestAdapter_CommandAdapter_Bad(t *testing.T) {
	adapter := CommandAdapter{
		name:      "missing-tool",
		binaries:  []string{"missing-tool"},
		languages: []string{"go"},
		category:  "security",
	}

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	require.Equal(t, "skipped", result.Tool.Status)
	assert.Equal(t, "0s", result.Tool.Duration)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "info", result.Findings[0].Severity)
	assert.Equal(t, "missing-tool", result.Findings[0].Code)
	assert.Equal(t, "missing-tool", result.Findings[0].Tool)
	assert.Equal(t, "security", result.Findings[0].Category)
	assert.Equal(t, 1, result.Tool.Findings)
}

func TestAdapter_CommandAdapter_ParsesStdoutAndStderr(t *testing.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, "mixed-output-tool")
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		"mixed-output-tool",
		[]string{"mixed-output-tool"},
		[]string{"go"},
		"security",
		"",
		false,
		true,
		func(_ string, _ []string) []string {
			return []string{"scan"}
		},
		parseJSONDiagnostics,
	).(CommandAdapter)

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	if result.Tool.Status != "failed" {
		t.Fatalf("status = %q, want failed", result.Tool.Status)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("findings = %d, want 2: %#v", len(result.Findings), result.Findings)
	}
	if result.Tool.Findings != 2 {
		t.Fatalf("tool findings = %d, want 2", result.Tool.Findings)
	}

	var foundParseError bool
	var foundStderrFinding bool
	for _, finding := range result.Findings {
		if finding.Code == "parse-error" {
			foundParseError = true
		}
		if finding.Code == "S1" && finding.File == "src/secret.go" {
			foundStderrFinding = true
		}
	}
	if !foundParseError {
		t.Fatal("missing stdout parse-error finding")
	}
	if !foundStderrFinding {
		t.Fatal("missing stderr diagnostic finding")
	}
}

func TestAdapter_CommandAdapter_Ugly(t *testing.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, "slow-tool")
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		"slow-tool",
		[]string{"slow-tool"},
		[]string{"go"},
		"security",
		"",
		false,
		false,
		func(_ string, _ []string) []string {
			return []string{"scan"}
		},
		nil,
	).(CommandAdapter)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := adapter.Run(ctx, RunInput{Path: t.TempDir()}, nil)
	require.Equal(t, "timeout", result.Tool.Status)
	assert.Equal(t, "slow-tool 9.9.9", result.Tool.Version)
	assert.Empty(t, result.Findings)
}

func TestAdapter_ParseJSONDiagnostics_Good(t *testing.T) {
	output := `[
  {
    "location": {
      "path": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  },
  {
    "location": {
      "path": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  }
]`

	findings := parseJSONDiagnostics("gosec", "security", output)
	require.Len(t, findings, 1)
	assert.Equal(t, "gosec", findings[0].Tool)
	assert.Equal(t, "internal/foo/bar.go", findings[0].File)
	assert.Equal(t, 42, findings[0].Line)
	assert.Equal(t, 5, findings[0].Column)
	assert.Equal(t, "G104", findings[0].Code)
	assert.Equal(t, "Errors unhandled", findings[0].Message)
	assert.Equal(t, "warning", findings[0].Severity)
	assert.Equal(t, "security", findings[0].Category)
}

func TestAdapter_ParseJSONDiagnostics_Bad(t *testing.T) {
	findings := parseJSONDiagnostics("gosec", "security", "{not json")
	require.Len(t, findings, 1)
	assert.Equal(t, "error", findings[0].Severity)
	assert.Equal(t, "parse-error", findings[0].Code)
	assert.Equal(t, "gosec", findings[0].Tool)
	assert.Equal(t, "security", findings[0].Category)
	assert.Contains(t, findings[0].Message, "failed to parse JSON output")
}

func TestAdapter_ParseJSONDiagnostics_PartialOutput(t *testing.T) {
	output := `[
  {
    "location": {
      "path": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  }
]
not json`

	findings := parseJSONDiagnostics("gosec", "security", output)
	require.Len(t, findings, 2)
	assert.Equal(t, "G104", findings[0].Code)
	assert.Equal(t, "parse-error", findings[1].Code)
}

func TestAdapter_ParseJSONDiagnostics_Ugly(t *testing.T) {
	output := `[
  {
    "Location": {
      "Path": "src/main.go",
      "Start": {"Line": "7", "Column": "3"}
    },
    "Message": {"Text": "Potential issue"},
    "RuleID": "ABC123"
  }
]`

	findings := parseJSONDiagnostics("scanner", "security", output)
	require.Len(t, findings, 1)
	assert.Equal(t, "scanner", findings[0].Tool)
	assert.Equal(t, "src/main.go", findings[0].File)
	assert.Equal(t, 7, findings[0].Line)
	assert.Equal(t, 3, findings[0].Column)
	assert.Equal(t, "ABC123", findings[0].Code)
	assert.Equal(t, "Potential issue", findings[0].Message)
	assert.Equal(t, "error", findings[0].Severity)
	assert.Equal(t, "security", findings[0].Category)
}

func TestAdapter_CommandAdapter_JSONStdoutIgnoresStderr(t *testing.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, "json-tool")
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		"json-tool",
		[]string{"json-tool"},
		[]string{"go"},
		"security",
		"",
		false,
		true,
		func(_ string, _ []string) []string {
			return []string{"scan"}
		},
		parseJSONDiagnostics,
	).(CommandAdapter)

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	require.Equal(t, "failed", result.Tool.Status)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "X1", result.Findings[0].Code)
	assert.Equal(t, "src/main.go", result.Findings[0].File)
	assert.Equal(t, "boom", result.Findings[0].Message)
	assert.Equal(t, "warning", result.Findings[0].Severity)
}

func TestAdapter_ParseTextDiagnostics_Good(t *testing.T) {
	output := strings.Join([]string{
		"src/main.go:12:3:error: boom",
		"src/main.go:13:warning: caution",
	}, "\n")

	findings := parseTextDiagnostics("eslint", "security", output)
	require.Len(t, findings, 2)
	assert.Equal(t, 12, findings[0].Line)
	assert.Equal(t, 3, findings[0].Column)
	assert.Equal(t, "error", findings[0].Severity)
	assert.Equal(t, 13, findings[1].Line)
	assert.Equal(t, 0, findings[1].Column)
	assert.Equal(t, "warning", findings[1].Severity)
}

func TestAdapter_ParseTextDiagnostics_Bad(t *testing.T) {
	assert.Empty(t, parseTextDiagnostics("eslint", "security", ""))
}

func TestAdapter_ParseTextDiagnostics_Ugly(t *testing.T) {
	findings := parseTextDiagnostics("eslint", "security", "not parseable")
	require.Len(t, findings, 1)
	assert.Equal(t, "diagnostic", findings[0].Code)
	assert.Equal(t, "not parseable", findings[0].Message)
	assert.Equal(t, "error", findings[0].Severity)
}

func TestAdapter_ParseGovulncheckDiagnostics_Good(t *testing.T) {
	output := `{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
{"osv":{"id":"GO-2024-1234","summary":"Buffer overflow in foo","aliases":["CVE-2024-1234"],"affected":[{"ranges":[{"events":[{"fixed":"1.2.3"}]}]}]}}
{"finding":{"osv":"GO-2024-1234","trace":[{"package":"example.com/foo","function":"Bar"}]}}
`

	findings := parseGovulncheckDiagnostics("govulncheck", "security", output)
	require.Len(t, findings, 1)
	assert.Equal(t, "govulncheck", findings[0].Tool)
	assert.Equal(t, "example.com/foo", findings[0].File)
	assert.Equal(t, "GO-2024-1234", findings[0].Code)
	assert.Equal(t, "Buffer overflow in foo", findings[0].Message)
	assert.Equal(t, "error", findings[0].Severity)
	assert.Equal(t, "security", findings[0].Category)
}

func TestAdapter_ParseGovulncheckDiagnostics_Bad(t *testing.T) {
	assert.Empty(t, parseGovulncheckDiagnostics("govulncheck", "security", "not json"))
}

func TestAdapter_CatalogAdapter_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	adapter := CatalogAdapter{}
	assert.Equal(t, "catalog", adapter.Name())
	assert.True(t, adapter.Available())
	assert.Equal(t, []string{"go"}, adapter.Languages())
	assert.Equal(t, "catalog", adapter.Command())
	assert.Empty(t, adapter.Entitlement())
	assert.False(t, adapter.RequiresEntitlement())
	assert.True(t, adapter.MatchesLanguage(nil))
	assert.True(t, adapter.MatchesLanguage([]string{"go"}))
	assert.False(t, adapter.MatchesLanguage([]string{"rust"}))
	assert.Equal(t, "correctness", adapter.Category())
	assert.True(t, adapter.Fast())

	result := adapter.Run(context.Background(), RunInput{Path: dir}, []string{"input.go"})
	require.Equal(t, "failed", result.Tool.Status)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "catalog", result.Findings[0].Tool)
	assert.Equal(t, "go-cor-003", result.Findings[0].Code)
	assert.Equal(t, "correctness", result.Findings[0].Category)
	assert.Equal(t, "warning", result.Findings[0].Severity)
	assert.Equal(t, "Silent error swallowing with blank identifier", result.Findings[0].Title)
	assert.Equal(t, result.Findings[0].Title, result.Findings[0].Message)

	filtered := adapter.Run(context.Background(), RunInput{Path: dir, Category: "security"}, []string{"input.go"})
	require.Equal(t, "passed", filtered.Tool.Status)
	assert.Empty(t, filtered.Findings)
}

func TestAdapter_CatalogAdapter_Bad(t *testing.T) {
	adapter := CatalogAdapter{}
	assert.False(t, adapter.MatchesLanguage([]string{"php"}))
	assert.True(t, adapter.MatchesLanguage([]string{}))
}

func TestAdapter_CatalogAdapter_Ugly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte("package sample\n"), 0o644))

	adapter := CatalogAdapter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := adapter.Run(ctx, RunInput{Path: dir}, []string{"input.go"})
	require.Equal(t, "canceled", result.Tool.Status)
	assert.Empty(t, result.Findings)
}

func installTestCommand(t *testing.T, dir, name string) string {
	t.Helper()

	t.Setenv(adapterTestCommandEnv, name)
	t.Setenv(adapterTestProcessEnv, "1")

	path := filepath.Join(dir, testCommandFilename(name))
	require.NoError(t, linkOrCopyFile(os.Args[0], path))
	return path
}

func testCommandFilename(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func linkOrCopyFile(source, target string) error {
	if err := os.Link(source, target); err == nil {
		return nil
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	info, err := input.Stat()
	if err != nil {
		return err
	}

	output, err := os.Create(target)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(target, info.Mode().Perm())
}

func runAdapterTestCommand(name string, args []string) int {
	if isVersionCommand(args) {
		switch name {
		case "demo-tool":
			fmt.Fprintln(os.Stderr, "demo-tool version 1.2.3")
			return 0
		case "slow-tool":
			fmt.Fprintln(os.Stdout, "slow-tool 9.9.9")
			return 0
		case "json-tool":
			fmt.Fprintln(os.Stdout, "json-tool 1.0.0")
			return 0
		case "mixed-output-tool":
			fmt.Fprintln(os.Stdout, "mixed-output-tool 1.0.0")
			return 0
		}
	}

	switch name {
	case "demo-tool":
		return 0
	case "slow-tool":
		time.Sleep(time.Second)
		return 0
	case "json-tool":
		fmt.Fprintln(os.Stdout, `[{"location":{"path":"src/main.go","start":{"line":12,"column":3}},"message":{"text":"boom"},"rule_id":"X1","severity":"warn"}]`)
		fmt.Fprintln(os.Stderr, "debug noise")
		return 0
	case "mixed-output-tool":
		fmt.Fprintln(os.Stdout, "debug banner")
		fmt.Fprintln(os.Stderr, `[{"location":{"path":"src/secret.go","start":{"line":9,"column":2}},"message":{"text":"secret leaked"},"rule_id":"S1","severity":"error"}]`)
		return 1
	default:
		fmt.Fprintf(os.Stderr, "unknown test command %q\n", name)
		return 2
	}
}

func isVersionCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "--version", "-version", "version":
		return true
	default:
		return false
	}
}

func prependPath(t *testing.T, dir string) {
	t.Helper()

	oldPath := os.Getenv("PATH")
	if oldPath == "" {
		t.Setenv("PATH", dir)
		return
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
}
