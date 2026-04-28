package lint

import (
	"context"
	core "dappco.re/go"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	adapterTestCommandEnv = "LINT_ADAPTER_TEST_COMMAND"
	adapterTestProcessEnv = "LINT_ADAPTER_TEST_PROCESS"
)

func TestMain(m *M) {
	if os.Getenv(adapterTestProcessEnv) == "1" {
		os.Exit(runAdapterTestCommand(os.Getenv(adapterTestCommandEnv), os.Args[1:]))
	}
	os.Exit(m.Run())
}

func TestAdapter_CommandAdapter_Good(t *core.T) {
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
	core.AssertEqual(t, "demo-tool", adapter.Name())
	core.AssertTrue(t, adapter.Available())
	core.AssertEqual(t, "demo-tool", adapter.Command())
	core.AssertEqual(t, "lint.security", adapter.Entitlement())
	core.AssertTrue(t, adapter.RequiresEntitlement())
	core.AssertTrue(t, adapter.MatchesLanguage([]string{"go"}))
	core.AssertTrue(t, adapter.MatchesLanguage([]string{"security"}))
	core.AssertTrue(t, adapter.MatchesLanguage(nil))
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"rust"}))
	core.AssertEqual(t, "security", adapter.Category())
	core.AssertTrue(t, adapter.Fast())
	core.AssertEqual(t, []string{"go"}, langs)

	langs[0] = "mutated"
	core.AssertEqual(t, []string{"go"}, adapter.Languages())

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	RequireEqual(t, "passed", result.Tool.Status)
	core.AssertEqual(t, "demo-tool version 1.2.3", result.Tool.Version)
	core.AssertEqual(t, 0, result.Tool.Findings)
	core.AssertEmpty(t, result.Findings)
}

func TestAdapter_CommandAdapter_Bad(t *core.T) {
	adapter := CommandAdapter{
		name:      "missing-tool",
		binaries:  []string{"missing-tool"},
		languages: []string{"go"},
		category:  "security",
	}

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	RequireEqual(t, "skipped", result.Tool.Status)
	core.AssertEqual(t, "0s", result.Tool.Duration)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "info", result.Findings[0].Severity)
	core.AssertEqual(t, "missing-tool", result.Findings[0].Code)
	core.AssertEqual(t, "missing-tool", result.Findings[0].Tool)
	core.AssertEqual(t, "security", result.Findings[0].Category)
	core.AssertEqual(t, 1, result.Tool.Findings)
}

func TestAdapter_CommandAdapter_ParsesStdoutAndStderr(t *core.T) {
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

func TestAdapter_CommandAdapter_Ugly(t *core.T) {
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
	RequireEqual(t, "timeout", result.Tool.Status)
	core.AssertEqual(t, "slow-tool 9.9.9", result.Tool.Version)
	core.AssertEmpty(t, result.Findings)
}

func TestAdapter_ParseJSONDiagnostics_Good(t *core.T) {
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
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "gosec", findings[0].Tool)
	core.AssertEqual(t, "internal/foo/bar.go", findings[0].File)
	core.AssertEqual(t, 42, findings[0].Line)
	core.AssertEqual(t, 5, findings[0].Column)
	core.AssertEqual(t, "G104", findings[0].Code)
	core.AssertEqual(t, "Errors unhandled", findings[0].Message)
	core.AssertEqual(t, "warning", findings[0].Severity)
	core.AssertEqual(t, "security", findings[0].Category)
}

func TestAdapter_ParseJSONDiagnostics_Bad(t *core.T) {
	findings := parseJSONDiagnostics("gosec", "security", "{not json")
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "error", findings[0].Severity)
	core.AssertEqual(t, "parse-error", findings[0].Code)
	core.AssertEqual(t, "gosec", findings[0].Tool)
	core.AssertEqual(t, "security", findings[0].Category)
	core.AssertContains(t, findings[0].Message, "failed to parse JSON output")
}

func TestAdapter_ParseJSONDiagnostics_PartialOutput(t *core.T) {
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
	RequireLen(t, findings, 2)
	core.AssertEqual(t, "G104", findings[0].Code)
	core.AssertEqual(t, "parse-error", findings[1].Code)
}

func TestAdapter_ParseJSONDiagnostics_Ugly(t *core.T) {
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
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "scanner", findings[0].Tool)
	core.AssertEqual(t, "src/main.go", findings[0].File)
	core.AssertEqual(t, 7, findings[0].Line)
	core.AssertEqual(t, 3, findings[0].Column)
	core.AssertEqual(t, "ABC123", findings[0].Code)
	core.AssertEqual(t, "Potential issue", findings[0].Message)
	core.AssertEqual(t, "error", findings[0].Severity)
	core.AssertEqual(t, "security", findings[0].Category)
}

func TestAdapter_CommandAdapter_JSONStdoutIgnoresStderr(t *core.T) {
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
	RequireEqual(t, "failed", result.Tool.Status)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "X1", result.Findings[0].Code)
	core.AssertEqual(t, "src/main.go", result.Findings[0].File)
	core.AssertEqual(t, "boom", result.Findings[0].Message)
	core.AssertEqual(t, "warning", result.Findings[0].Severity)
}

func TestAdapter_ParseTextDiagnostics_Good(t *core.T) {
	output := strings.Join([]string{
		"src/main.go:12:3:error: boom",
		"src/main.go:13:warning: caution",
	}, "\n")

	findings := parseTextDiagnostics("eslint", "security", output)
	RequireLen(t, findings, 2)
	core.AssertEqual(t, 12, findings[0].Line)
	core.AssertEqual(t, 3, findings[0].Column)
	core.AssertEqual(t, "error", findings[0].Severity)
	core.AssertEqual(t, 13, findings[1].Line)
	core.AssertEqual(t, 0, findings[1].Column)
	core.AssertEqual(t, "warning", findings[1].Severity)
}

func TestAdapter_ParseTextDiagnostics_Bad(t *core.T) {
	core.AssertEmpty(t, parseTextDiagnostics("eslint", "security", ""))
}

func TestAdapter_ParseTextDiagnostics_Ugly(t *core.T) {
	findings := parseTextDiagnostics("eslint", "security", "not parseable")
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "diagnostic", findings[0].Code)
	core.AssertEqual(t, "not parseable", findings[0].Message)
	core.AssertEqual(t, "error", findings[0].Severity)
}

func TestAdapter_ParseGovulncheckDiagnostics_Good(t *core.T) {
	output := `{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
{"osv":{"id":"GO-2024-1234","summary":"Buffer overflow in foo","aliases":["CVE-2024-1234"],"affected":[{"ranges":[{"events":[{"fixed":"1.2.3"}]}]}]}}
{"finding":{"osv":"GO-2024-1234","trace":[{"package":"example.com/foo","function":"Bar"}]}}
`

	findings := parseGovulncheckDiagnostics("govulncheck", "security", output)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "govulncheck", findings[0].Tool)
	core.AssertEqual(t, "example.com/foo", findings[0].File)
	core.AssertEqual(t, "GO-2024-1234", findings[0].Code)
	core.AssertEqual(t, "Buffer overflow in foo", findings[0].Message)
	core.AssertEqual(t, "error", findings[0].Severity)
	core.AssertEqual(t, "security", findings[0].Category)
}

func TestAdapter_ParseGovulncheckDiagnostics_Bad(t *core.T) {
	core.AssertEmpty(t, parseGovulncheckDiagnostics("govulncheck", "security", "not json"))
}

func TestAdapter_CatalogAdapter_Good(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	adapter := CatalogAdapter{}
	core.AssertEqual(t, "catalog", adapter.Name())
	core.AssertTrue(t, adapter.Available())
	core.AssertEqual(t, []string{"go"}, adapter.Languages())
	core.AssertEqual(t, "catalog", adapter.Command())
	core.AssertEmpty(t, adapter.Entitlement())
	core.AssertFalse(t, adapter.RequiresEntitlement())
	core.AssertTrue(t, adapter.MatchesLanguage(nil))
	core.AssertTrue(t, adapter.MatchesLanguage([]string{"go"}))
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"rust"}))
	core.AssertEqual(t, "correctness", adapter.Category())
	core.AssertTrue(t, adapter.Fast())

	result := adapter.Run(context.Background(), RunInput{Path: dir}, []string{"input.go"})
	RequireEqual(t, "failed", result.Tool.Status)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "catalog", result.Findings[0].Tool)
	core.AssertEqual(t, "go-cor-003", result.Findings[0].Code)
	core.AssertEqual(t, "correctness", result.Findings[0].Category)
	core.AssertEqual(t, "warning", result.Findings[0].Severity)
	core.AssertEqual(t, "Silent error swallowing with blank identifier", result.Findings[0].Title)
	core.AssertEqual(t, result.Findings[0].Title, result.Findings[0].Message)

	filtered := adapter.Run(context.Background(), RunInput{Path: dir, Category: "security"}, []string{"input.go"})
	RequireEqual(t, "passed", filtered.Tool.Status)
	core.AssertEmpty(t, filtered.Findings)
}

func TestAdapter_CatalogAdapter_Bad(t *core.T) {
	adapter := CatalogAdapter{}
	core.AssertFalse(t, adapter.MatchesLanguage([]string{"php"}))
	core.AssertTrue(t, adapter.MatchesLanguage([]string{}))
}

func TestAdapter_CatalogAdapter_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "input.go"), []byte("package sample\n"), 0o644))

	adapter := CatalogAdapter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := adapter.Run(ctx, RunInput{Path: dir}, []string{"input.go"})
	RequireEqual(t, "canceled", result.Tool.Status)
	core.AssertEmpty(t, result.Findings)
}

func installTestCommand(t *core.T, dir, name string) string {
	t.Helper()

	t.Setenv(adapterTestCommandEnv, name)
	t.Setenv(adapterTestProcessEnv, "1")

	path := filepath.Join(dir, testCommandFilename(name))
	core.RequireNoError(t, linkOrCopyFile(os.Args[0], path))
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

func prependPath(t *core.T, dir string) {
	t.Helper()

	oldPath := os.Getenv("PATH")
	if oldPath == "" {
		t.Setenv("PATH", dir)
		return
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
}
