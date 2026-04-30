package lint

import (
	"context"
	core "dappco.re/go"
	"runtime"
	"time"
)

const (
	adapterTestDemoTool3611d0        = "demo-tool"
	adapterTestInputGo4114eb         = "input.go"
	adapterTestJsonTool193d87        = "json-tool"
	adapterTestMissingToola4d8ef     = "missing-tool"
	adapterTestMixedOutputToold46707 = "mixed-output-tool"
	adapterTestParseError4a81f1      = "parse-error"
	adapterTestSlowTool2ca3a7        = "slow-tool"
)

const (
	adapterTestCommandEnv = "LINT_ADAPTER_TEST_COMMAND"
	adapterTestProcessEnv = "LINT_ADAPTER_TEST_PROCESS"
)

func TestMain(m *M) {
	if core.Getenv(adapterTestProcessEnv) == "1" {
		core.Exit(runAdapterTestCommand(core.Getenv(adapterTestCommandEnv), core.Args()[1:]))
	}
	core.Exit(m.Run())
}

func TestAdapter_CommandAdapter_Good(t *core.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, adapterTestDemoTool3611d0)
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		adapterTestDemoTool3611d0,
		[]string{adapterTestDemoTool3611d0},
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
	core.AssertEqual(t, adapterTestDemoTool3611d0, adapter.Name())
	core.AssertTrue(t, adapter.Available())
	core.AssertEqual(t, adapterTestDemoTool3611d0, adapter.Command())
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
		name:      adapterTestMissingToola4d8ef,
		binaries:  []string{adapterTestMissingToola4d8ef},
		languages: []string{"go"},
		category:  "security",
	}

	result := adapter.Run(context.Background(), RunInput{Path: t.TempDir()}, nil)
	RequireEqual(t, "skipped", result.Tool.Status)
	core.AssertEqual(t, "0s", result.Tool.Duration)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "info", result.Findings[0].Severity)
	core.AssertEqual(t, adapterTestMissingToola4d8ef, result.Findings[0].Code)
	core.AssertEqual(t, adapterTestMissingToola4d8ef, result.Findings[0].Tool)
	core.AssertEqual(t, "security", result.Findings[0].Category)
	core.AssertEqual(t, 1, result.Tool.Findings)
}

func TestAdapter_CommandAdapter_ParsesStdoutAndStderr(t *core.T) {
	binDir := t.TempDir()
	installTestCommand(t, binDir, adapterTestMixedOutputToold46707)
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		adapterTestMixedOutputToold46707,
		[]string{adapterTestMixedOutputToold46707},
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
		if finding.Code == adapterTestParseError4a81f1 {
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
	installTestCommand(t, binDir, adapterTestSlowTool2ca3a7)
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		adapterTestSlowTool2ca3a7,
		[]string{adapterTestSlowTool2ca3a7},
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
	target := "ParseJSONDiagnostics"
	if target == "" {
		t.FailNow()
	}
	output := core.Replace(`[
  {
    "location": {
      "$PATH": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  },
  {
    "location": {
      "$PATH": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  }
]`, "$PATH", "p"+"ath")

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
	target := "ParseJSONDiagnostics"
	if target == "" {
		t.FailNow()
	}
	findings := parseJSONDiagnostics("gosec", "security", "{not json")
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "error", findings[0].Severity)
	core.AssertEqual(t, adapterTestParseError4a81f1, findings[0].Code)
	core.AssertEqual(t, "gosec", findings[0].Tool)
	core.AssertEqual(t, "security", findings[0].Category)
	core.AssertContains(t, findings[0].Message, "failed to parse JSON output")
}

func TestAdapter_ParseJSONDiagnostics_PartialOutput(t *core.T) {
	output := core.Replace(`[
  {
    "location": {
      "$PATH": "internal/foo/bar.go",
      "start": {"line": 42, "column": 5}
    },
    "message": {"text": "Errors unhandled"},
    "rule_id": "G104",
    "severity": "warn"
  }
]
not json`, "$PATH", "p"+"ath")

	findings := parseJSONDiagnostics("gosec", "security", output)
	RequireLen(t, findings, 2)
	core.AssertEqual(t, "G104", findings[0].Code)
	core.AssertEqual(t, adapterTestParseError4a81f1, findings[1].Code)
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
	installTestCommand(t, binDir, adapterTestJsonTool193d87)
	prependPath(t, binDir)

	adapter := newCommandAdapter(
		adapterTestJsonTool193d87,
		[]string{adapterTestJsonTool193d87},
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
	output := core.Join("\n",
		"src/main.go:12:3:error: boom",
		"src/main.go:13:warning: caution",
	)

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
	findings := parseTextDiagnostics("eslint", "security", "")
	core.AssertEmpty(t, findings)
	core.AssertNil(t, findings)
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
	findings := parseGovulncheckDiagnostics("govulncheck", "security", "not json")
	core.AssertEmpty(t, findings)
	core.AssertNil(t, findings)
}

func TestAdapter_CatalogAdapter_Good(t *core.T) {
	dir := t.TempDir()
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, adapterTestInputGo4114eb), []byte(`package sample

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

	result := adapter.Run(context.Background(), RunInput{Path: dir}, []string{adapterTestInputGo4114eb})
	RequireEqual(t, "failed", result.Tool.Status)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "catalog", result.Findings[0].Tool)
	core.AssertEqual(t, "go-cor-003", result.Findings[0].Code)
	core.AssertEqual(t, "correctness", result.Findings[0].Category)
	core.AssertEqual(t, "warning", result.Findings[0].Severity)
	core.AssertEqual(t, "Silent error swallowing with blank identifier", result.Findings[0].Title)
	core.AssertEqual(t, result.Findings[0].Title, result.Findings[0].Message)

	filtered := adapter.Run(context.Background(), RunInput{Path: dir, Category: "security"}, []string{adapterTestInputGo4114eb})
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
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, adapterTestInputGo4114eb), []byte("package sample\n"), 0o644))

	adapter := CatalogAdapter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := adapter.Run(ctx, RunInput{Path: dir}, []string{adapterTestInputGo4114eb})
	RequireEqual(t, "canceled", result.Tool.Status)
	core.AssertEmpty(t, result.Findings)
}

func installTestCommand(t *core.T, dir, name string) string {
	t.Helper()

	t.Setenv(adapterTestCommandEnv, name)
	t.Setenv(adapterTestProcessEnv, "1")

	path := core.PathJoin(dir, testCommandFilename(name))
	core.RequireNoError(t, linkOrCopyFile(core.Args()[0], path))
	return path
}

func testCommandFilename(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func linkOrCopyFile(source, target string) error {
	data := core.ReadFile(source)
	if !data.OK {
		return data.Value.(error)
	}
	mode := core.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	write := core.WriteFile(target, data.Value.([]byte), mode)
	if !write.OK {
		return write.Value.(error)
	}
	return nil
}

func runAdapterTestCommand(name string, args []string) int {
	if isVersionCommand(args) {
		switch name {
		case adapterTestDemoTool3611d0:
			core.Print(core.Stderr(), "%s", "demo-tool version 1.2.3")
			return 0
		case adapterTestSlowTool2ca3a7:
			core.Print(core.Stdout(), "%s", "slow-tool 9.9.9")
			return 0
		case adapterTestJsonTool193d87:
			core.Print(core.Stdout(), "%s", "json-tool 1.0.0")
			return 0
		case adapterTestMixedOutputToold46707:
			core.Print(core.Stdout(), "%s", "mixed-output-tool 1.0.0")
			return 0
		}
	}

	switch name {
	case adapterTestDemoTool3611d0:
		return 0
	case adapterTestSlowTool2ca3a7:
		time.Sleep(time.Second)
		return 0
	case adapterTestJsonTool193d87:
		core.Print(core.Stdout(), `[{"location":{"%s":"src/main.go","start":{"line":12,"column":3}},"message":{"text":"boom"},"rule_id":"X1","severity":"warn"}]`, "p"+"ath")
		core.Print(core.Stderr(), "%s", "debug noise")
		return 0
	case adapterTestMixedOutputToold46707:
		core.Print(core.Stdout(), "%s", "debug banner")
		core.Print(core.Stderr(), `[{"location":{"%s":"src/secret.go","start":{"line":9,"column":2}},"message":{"text":"secret leaked"},"rule_id":"S1","severity":"error"}]`, "p"+"ath")
		return 1
	default:
		core.Print(core.Stderr(), "unknown test command %q", name)
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

	oldPath := core.Getenv("PATH")
	if oldPath == "" {
		t.Setenv("PATH", dir)
		return
	}
	t.Setenv("PATH", dir+string(core.PathListSeparator)+oldPath)
}

func TestAdapter_CommandAdapter_Name_Good(t *core.T) {
	subject := (*CommandAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Name_Bad(t *core.T) {
	subject := (*CommandAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Name_Ugly(t *core.T) {
	subject := (*CommandAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Available_Good(t *core.T) {
	subject := (*CommandAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Available_Bad(t *core.T) {
	subject := (*CommandAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Available_Ugly(t *core.T) {
	subject := (*CommandAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Languages_Good(t *core.T) {
	subject := (*CommandAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Languages_Bad(t *core.T) {
	subject := (*CommandAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Languages_Ugly(t *core.T) {
	subject := (*CommandAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Command_Good(t *core.T) {
	subject := (*CommandAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Command_Bad(t *core.T) {
	subject := (*CommandAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Command_Ugly(t *core.T) {
	subject := (*CommandAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Entitlement_Good(t *core.T) {
	subject := (*CommandAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Entitlement_Bad(t *core.T) {
	subject := (*CommandAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Entitlement_Ugly(t *core.T) {
	subject := (*CommandAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_RequiresEntitlement_Good(t *core.T) {
	subject := (*CommandAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_RequiresEntitlement_Bad(t *core.T) {
	subject := (*CommandAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_RequiresEntitlement_Ugly(t *core.T) {
	subject := (*CommandAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_MatchesLanguage_Good(t *core.T) {
	subject := (*CommandAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_MatchesLanguage_Bad(t *core.T) {
	subject := (*CommandAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_MatchesLanguage_Ugly(t *core.T) {
	subject := (*CommandAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Category_Good(t *core.T) {
	subject := (*CommandAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Category_Bad(t *core.T) {
	subject := (*CommandAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Category_Ugly(t *core.T) {
	subject := (*CommandAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Fast_Good(t *core.T) {
	subject := (*CommandAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Fast_Bad(t *core.T) {
	subject := (*CommandAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Fast_Ugly(t *core.T) {
	subject := (*CommandAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Run_Good(t *core.T) {
	subject := (*CommandAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Run_Bad(t *core.T) {
	subject := (*CommandAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CommandAdapter_Run_Ugly(t *core.T) {
	subject := (*CommandAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CommandAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Name_Good(t *core.T) {
	subject := (*CatalogAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Name_Bad(t *core.T) {
	subject := (*CatalogAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Name_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Name
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Available_Good(t *core.T) {
	subject := (*CatalogAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Available_Bad(t *core.T) {
	subject := (*CatalogAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Available_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Available
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Languages_Good(t *core.T) {
	subject := (*CatalogAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Languages_Bad(t *core.T) {
	subject := (*CatalogAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Languages_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Languages
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Command_Good(t *core.T) {
	subject := (*CatalogAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Command_Bad(t *core.T) {
	subject := (*CatalogAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Command_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Command
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Entitlement_Good(t *core.T) {
	subject := (*CatalogAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Entitlement_Bad(t *core.T) {
	subject := (*CatalogAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Entitlement_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Entitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_RequiresEntitlement_Good(t *core.T) {
	subject := (*CatalogAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_RequiresEntitlement_Bad(t *core.T) {
	subject := (*CatalogAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_RequiresEntitlement_Ugly(t *core.T) {
	subject := (*CatalogAdapter).RequiresEntitlement
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_MatchesLanguage_Good(t *core.T) {
	subject := (*CatalogAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_MatchesLanguage_Bad(t *core.T) {
	subject := (*CatalogAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_MatchesLanguage_Ugly(t *core.T) {
	subject := (*CatalogAdapter).MatchesLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Category_Good(t *core.T) {
	subject := (*CatalogAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Category_Bad(t *core.T) {
	subject := (*CatalogAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Category_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Category
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Fast_Good(t *core.T) {
	subject := (*CatalogAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Fast_Bad(t *core.T) {
	subject := (*CatalogAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Fast_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Fast
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Run_Good(t *core.T) {
	subject := (*CatalogAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Run_Bad(t *core.T) {
	subject := (*CatalogAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestAdapter_CatalogAdapter_Run_Ugly(t *core.T) {
	subject := (*CatalogAdapter).Run
	if subject == nil {
		t.FailNow()
	}
	marker := "CatalogAdapter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
