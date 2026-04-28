package lint

import (
	core "dappco.re/go"
	"embed"
	"os"
	"path/filepath"
)

//go:embed testdata/catalog/*.yaml
var testCatalogFS embed.FS

func TestLoadDir_Good(t *core.T) {
	// Use the real catalog/ directory at the repo root.
	dir := findCatalogDir(t)
	cat, err := LoadDir(dir)
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, cat.Rules)

	// All 18 rules should load (6 security + 7 correctness + 5 modernise).
	core.AssertLen(t, cat.Rules, 18)

	// Verify we can find a rule from each file.
	core.AssertNotNil(t, cat.ByID("go-sec-001"))
	core.AssertNotNil(t, cat.ByID("go-cor-001"))
	core.AssertNotNil(t, cat.ByID("go-mod-001"))
}

func TestLoadDir_SortsFilesDeterministically(t *core.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "z.yaml"), []byte(`- id: z-rule
  title: "Z rule"
  severity: info
  languages: [go]
  pattern: 'z'
  fix: "z"
  detection: regex
  auto_fixable: false
`), 0o644)
	core.RequireNoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(`- id: a-rule
  title: "A rule"
  severity: info
  languages: [go]
  pattern: 'a'
  fix: "a"
  detection: regex
  auto_fixable: false
`), 0o644)
	core.RequireNoError(t, err)

	cat, err := LoadDir(dir)
	core.RequireNoError(t, err)
	RequireLen(t, cat.Rules, 2)
	core.AssertEqual(t, "a-rule", cat.Rules[0].ID)
	core.AssertEqual(t, "z-rule", cat.Rules[1].ID)
}

func TestLoadDir_Bad_NonexistentDir(t *core.T) {
	_, err := LoadDir("/nonexistent/path/that/does/not/exist")
	core.AssertError(t, err)
}

func TestLoadDir_Bad_EmptyDir(t *core.T) {
	dir := t.TempDir()
	cat, err := LoadDir(dir)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, cat.Rules)
}

func TestLoadDir_Bad_InvalidYAML(t *core.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("{{{"), 0o644)
	core.RequireNoError(t, err)

	_, err = LoadDir(dir)
	core.AssertError(t, err)
}

func TestLoadFS_Good(t *core.T) {
	cat, err := LoadFS(testCatalogFS, "testdata/catalog")
	core.RequireNoError(t, err)
	core.AssertLen(t, cat.Rules, 2)
}

func TestForLanguage_Good(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-1", Languages: []string{"go"}},
			{ID: "php-1", Languages: []string{"php"}},
			{ID: "both-1", Languages: []string{"go", "php"}},
		},
	}

	goRules := cat.ForLanguage("go")
	core.AssertLen(t, goRules, 2)
	core.AssertEqual(t, "go-1", goRules[0].ID)
	core.AssertEqual(t, "both-1", goRules[1].ID)
}

func TestForLanguage_Bad_NoMatch(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-1", Languages: []string{"go"}},
		},
	}
	core.AssertEmpty(t, cat.ForLanguage("rust"))
}

func TestAtSeverity_Good(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "info-1", Severity: "info"},
			{ID: "low-1", Severity: "low"},
			{ID: "med-1", Severity: "medium"},
			{ID: "high-1", Severity: "high"},
			{ID: "crit-1", Severity: "critical"},
		},
	}

	high := cat.AtSeverity("high")
	core.AssertLen(t, high, 2)
	core.AssertEqual(t, "high-1", high[0].ID)
	core.AssertEqual(t, "crit-1", high[1].ID)

	all := cat.AtSeverity("info")
	core.AssertLen(t, all, 5)

	crit := cat.AtSeverity("critical")
	core.AssertLen(t, crit, 1)
}

func TestAtSeverity_Bad_UnknownSeverity(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "high-1", Severity: "high"},
		},
	}
	// Unknown severity returns empty.
	core.AssertEmpty(t, cat.AtSeverity("catastrophic"))
}

func TestByID_Good(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-sec-001", Title: "SQL injection"},
			{ID: "go-sec-002", Title: "Path traversal"},
		},
	}

	r := cat.ByID("go-sec-002")
	RequireNotNil(t, r)
	core.AssertEqual(t, "Path traversal", r.Title)
}

func TestByID_Bad_NotFound(t *core.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-sec-001"},
		},
	}
	core.AssertNil(t, cat.ByID("nonexistent"))
}

func TestLoadDir_Good_AllRulesValidate(t *core.T) {
	dir := findCatalogDir(t)
	cat, err := LoadDir(dir)
	core.RequireNoError(t, err)

	for _, r := range cat.Rules {
		err := r.Validate()
		core.AssertNoError(t, err, "rule %s failed validation", r.ID)
	}
}

// findCatalogDir locates the catalog/ directory relative to the repo root.
func findCatalogDir(t *core.T) string {
	t.Helper()
	// Walk up from the test file to find the repo root with catalog/.
	dir, err := os.Getwd()
	core.RequireNoError(t, err)
	for {
		candidate := filepath.Join(dir, "catalog")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find catalog/ directory")
		}
		dir = parent
	}
}
