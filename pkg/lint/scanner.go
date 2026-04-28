package lint

import (
	"io/fs"
	"path/filepath" // Note: AX-6 — WalkDir and Rel do not have core equivalents.
	"slices"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// extensionMap maps file extensions to language identifiers.
var extensionMap = map[string]string{
	".go":   "go",
	".php":  "php",
	".ts":   "ts",
	".tsx":  "ts",
	".js":   "js",
	".jsx":  "js",
	".cpp":  "cpp",
	".cc":   "cpp",
	".c":    "cpp",
	".h":    "cpp",
	".py":   "python",
	".rs":   "rust",
	".sh":   "shell",
	".yaml": "yaml",
	".yml":  "yaml",
	".json": "json",
	".md":   "markdown",
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
//
//	lint.DetectLanguage("main.go")
//	lint.DetectLanguage("Dockerfile")
func DetectLanguage(filename string) string {
	base := core.PathBase(filename)
	if core.HasPrefix(base, "Dockerfile") {
		return "dockerfile"
	}

	ext := core.PathExt(base)
	if lang, ok := extensionMap[ext]; ok {
		return lang
	}
	return ""
}

func shouldSkipTraversalRoot(path string) bool {
	cleanedPath := core.CleanPath(path, "/")
	if cleanedPath == "." {
		return false
	}

	base := core.PathBase(cleanedPath)
	if base == "." || base == "/" {
		return false
	}

	return IsExcludedDir(base)
}

// Scanner walks directory trees and matches files against lint rules.
type Scanner struct {
	matcher *Matcher
	rules   []Rule
}

// NewScanner creates a Scanner with the given rules and default directory exclusions.
func NewScanner(rules []Rule) (*Scanner, error) {
	matcher, err := NewMatcher(rules)
	if err != nil {
		return nil, err
	}
	return &Scanner{
		matcher: matcher,
		rules:   rules,
	}, nil
}

// ScanDir walks the directory tree rooted at root, scanning each recognised file.
// Directories in the exclude list are skipped entirely.
func (s *Scanner) ScanDir(root string) ([]Finding, error) {
	var findings []Finding

	if shouldSkipTraversalRoot(root) {
		return findings, nil
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories and hidden directories.
		if d.IsDir() {
			name := d.Name()
			if IsExcludedDir(name) {
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
		matcher, err := NewMatcher(langRules)
		if err != nil {
			return err
		}

		// Use a relative path from root for cleaner output.
		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relPath = path
		}

		found := matcher.Match(relPath, content)
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

	lang := DetectLanguage(core.PathBase(path))
	if lang == "" {
		return nil, nil
	}

	langRules := filterRulesByLanguage(s.rules, lang)
	if len(langRules) == 0 {
		return nil, nil
	}

	matcher, err := NewMatcher(langRules)
	if err != nil {
		return nil, err
	}

	return matcher.Match(path, content), nil
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
	return slices.Contains(defaultExcludes, name) || core.HasPrefix(name, ".")
}
