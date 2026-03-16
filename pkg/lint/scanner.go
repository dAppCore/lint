package lint

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

// extensionMap maps file extensions to language identifiers.
var extensionMap = map[string]string{
	".go":  "go",
	".php": "php",
	".ts":  "ts",
	".tsx": "ts",
	".js":  "js",
	".jsx": "js",
	".cpp": "cpp",
	".cc":  "cpp",
	".c":   "cpp",
	".h":   "cpp",
	".py":  "py",
}

// defaultExcludes lists directory names that are always skipped during scanning.
var defaultExcludes = []string{
	"vendor",
	"node_modules",
	".git",
	"testdata",
	".core",
}

// DetectLanguage returns the language identifier for a filename based on its extension.
// Returns an empty string for unrecognised extensions.
func DetectLanguage(filename string) string {
	ext := filepath.Ext(filename)
	if lang, ok := extensionMap[ext]; ok {
		return lang
	}
	return ""
}

// Scanner walks directory trees and matches files against lint rules.
type Scanner struct {
	matcher  *Matcher
	rules    []Rule
	excludes []string
}

// NewScanner creates a Scanner with the given rules and default directory exclusions.
func NewScanner(rules []Rule) (*Scanner, error) {
	m, err := NewMatcher(rules)
	if err != nil {
		return nil, err
	}
	return &Scanner{
		matcher:  m,
		rules:    rules,
		excludes: slices.Clone(defaultExcludes),
	}, nil
}

// ScanDir walks the directory tree rooted at root, scanning each recognised file.
// Directories in the exclude list are skipped entirely.
func (s *Scanner) ScanDir(root string) ([]Finding, error) {
	var findings []Finding

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories.
		if d.IsDir() {
			name := d.Name()
			if slices.Contains(s.excludes, name) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan files with recognised language extensions.
		lang := DetectLanguage(d.Name())
		if lang == "" {
			return nil
		}

		// Only match rules that target this file's language.
		langRules := filterRulesByLanguage(s.rules, lang)
		if len(langRules) == 0 {
			return nil
		}

		raw, err := coreio.Local.Read(path)
		if err != nil {
			return coreerr.E("Scanner.ScanDir", "reading "+path, err)
		}
		content := []byte(raw)

		// Build a matcher scoped to this file's language.
		m, err := NewMatcher(langRules)
		if err != nil {
			return err
		}

		// Use a relative path from root for cleaner output.
		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relPath = path
		}

		found := m.Match(relPath, content)
		findings = append(findings, found...)
		return nil
	})

	if err != nil {
		return nil, coreerr.E("Scanner.ScanDir", "scanning "+root, err)
	}

	return findings, nil
}

// ScanFile scans a single file against all rules.
func (s *Scanner) ScanFile(path string) ([]Finding, error) {
	raw, err := coreio.Local.Read(path)
	if err != nil {
		return nil, coreerr.E("Scanner.ScanFile", "reading "+path, err)
	}
	content := []byte(raw)

	lang := DetectLanguage(filepath.Base(path))
	if lang == "" {
		return nil, nil
	}

	langRules := filterRulesByLanguage(s.rules, lang)
	if len(langRules) == 0 {
		return nil, nil
	}

	m, err := NewMatcher(langRules)
	if err != nil {
		return nil, err
	}

	return m.Match(path, content), nil
}

// filterRulesByLanguage returns rules that include the given language.
func filterRulesByLanguage(rules []Rule, lang string) []Rule {
	var result []Rule
	for _, r := range rules {
		if slices.Contains(r.Languages, lang) {
			result = append(result, r)
		}
	}
	return result
}

// languagesFromRules collects all unique languages from a set of rules.
func languagesFromRules(rules []Rule) []string {
	seen := make(map[string]bool)
	for _, r := range rules {
		for _, l := range r.Languages {
			seen[l] = true
		}
	}
	var langs []string
	for l := range seen {
		langs = append(langs, l)
	}
	// Sort for deterministic output.
	slices.Sort(langs)
	return langs
}

// IsExcludedDir checks whether a directory name should be skipped.
func IsExcludedDir(name string) bool {
	return slices.Contains(defaultExcludes, name) || strings.HasPrefix(name, ".")
}
