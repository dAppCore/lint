package lint

import (
	"os"
	"slices"

	core "dappco.re/go"
)

var projectLanguageByExtension = map[string]string{
	".go":   "go",
	".php":  "php",
	".cpp":  "cpp",
	".cc":   "cpp",
	".c":    "cpp",
	".h":    "cpp",
	".js":   "js",
	".jsx":  "js",
	".ts":   "ts",
	".tsx":  "ts",
	".py":   "python",
	".rs":   "rust",
	".sh":   "shell",
	".yaml": "yaml",
	".yml":  "yaml",
	".json": "json",
	".md":   "markdown",
}

// Detect returns the project languages inferred from markers and file names.
//
//	lint.Detect(".")
//	lint.Detect("/path/to/project")
func Detect(path string) []string {
	if path == "" {
		path = "."
	}

	seen := make(map[string]bool)
	info, err := os.Stat(path)
	if err != nil {
		return []string{}
	}

	if !info.IsDir() {
		recordDetectedPath(seen, path)
		return sortedDetectedLanguages(seen)
	}

	if shouldSkipTraversalRoot(path) {
		return []string{}
	}

	walkDetectedProject(path, path, seen)

	return sortedDetectedLanguages(seen)
}

func walkDetectedProject(root string, current string, seen map[string]bool) {
	entries, err := os.ReadDir(current)
	if err != nil {
		return
	}

	for _, entry := range entries {
		currentPath := core.JoinPath(current, entry.Name())
		if entry.IsDir() {
			if currentPath != root && IsExcludedDir(entry.Name()) {
				continue
			}
			walkDetectedProject(root, currentPath, seen)
			continue
		}

		recordDetectedPath(seen, currentPath)
	}
}

func detectFromFiles(files []string) []string {
	seen := make(map[string]bool)
	for _, file := range files {
		recordDetectedPath(seen, file)
	}
	return sortedDetectedLanguages(seen)
}

func recordDetectedPath(seen map[string]bool, path string) {
	name := core.PathBase(path)
	matchedMarker := false

	switch {
	case name == "go.mod":
		seen["go"] = true
		matchedMarker = true
	case name == "composer.json":
		seen["php"] = true
		matchedMarker = true
	case name == "package.json":
		seen["js"] = true
		matchedMarker = true
	case name == "tsconfig.json":
		seen["ts"] = true
		matchedMarker = true
	case name == "requirements.txt", name == "pyproject.toml":
		seen["python"] = true
		matchedMarker = true
	case name == "Cargo.toml":
		seen["rust"] = true
		matchedMarker = true
	case core.HasPrefix(name, "Dockerfile"):
		seen["dockerfile"] = true
		matchedMarker = true
	}

	if matchedMarker {
		return
	}

	if lang, ok := projectLanguageByExtension[core.Lower(core.PathExt(name))]; ok {
		seen[lang] = true
	}
}

func sortedDetectedLanguages(seen map[string]bool) []string {
	var languages []string
	for language := range seen {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	if languages == nil {
		return []string{}
	}
	return languages
}
