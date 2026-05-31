package lint

import (
	"time"

	core "dappco.re/go"
)

// TestCombinedOutput_GoodBadUgly joins stdout and stderr, returning the
// non-empty one when the other is blank.
func TestCombinedOutput_GoodBadUgly(t *core.T) {
	core.AssertEqual(t, "out", combinedOutput("out", ""))
	core.AssertEqual(t, "err", combinedOutput("", "err"))
	core.AssertEqual(t, "out\nerr", combinedOutput("out", "err"))
	core.AssertEqual(t, "", combinedOutput("", ""))
}

// TestGoProjectArguments_GoodBadUgly returns the prefix plus explicit files, or
// falls back to ./... when no files are given.
func TestGoProjectArguments_GoodBadUgly(t *core.T) {
	build := goProjectArguments("vet")
	core.AssertEqual(t, []string{"vet", "./..."}, build("", nil))
	core.AssertEqual(t, []string{"vet", "a.go", "b.go"}, build("", []string{"a.go", "b.go"}))
}

// TestRecursiveProjectPathArguments_GoodBadUgly returns "." when no files are
// supplied and the files otherwise.
func TestRecursiveProjectPathArguments_GoodBadUgly(t *core.T) {
	build := recursiveProjectPathArguments("run")
	core.AssertEqual(t, []string{"run", "."}, build("", nil))
	core.AssertEqual(t, []string{"run", "x"}, build("", []string{"x"}))
}

// TestFilePathArguments_GoodBadUgly mirrors the path-or-files behaviour for the
// file-scoped builder.
func TestFilePathArguments_GoodBadUgly(t *core.T) {
	build := filePathArguments("--check")
	core.AssertEqual(t, []string{"--check", "."}, build("", nil))
	core.AssertEqual(t, []string{"--check", "one"}, build("", []string{"one"}))
}

// TestPhpmdArguments_GoodBadUgly comma-joins file targets and always appends
// the json format plus ruleset list.
func TestPhpmdArguments_GoodBadUgly(t *core.T) {
	build := phpmdArguments()
	core.AssertEqual(t, []string{".", "json", "cleancode,codesize,controversial,design,naming,unusedcode"}, build("", nil))
	withFiles := build("", []string{"src/A.php", "src/B.php"})
	core.AssertEqual(t, "src/A.php,src/B.php", withFiles[0])
	core.AssertEqual(t, "json", withFiles[1])
}

// TestNormaliseFindings_Good_FillsToolAndCategory backfills the adapter's tool
// and category on findings that omit them.
func TestNormaliseFindings_Good_FillsToolAndCategory(t *core.T) {
	adapter := CommandAdapter{name: "gosec", category: "security"}
	findings := []Finding{{File: "a.go", Severity: "high"}}
	adapter.normaliseFindings(findings)
	core.AssertEqual(t, "gosec", findings[0].Tool)
	core.AssertEqual(t, "security", findings[0].Category)
}

// TestNormaliseFindings_Ugly_DefaultsSeverityFromCategory assigns the
// category's default severity when a finding has none.
func TestNormaliseFindings_Ugly_DefaultsSeverityFromCategory(t *core.T) {
	adapter := CommandAdapter{name: "gosec", category: "security"}
	findings := []Finding{{File: "a.go"}}
	adapter.normaliseFindings(findings)
	core.AssertEqual(t, "error", findings[0].Severity)
}

// TestNormaliseFindings_Bad_PreservesExplicitToolAndCategory leaves explicit
// tool/category values intact.
func TestNormaliseFindings_Bad_PreservesExplicitToolAndCategory(t *core.T) {
	adapter := CommandAdapter{name: "gosec", category: "security"}
	findings := []Finding{{File: "a.go", Tool: "custom", Category: "quality", Severity: "warning"}}
	adapter.normaliseFindings(findings)
	core.AssertEqual(t, "custom", findings[0].Tool)
	core.AssertEqual(t, "quality", findings[0].Category)
}

// TestCatalogErrorResult_GoodBadUgly stamps a failed status and a single
// error finding carrying the supplied code and message.
func TestCatalogErrorResult_GoodBadUgly(t *core.T) {
	result := catalogErrorResult(AdapterResult{}, time.Now(), "scan-failed", core.Errorf("boom"))
	core.AssertEqual(t, "failed", result.Tool.Status)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "scan-failed", result.Findings[0].Code)
	core.AssertEqual(t, "boom", result.Findings[0].Message)
	core.AssertEqual(t, "error", result.Findings[0].Severity)
	core.AssertEqual(t, 1, result.Tool.Findings)
}

// TestDefaultSeverityForCategory_GoodBadUgly maps categories to default
// severities with a warning fallback.
func TestDefaultSeverityForCategory_GoodBadUgly(t *core.T) {
	core.AssertEqual(t, "error", defaultSeverityForCategory("security"))
	core.AssertEqual(t, "warning", defaultSeverityForCategory("compliance"))
	core.AssertEqual(t, "warning", defaultSeverityForCategory("quality"))
}
