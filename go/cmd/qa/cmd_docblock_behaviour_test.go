package qa

import (
	"go/token"

	. "dappco.re/go"
)

// TestCheckDocblockCoverage_Good_CountsAllExportedKinds builds a fixture with a
// documented func, type, const and var plus an exported method, and asserts
// every exported symbol is counted as documented. This exercises checkGenDecl,
// checkTypeSpec, checkValueSpec, mergedDoc, kindFromToken and
// getReceiverTypeName via the public entry point.
func TestCheckDocblockCoverage_Good_CountsAllExportedKinds(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "documented.go"), `package sample

// Service does work.
type Service struct{}

// Run runs the service.
func (s Service) Run() {}

// MaxItems caps the queue.
const MaxItems = 10

// DefaultName is the fallback.
var DefaultName = "x"

// Helper helps.
func Helper() {}
`)
	restoreWorkingDir(t, dir)

	coverage := CheckDocblockCoverage([]string{"."})
	RequireResultOK(t, coverage)
	result := coverage.Value.(*DocblockResult)
	AssertEqual(t, 5, result.Total)
	AssertEqual(t, 5, result.Documented)
	RequireLen(t, result.Missing, 0)
}

// TestCheckDocblockCoverage_Bad_FlagsUndocumentedKinds asserts that an
// undocumented exported func, type, const and var are each reported as missing
// with the correct kind label (func / type / const / var).
func TestCheckDocblockCoverage_Bad_FlagsUndocumentedKinds(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "bare.go"), `package sample

type Widget struct{}

func Build() {}

const Limit = 5

var Counter = 0
`)
	restoreWorkingDir(t, dir)

	coverage := CheckDocblockCoverage([]string{"."})
	RequireResultOK(t, coverage)
	result := coverage.Value.(*DocblockResult)
	AssertEqual(t, 4, result.Total)
	AssertEqual(t, 0, result.Documented)
	RequireLen(t, result.Missing, 4)

	byName := make(map[string]string, len(result.Missing))
	for _, missing := range result.Missing {
		byName[missing.Name] = missing.Kind
	}
	AssertEqual(t, "func", byName["Build"])
	AssertEqual(t, "type", byName["Widget"])
	AssertEqual(t, "const", byName["Limit"])
	AssertEqual(t, "var", byName["Counter"])
}

// TestCheckDocblockCoverage_Ugly_SkipsUnexportedAndUnexportedReceivers asserts
// unexported symbols and methods on unexported receivers are not counted, so a
// file with only those reports zero total symbols.
func TestCheckDocblockCoverage_Ugly_SkipsUnexportedAndUnexportedReceivers(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "internal.go"), `package sample

type service struct{}

func (s service) Run() {}

func helper() {}

const limit = 5
`)
	restoreWorkingDir(t, dir)

	coverage := CheckDocblockCoverage([]string{"."})
	RequireResultOK(t, coverage)
	result := coverage.Value.(*DocblockResult)
	AssertEqual(t, 0, result.Total)
}

// TestCheckDocblockCoverage_Ugly_PointerReceiverResolvesExportedType ensures a
// pointer-receiver method on an exported type is counted (getReceiverTypeName
// must unwrap *ast.StarExpr).
func TestCheckDocblockCoverage_Ugly_PointerReceiverResolvesExportedType(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "ptr.go"), `package sample

type Engine struct{}

func (e *Engine) Start() {}
`)
	restoreWorkingDir(t, dir)

	coverage := CheckDocblockCoverage([]string{"."})
	RequireResultOK(t, coverage)
	result := coverage.Value.(*DocblockResult)
	// Engine (type) + Start (method on exported pointer receiver) = 2 symbols.
	AssertEqual(t, 2, result.Total)
}

// TestExpandRecursivePattern_Good_WalksAndSkips builds a nested tree with a
// skipped vendor dir and a hidden dir, asserting the recursive walk descends
// into real package dirs and skips the rest.
func TestExpandRecursivePattern_Good_WalksAndSkips(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "root.go"), "package sample\n\nfunc Root() {}\n")
	writeTestFile(t, PathJoin(dir, "sub", "child.go"), "package sub\n\nfunc Child() {}\n")
	writeTestFile(t, PathJoin(dir, "vendor", "dep.go"), "package dep\n\nfunc Dep() {}\n")
	writeTestFile(t, PathJoin(dir, ".hidden", "secret.go"), "package secret\n\nfunc Secret() {}\n")
	restoreWorkingDir(t, dir)

	expanded := expandPatterns([]string{"./..."})
	RequireResultOK(t, expanded)
	dirs := expanded.Value.([]string)

	seen := make(map[string]bool)
	for _, found := range dirs {
		seen[found] = true
	}
	AssertTrue(t, seen["."])
	AssertTrue(t, seen[PathJoin(".", "sub")])
	AssertFalse(t, seen[PathJoin(".", "vendor")])
	AssertFalse(t, seen[PathJoin(".", ".hidden")])
}

// TestShouldSkipDocblockDir_GoodBadUgly covers the skip predicate across normal,
// skipped and edge-case directory names.
func TestShouldSkipDocblockDir_GoodBadUgly(t *T) {
	AssertFalse(t, shouldSkipDocblockDir("pkg"))
	AssertFalse(t, shouldSkipDocblockDir("."))
	AssertTrue(t, shouldSkipDocblockDir("vendor"))
	AssertTrue(t, shouldSkipDocblockDir("testdata"))
	AssertTrue(t, shouldSkipDocblockDir(".git"))
}

// TestKindFromToken_GoodBadUgly maps declaration tokens to kind labels.
func TestKindFromToken_GoodBadUgly(t *T) {
	AssertEqual(t, "const", kindFromToken(token.CONST))
	AssertEqual(t, "var", kindFromToken(token.VAR))
	AssertEqual(t, "value", kindFromToken(token.TYPE))
}

// TestRunDocblockCheck_Good_PassesAtThreshold runs the full text-output path on
// a fully documented fixture and asserts success plus a coverage line.
func TestRunDocblockCheck_Good_PassesAtThreshold(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "ok.go"), "package sample\n\n// Alpha is documented.\nfunc Alpha() {}\n")
	restoreWorkingDir(t, dir)

	output := captureStdout(t, func() {
		RequireResultOK(t, RunDocblockCheck([]string{"."}, 100, true, false))
	})
	AssertContains(t, output, "100.0%")
}

// TestRunDocblockCheck_Bad_VerboseListsMissing runs the verbose text path on an
// undocumented fixture and asserts the missing symbol is listed and the run
// fails the threshold.
func TestRunDocblockCheck_Bad_VerboseListsMissing(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "bare.go"), "package sample\n\nfunc Bare() {}\n")
	restoreWorkingDir(t, dir)

	output := captureStdout(t, func() {
		RequireResultError(t, RunDocblockCheck([]string{"."}, 100, true, false))
	})
	AssertContains(t, output, "bare.go")
}
