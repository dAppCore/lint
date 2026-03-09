package lint

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/catalog/*.yaml
var testCatalogFS embed.FS

func TestLoadDir_Good(t *testing.T) {
	// Use the real catalog/ directory at the repo root.
	dir := findCatalogDir(t)
	cat, err := LoadDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, cat.Rules)

	// The seed file has at least one rule.
	assert.Equal(t, "go-sec-001", cat.Rules[0].ID)
}

func TestLoadDir_Bad_NonexistentDir(t *testing.T) {
	_, err := LoadDir("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestLoadDir_Bad_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, cat.Rules)
}

func TestLoadDir_Bad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("{{{"), 0o644)
	require.NoError(t, err)

	_, err = LoadDir(dir)
	assert.Error(t, err)
}

func TestLoadFS_Good(t *testing.T) {
	cat, err := LoadFS(testCatalogFS, "testdata/catalog")
	require.NoError(t, err)
	assert.Len(t, cat.Rules, 2)
}

func TestForLanguage_Good(t *testing.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-1", Languages: []string{"go"}},
			{ID: "php-1", Languages: []string{"php"}},
			{ID: "both-1", Languages: []string{"go", "php"}},
		},
	}

	goRules := cat.ForLanguage("go")
	assert.Len(t, goRules, 2)
	assert.Equal(t, "go-1", goRules[0].ID)
	assert.Equal(t, "both-1", goRules[1].ID)
}

func TestForLanguage_Bad_NoMatch(t *testing.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-1", Languages: []string{"go"}},
		},
	}
	assert.Empty(t, cat.ForLanguage("rust"))
}

func TestAtSeverity_Good(t *testing.T) {
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
	assert.Len(t, high, 2)
	assert.Equal(t, "high-1", high[0].ID)
	assert.Equal(t, "crit-1", high[1].ID)

	all := cat.AtSeverity("info")
	assert.Len(t, all, 5)

	crit := cat.AtSeverity("critical")
	assert.Len(t, crit, 1)
}

func TestAtSeverity_Bad_UnknownSeverity(t *testing.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "high-1", Severity: "high"},
		},
	}
	// Unknown severity returns empty.
	assert.Empty(t, cat.AtSeverity("catastrophic"))
}

func TestByID_Good(t *testing.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-sec-001", Title: "SQL injection"},
			{ID: "go-sec-002", Title: "Path traversal"},
		},
	}

	r := cat.ByID("go-sec-002")
	require.NotNil(t, r)
	assert.Equal(t, "Path traversal", r.Title)
}

func TestByID_Bad_NotFound(t *testing.T) {
	cat := &Catalog{
		Rules: []Rule{
			{ID: "go-sec-001"},
		},
	}
	assert.Nil(t, cat.ByID("nonexistent"))
}

// findCatalogDir locates the catalog/ directory relative to the repo root.
func findCatalogDir(t *testing.T) string {
	t.Helper()
	// Walk up from the test file to find the repo root with catalog/.
	dir, err := os.Getwd()
	require.NoError(t, err)
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
