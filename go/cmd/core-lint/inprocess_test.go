package main

import (
	"bytes"

	. "dappco.re/go"

	lintpkg "dappco.re/go/lint/pkg/lint"
)

// newWriters returns commandWriters backed by in-memory buffers so dispatch
// functions can be exercised in-process (the subprocess CLI tests in
// main_test.go validate the built artefact, but report 0% instrumented
// coverage because the binary runs out-of-process).
//
//	w, stdout, stderr := newWriters()
//	runRoot([]string{"detect", dir}, w)
//	_ = stdout.String()
func newWriters() (commandWriters, *bytes.Buffer, *bytes.Buffer) {
	stdout := NewBuffer()
	stderr := NewBuffer()
	return commandWriters{stdout: stdout, stderr: stderr}, stdout, stderr
}

// firstToken returns the leading whitespace-delimited token of s, used to pull
// a rule ID out of a catalog-list line ("%-14s [...] title").
//
//	id := firstToken("go-cor-003 [high]   Title")  // "go-cor-003"
func firstToken(s string) string {
	for index, value := range s {
		if value == ' ' || value == '\t' || value == '\n' {
			return s[:index]
		}
	}
	return s
}

func TestRunRoot_Good_PrintsHelpWhenNoArgs(t *T) {
	w, stdout, _ := newWriters()
	result := runRoot(nil, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "Commands: run, detect, tools, init, hook, lint, catalog")
}

func TestRunRoot_Good_DispatchesDetect(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))

	w, stdout, _ := newWriters()
	result := runRoot([]string{"detect", dir}, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "go")
}

func TestRunRoot_Bad_UnknownCommandFails(t *T) {
	w, _, _ := newWriters()
	result := runRoot([]string{"frobnicate"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown command frobnicate")
}

func TestRunLintNamespace_Good_PrintsHelpWhenEmpty(t *T) {
	w, stdout, _ := newWriters()
	result := runLintNamespace(nil, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "Commands:")
}

func TestRunLintNamespace_Bad_UnknownSubcommandFails(t *T) {
	w, _, _ := newWriters()
	result := runLintNamespace([]string{"wibble"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown lint command wibble")
}

func TestRunDetectCommand_Good_JSONOutput(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "package.json"), []byte("{}\n"), 0o644))

	w, stdout, _ := newWriters()
	result := runDetectCommand([]string{mainTestOutput231216, "json", dir}, w)
	AssertTrue(t, result.OK, result.Error())

	var languages []string
	RequireResultOK(t, JSONUnmarshal([]byte(stdout.String()), &languages))
	AssertEqual(t, []string{"go", "js"}, languages)
}

func TestRunDetectCommand_Ugly_UnsupportedOutputFails(t *T) {
	dir := t.TempDir()
	w, _, _ := newWriters()
	result := runDetectCommand([]string{mainTestOutput231216, "yaml", dir}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unsupported output format yaml")
}

func TestRunToolsCommand_Good_TextLists(t *T) {
	w, stdout, _ := newWriters()
	result := runToolsCommand([]string{"--lang", "go"}, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "langs=go")
}

func TestRunToolsCommand_Good_JSONOutput(t *T) {
	w, stdout, _ := newWriters()
	result := runToolsCommand([]string{mainTestOutput231216, "json"}, w)
	AssertTrue(t, result.OK, result.Error())

	var tools []lintpkg.ToolInfo
	RequireResultOK(t, JSONUnmarshal([]byte(stdout.String()), &tools))
	RequireNotEmpty(t, tools)
}

func TestRunToolsCommand_Ugly_UnsupportedOutputFails(t *T) {
	w, _, _ := newWriters()
	result := runToolsCommand([]string{mainTestOutput231216, "csv"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unsupported output format csv")
}

func TestRunInitCommand_Good_WritesConfig(t *T) {
	dir := t.TempDir()
	w, stdout, _ := newWriters()
	result := runInitCommand([]string{dir}, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), ".core/lint.yaml")

	content := ReadFile(PathJoin(dir, ".core", "lint.yaml"))
	RequireResultOK(t, content)
	AssertContains(t, string(content.Value.([]byte)), "fail_on: error")
}

func TestRunInitCommand_Bad_ExistingConfigWithoutForceFails(t *T) {
	dir := t.TempDir()
	w, _, _ := newWriters()
	RequireResultOK(t, runInitCommand([]string{dir}, w))

	w2, _, _ := newWriters()
	result := runInitCommand([]string{dir}, w2)
	AssertFalse(t, result.OK)
}

func TestRunHookCommand_Bad_NoSubcommandFails(t *T) {
	w, _, _ := newWriters()
	result := runHookCommand(nil, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "subcommand required")
}

func TestRunHookCommand_Ugly_UnknownSubcommandFails(t *T) {
	dir := t.TempDir()
	w, _, _ := newWriters()
	result := runHookCommand([]string{"frob", dir}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown hook command frob")
}

func TestRunCatalogNamespace_Bad_NoSubcommandFails(t *T) {
	w, _, _ := newWriters()
	result := runCatalogNamespace(nil, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "subcommand required")
}

func TestRunCatalogNamespace_Ugly_UnknownSubcommandFails(t *T) {
	w, _, _ := newWriters()
	result := runCatalogNamespace([]string{"explode"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unknown catalog command explode")
}

func TestRunCatalogList_Good_ListsRules(t *T) {
	w, stdout, stderr := newWriters()
	result := runCatalogNamespace([]string{"list"}, w)
	AssertTrue(t, result.OK, result.Error())
	RequireNotEmpty(t, stdout.String())
	AssertContains(t, stderr.String(), "rule(s)")
}

func TestRunCatalogList_Good_LanguageFilterNarrows(t *T) {
	w, stdout, _ := newWriters()
	result := runCatalogNamespace([]string{"list", "--lang", "go"}, w)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "go-")
}

func TestRunCatalogShow_Good_RendersRuleJSON(t *T) {
	w, listOut, _ := newWriters()
	RequireResultOK(t, runCatalogNamespace([]string{"list", "--lang", "go"}, w))
	firstID := firstToken(listOut.String())

	w2, stdout, _ := newWriters()
	result := runCatalogNamespace([]string{"show", firstID}, w2)
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), firstID)
}

func TestRunCatalogShow_Bad_NoIDFails(t *T) {
	w, _, _ := newWriters()
	result := runCatalogNamespace([]string{"show"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "rule ID required")
}

func TestRunCatalogShow_Ugly_UnknownRuleFails(t *T) {
	w, _, _ := newWriters()
	result := runCatalogNamespace([]string{"show", "no-such-rule-xyz"}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "not found")
}

func TestRunCheckCommand_Good_TextFindings(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	stdout := NewBuffer()
	stderr := NewBuffer()
	result := runCheckCommand(stdout, stderr, []string{dir}, checkOptions{format: "text"})
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stdout.String(), "finding(s)")
}

func TestRunCheckCommand_Good_JSONFindings(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))

	stdout := NewBuffer()
	stderr := NewBuffer()
	result := runCheckCommand(stdout, stderr, []string{dir}, checkOptions{format: "json"})
	AssertTrue(t, result.OK, result.Error())

	var findings []lintpkg.Finding
	RequireResultOK(t, JSONUnmarshal([]byte(stdout.String()), &findings))
	RequireNotEmpty(t, findings)
}

func TestRunCheckCommand_Bad_UnknownLanguageReportsNoRules(t *T) {
	dir := t.TempDir()
	stdout := NewBuffer()
	stderr := NewBuffer()
	result := runCheckCommand(stdout, stderr, []string{dir}, checkOptions{format: "text", language: "cobol"})
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stderr.String(), "no rules for language")
}

func TestRunCheckCommand_Ugly_UnknownSeverityReportsNoRules(t *T) {
	dir := t.TempDir()
	stdout := NewBuffer()
	stderr := NewBuffer()
	result := runCheckCommand(stdout, stderr, []string{dir}, checkOptions{format: "text", severity: "apocalyptic"})
	AssertTrue(t, result.OK, result.Error())
	AssertContains(t, stderr.String(), "no rules at severity")
}

func TestRunCheckCommand_Ugly_MissingPathFails(t *T) {
	stdout := NewBuffer()
	stderr := NewBuffer()
	result := runCheckCommand(stdout, stderr, []string{"no-such-path-xyz"}, checkOptions{format: "text"})
	AssertFalse(t, result.OK)
}

func TestRunRunCommand_Good_CleanCodePasses(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestCleanGob56c68), []byte("package sample\n\nfunc Clean() {}\n"), 0o644))
	t.Setenv("PATH", t.TempDir())

	w, _, _ := newWriters()
	result := runRunCommand("run", []string{mainTestOutput231216, "json", "--files", mainTestCleanGob56c68, dir}, lintpkg.RunInput{}, w)
	AssertTrue(t, result.OK, result.Error())
}

func TestRunRunCommand_Bad_FindingsFailOnTriggers(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	RequireResultOK(t, WriteFile(PathJoin(dir, "input.go"), []byte(`package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	_ = svc.Process("data")
}
`), 0o644))
	t.Setenv("PATH", t.TempDir())

	w, _, _ := newWriters()
	result := runRunCommand("run", []string{mainTestOutput231216, "json", "--fail-on", "warning", dir}, lintpkg.RunInput{}, w)
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "lint failed (fail-on=warning)")
}

func TestRunRunCommand_Ugly_UnsupportedOutputFails(t *T) {
	dir := t.TempDir()
	RequireResultOK(t, WriteFile(PathJoin(dir, mainTestGoMod3af676), []byte(mainTestModuleExampleComTeste3feb4), 0o644))
	t.Setenv("PATH", t.TempDir())

	w, _, _ := newWriters()
	result := runRunCommand("run", []string{mainTestOutput231216, "toml", dir}, lintpkg.RunInput{}, w)
	AssertFalse(t, result.OK)
}

func TestWriteReport_Good_AllFormats(t *T) {
	report := lintpkg.Report{Summary: lintpkg.Summary{Passed: true}}
	// github output only emits annotation lines per finding; with no findings
	// the writer succeeds but produces no bytes, so it is not asserted non-empty.
	for _, format := range []string{"json", "text", "sarif"} {
		buffer := NewBuffer()
		result := writeReport(buffer, format, report)
		AssertTrue(t, result.OK, format+": "+result.Error())
		RequireNotEmpty(t, buffer.String(), format)
	}
	githubBuffer := NewBuffer()
	AssertTrue(t, writeReport(githubBuffer, "github", report).OK)
}

func TestWriteReport_Ugly_UnknownFormatFails(t *T) {
	buffer := NewBuffer()
	result := writeReport(buffer, "xml", lintpkg.Report{})
	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "unsupported output format xml")
}

func TestParseArgs_Good_SeparatesFlagsAndPositionals(t *T) {
	parsed := parseArgs([]string{"--output", "json", "-l", "go", "path/one", "path/two"})
	AssertEqual(t, "json", firstFlag(parsed, "output", "o", "text"))
	AssertEqual(t, "go", firstFlag(parsed, "lang", "l", ""))
	AssertEqual(t, []string{"path/one", "path/two"}, parsed.positionals)
}

func TestParseArgs_Good_BareFlagBecomesTrue(t *T) {
	parsed := parseArgs([]string{"--hook"})
	AssertTrue(t, boolFlag(parsed, "hook", false))
}

func TestFirstFlag_Ugly_FallsBackWhenAbsent(t *T) {
	parsed := parseArgs(nil)
	AssertEqual(t, "fallback", firstFlag(parsed, "missing", "m", "fallback"))
}

func TestBoolFlag_Good_RecognisesTruthyValues(t *T) {
	for _, value := range []string{"true", "1", "yes"} {
		parsed := parseArgs([]string{"--ci=" + value})
		AssertTrue(t, boolFlag(parsed, "ci", false), value)
	}
	parsed := parseArgs([]string{"--ci=no"})
	AssertFalse(t, boolFlag(parsed, "ci", true))
}

func TestRunInputFromArgs_Good_AppliesFlagsAndDefaults(t *T) {
	input := runInputFromArgs(
		[]string{"--output", "json", "--config", "cfg.yaml", "--fail-on", "error", "--ci=true", "myproject"},
		lintpkg.RunInput{Lang: "go"},
	)
	AssertEqual(t, "json", input.Output)
	AssertEqual(t, "cfg.yaml", input.Config)
	AssertEqual(t, "error", input.FailOn)
	AssertEqual(t, "go", input.Lang)
	AssertTrue(t, input.CI)
	AssertEqual(t, "myproject", input.Path)
}

func TestRunInputFromArgs_Ugly_DefaultsPathToDot(t *T) {
	input := runInputFromArgs(nil, lintpkg.RunInput{})
	AssertEqual(t, ".", input.Path)
}

func TestSortedCatalogRules_Good_OrdersBySeverityThenID(t *T) {
	rules := []lintpkg.Rule{
		{ID: "z", Severity: "low"},
		{ID: "a", Severity: "low"},
		{ID: "m", Severity: "high"},
	}
	sorted := sortedCatalogRules(rules)
	AssertEqual(t, "m", sorted[0].ID)
	AssertEqual(t, "a", sorted[1].ID)
	AssertEqual(t, "z", sorted[2].ID)
}

func TestWriteCatalogSummary_Good_RendersSeverityBreakdown(t *T) {
	findings := []lintpkg.Finding{
		{Severity: "high"},
		{Severity: "low"},
		{Severity: "low"},
	}
	buffer := NewBuffer()
	result := writeCatalogSummary(buffer, findings)
	AssertTrue(t, result.OK, result.Error())
	text := buffer.String()
	AssertContains(t, text, "3 finding(s)")
	AssertContains(t, text, "1 high")
	AssertContains(t, text, "2 low")
}
