package lint

import (
	"io/fs"
	"slices"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
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
func NewScanner(rules []Rule) core.Result {
	matcherResult := NewMatcher(rules)
	if !matcherResult.OK {
		return matcherResult
	}
	return core.Ok(&Scanner{
		matcher: matcherResult.Value.(*Matcher),
		rules:   rules,
	})
}

// ScanDir walks the directory tree rooted at root, scanning each recognised file.
// Directories in the exclude list are skipped entirely.
func (s *Scanner) ScanDir(root string) core.Result {
	var findings []Finding

	if shouldSkipTraversalRoot(root) {
		return core.Ok(findings)
	}

	err := core.PathWalkDir(root, func(path string, d fs.DirEntry, err error) error {
		entryResult := s.scanDirEntry(root, path, d, err)
		if !entryResult.OK {
			if entryResult.Value == fs.SkipDir {
				return fs.SkipDir
			}
			entryErr, _ := entryResult.Value.(error)
			return entryErr
		}
		findings = append(findings, entryResult.Value.([]Finding)...)
		return nil
	})

	if err != nil {
		return core.Fail(core.E("Scanner.ScanDir", "scanning "+root, err))
	}

	return core.Ok(findings)
}

func (s *Scanner) scanDirEntry(root string, path string, d fs.DirEntry, err error) core.Result {
	if err != nil {
		return core.Fail(err)
	}
	if d.IsDir() {
		if IsExcludedDir(d.Name()) {
			return core.Fail(fs.SkipDir)
		}
		return core.Ok([]Finding{})
	}

	langRules := rulesForFile(s.rules, d.Name())
	if len(langRules) == 0 {
		return core.Ok([]Finding{})
	}
	raw, err := coreio.Local.Read(path)
	if err != nil {
		return core.Fail(core.E("Scanner.ScanDir", "reading "+path, err))
	}
	matcherResult := NewMatcher(langRules)
	if !matcherResult.OK {
		return matcherResult
	}
	matcher := matcherResult.Value.(*Matcher)
	return core.Ok(matcher.Match(relativeScanPath(root, path), []byte(raw)))
}

func rulesForFile(rules []Rule, name string) []Rule {
	lang := DetectLanguage(name)
	if lang == "" {
		return nil
	}
	return filterRulesByLanguage(rules, lang)
}

func relativeScanPath(root string, path string) string {
	relResult := core.PathRel(root, path)
	if relResult.OK {
		return relResult.Value.(string)
	}
	return path
}

// ScanFile scans a single file against all rules.
func (s *Scanner) ScanFile(path string) core.Result {
	raw, err := coreio.Local.Read(path)
	if err != nil {
		return core.Fail(core.E("Scanner.ScanFile", "reading "+path, err))
	}
	content := []byte(raw)

	lang := DetectLanguage(core.PathBase(path))
	if lang == "" {
		var findings []Finding
		return core.Ok(findings)
	}

	langRules := filterRulesByLanguage(s.rules, lang)
	if len(langRules) == 0 {
		var findings []Finding
		return core.Ok(findings)
	}

	matcherResult := NewMatcher(langRules)
	if !matcherResult.OK {
		return matcherResult
	}
	matcher := matcherResult.Value.(*Matcher)

	return core.Ok(matcher.Match(path, content))
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
