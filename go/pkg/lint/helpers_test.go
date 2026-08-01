package lint

import (
	core "dappco.re/go"
)

// TestPassesThreshold_Good_ErrorGate passes only when there are no errors at
// the default/error threshold.
func TestPassesThreshold_Good_ErrorGate(t *core.T) {
	core.AssertTrue(t, passesThreshold(Summary{Errors: 0, Warnings: 5}, ""))
	core.AssertTrue(t, passesThreshold(Summary{Errors: 0, Warnings: 5}, "error"))
	core.AssertFalse(t, passesThreshold(Summary{Errors: 1}, "error"))
}

// TestPassesThreshold_Bad_WarningGate fails when warnings are present at the
// warning threshold.
func TestPassesThreshold_Bad_WarningGate(t *core.T) {
	core.AssertFalse(t, passesThreshold(Summary{Warnings: 1}, "warning"))
	core.AssertTrue(t, passesThreshold(Summary{Errors: 0, Warnings: 0}, "warning"))
}

// TestPassesThreshold_Ugly_InfoGateAndUnknown treats info as total==0 and an
// unknown threshold as the error gate.
func TestPassesThreshold_Ugly_InfoGateAndUnknown(t *core.T) {
	core.AssertFalse(t, passesThreshold(Summary{Total: 1}, "info"))
	core.AssertTrue(t, passesThreshold(Summary{Total: 0}, "info"))
	core.AssertTrue(t, passesThreshold(Summary{Errors: 0}, "mystery"))
}

// TestSortFindings_Good_OrdersByFileLineColumnToolCode sorts findings by the
// full key tuple.
func TestSortFindings_Good_OrdersByFileLineColumnToolCode(t *core.T) {
	findings := []Finding{
		{File: "b.go", Line: 1},
		{File: "a.go", Line: 2},
		{File: "a.go", Line: 1, Column: 5},
		{File: "a.go", Line: 1, Column: 2},
	}
	sortFindings(findings)
	core.AssertEqual(t, "a.go", findings[0].File)
	core.AssertEqual(t, 1, findings[0].Line)
	core.AssertEqual(t, 2, findings[0].Column)
	core.AssertEqual(t, 5, findings[1].Column)
	core.AssertEqual(t, 2, findings[2].Line)
	core.AssertEqual(t, "b.go", findings[3].File)
}

// TestSortFindings_Ugly_TieBreaksOnToolThenCode disambiguates identical
// positions by tool then code.
func TestSortFindings_Ugly_TieBreaksOnToolThenCode(t *core.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Tool: "vet", Code: "z"},
		{File: "a.go", Line: 1, Tool: "gosec", Code: "b"},
		{File: "a.go", Line: 1, Tool: "gosec", Code: "a"},
	}
	sortFindings(findings)
	core.AssertEqual(t, "gosec", findings[0].Tool)
	core.AssertEqual(t, "a", findings[0].Code)
	core.AssertEqual(t, "b", findings[1].Code)
	core.AssertEqual(t, "vet", findings[2].Tool)
}

// TestShouldIncludeLanguageGroups_GoodBadUgly includes language groups by
// default and for non-security categories, and excludes them when only
// security/compliance is requested.
func TestShouldIncludeLanguageGroups_GoodBadUgly(t *core.T) {
	core.AssertTrue(t, shouldIncludeLanguageGroups(nil))
	core.AssertTrue(t, shouldIncludeLanguageGroups([]string{"quality"}))
	core.AssertFalse(t, shouldIncludeLanguageGroups([]string{"security"}))
	core.AssertFalse(t, shouldIncludeLanguageGroups([]string{"security", "compliance"}))
}

// TestGroupForLanguage_GoodBadUgly maps known languages to their tool group and
// returns nil for anything unmapped.
func TestGroupForLanguage_GoodBadUgly(t *core.T) {
	groups := ToolGroups{
		Go:     []string{"go-tool"},
		PHP:    []string{"php-tool"},
		JS:     []string{"js-tool"},
		TS:     []string{"ts-tool"},
		Python: []string{"py-tool"},
		Infra:  []string{"infra-tool"},
	}
	core.AssertEqual(t, []string{"go-tool"}, groupForLanguage(groups, "go"))
	core.AssertEqual(t, []string{"php-tool"}, groupForLanguage(groups, "php"))
	core.AssertEqual(t, []string{"js-tool"}, groupForLanguage(groups, "js"))
	core.AssertEqual(t, []string{"ts-tool"}, groupForLanguage(groups, "ts"))
	core.AssertEqual(t, []string{"py-tool"}, groupForLanguage(groups, "python"))
	core.AssertEqual(t, []string{"infra-tool"}, groupForLanguage(groups, "yaml"))
	core.AssertNil(t, groupForLanguage(groups, "cobol"))
}

// TestSarifLevel_GoodBadUgly maps severities to SARIF levels with a warning
// fallback.
func TestSarifLevel_GoodBadUgly(t *core.T) {
	core.AssertEqual(t, "error", sarifLevel("error"))
	core.AssertEqual(t, "warning", sarifLevel("warning"))
	core.AssertEqual(t, "note", sarifLevel("info"))
	core.AssertEqual(t, "warning", sarifLevel("bananas"))
}

// TestLanguagesFromRules_GoodBadUgly collects a deduplicated, sorted set of
// languages across rules.
func TestLanguagesFromRules_GoodBadUgly(t *core.T) {
	rules := []Rule{
		{Languages: []string{"go", "php"}},
		{Languages: []string{"go"}},
		{Languages: []string{"js"}},
	}
	core.AssertEqual(t, []string{"go", "js", "php"}, languagesFromRules(rules))
	core.AssertEmpty(t, languagesFromRules(nil))
}

// TestParseVulnerabilityLine_Good_ParsesBlock parses a govulncheck text block
// into a Vulnerability with ID, package, version and description.
func TestParseVulnerabilityLine_Good_ParsesBlock(t *core.T) {
	var cur Vulnerability
	var inBlock bool
	var vulns []Vulnerability

	// The parser takes the second whitespace field of the header line as the ID
	// ("Vulnerability" "#1:" "GO-..."), matching govulncheck's "Vulnerability #N: ID" layout.
	lines := []string{
		"Vulnerability #1: GO-2024-0001",
		"A serious problem in the parser.",
		"  More detail.",
		"    Package: example.com/dep",
		"    Found in version: v1.2.3",
		"",
	}
	for _, line := range lines {
		parseVulnerabilityLine(line, &cur, &inBlock, &vulns)
	}

	RequireLen(t, vulns, 1)
	core.AssertEqual(t, "#1:", vulns[0].ID)
	core.AssertEqual(t, "example.com/dep", vulns[0].Package)
	core.AssertEqual(t, "v1.2.3", vulns[0].Version)
	core.AssertEqual(t, "A serious problem in the parser.", vulns[0].Description)
}

// TestParseVulnerabilityLine_Ugly_BackToBackBlocksFlushPrevious starts a second
// block while the first is open, flushing the first vulnerability.
func TestParseVulnerabilityLine_Ugly_BackToBackBlocksFlushPrevious(t *core.T) {
	var cur Vulnerability
	var inBlock bool
	var vulns []Vulnerability

	for _, line := range []string{
		"Vulnerability #1: GO-A",
		"    Package: dep-a",
		"Vulnerability #2: GO-B",
		"    Package: dep-b",
	} {
		parseVulnerabilityLine(line, &cur, &inBlock, &vulns)
	}
	// First block flushed when the second started; the second is still current.
	// IDs are the "#N:" field per the header layout (see _Good test).
	RequireLen(t, vulns, 1)
	core.AssertEqual(t, "#1:", vulns[0].ID)
	core.AssertEqual(t, "dep-a", vulns[0].Package)
	core.AssertEqual(t, "#2:", cur.ID)
}

// TestParseVulnerabilityLine_Bad_OutsideBlockIgnored ignores body lines that
// appear before any vulnerability header.
func TestParseVulnerabilityLine_Bad_OutsideBlockIgnored(t *core.T) {
	var cur Vulnerability
	var inBlock bool
	var vulns []Vulnerability

	parseVulnerabilityLine("    Package: orphan", &cur, &inBlock, &vulns)
	core.AssertEqual(t, "", cur.Package)
	core.AssertEmpty(t, vulns)
}

// TestWriteDefaultConfig_Good_WritesYAML writes the default config to a fresh
// directory and reports the target path.
func TestWriteDefaultConfig_Good_WritesYAML(t *core.T) {
	dir := t.TempDir()
	result := NewService().WriteDefaultConfig(dir, false)
	RequireResultOK(t, result)
	core.AssertContains(t, result.Value.(string), DefaultConfigPath)

	content := core.ReadFile(core.JoinPath(dir, DefaultConfigPath))
	RequireResultOK(t, content)
	core.AssertNotEmpty(t, string(content.Value.([]byte)))
}

// TestWriteDefaultConfig_Bad_ExistingWithoutForceFails refuses to overwrite an
// existing config when force is false.
func TestWriteDefaultConfig_Bad_ExistingWithoutForceFails(t *core.T) {
	dir := t.TempDir()
	RequireResultOK(t, NewService().WriteDefaultConfig(dir, false))
	result := NewService().WriteDefaultConfig(dir, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already exists")
}

// TestWriteDefaultConfig_Ugly_ForceOverwrites overwrites an existing config when
// force is true.
func TestWriteDefaultConfig_Ugly_ForceOverwrites(t *core.T) {
	dir := t.TempDir()
	RequireResultOK(t, NewService().WriteDefaultConfig(dir, false))
	RequireResultOK(t, NewService().WriteDefaultConfig(dir, true))
}

// TestRegister_Good_BuildsCoreAttachedService builds a Core with the lint
// service via Register and runs the startup/shutdown lifecycle.
func TestRegister_Good_BuildsCoreAttachedService(t *core.T) {
	c := core.New(core.WithService(Register))
	RequireNotNil(t, c)

	serviceResult := Register(c)
	RequireResultOK(t, serviceResult)
	svc := serviceResult.Value.(*Service)
	RequireNotNil(t, svc)

	RequireResultOK(t, svc.OnStartup(t.Context()))
	RequireResultOK(t, svc.OnShutdown(t.Context()))
}

// TestOnStartup_Ugly_NilRuntimeIsNoOp returns OK for a library-constructed
// service with no attached Core.
func TestOnStartup_Ugly_NilRuntimeIsNoOp(t *core.T) {
	svc := NewService()
	RequireResultOK(t, svc.OnStartup(t.Context()))
	RequireResultOK(t, svc.OnShutdown(t.Context()))
}
