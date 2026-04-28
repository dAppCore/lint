package lint

import (
	core "dappco.re/go"
	"os"
	"path/filepath"
)

func TestDetectLanguage_Good(t *core.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"main.go", "go"},
		{"handler.go", "go"},
		{"model.php", "php"},
		{"app.ts", "ts"},
		{"component.tsx", "ts"},
		{"main.cpp", "cpp"},
		{"lib.cc", "cpp"},
		{"header.h", "cpp"},
		{"core.c", "cpp"},
		{"app.js", "js"},
		{"component.jsx", "js"},
		{"unknown.rs", "rust"},
		{"noextension", ""},
		{"file.py", "python"},
		{"Dockerfile", "dockerfile"},
		{"services/Dockerfile.prod", "dockerfile"},
		{"configs/settings.yaml", "yaml"},
		{"configs/settings.yml", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *core.T) {
			got := DetectLanguage(tt.filename)
			core.AssertEqual(t, tt.want, got)
		})
	}
}

func TestDetectLanguage_Bad_UnknownExtension(t *core.T) {
	core.AssertEqual(t, "", DetectLanguage("notes.txt"))
	core.AssertEqual(t, "", DetectLanguage("README"))
	core.AssertEqual(t, "", DetectLanguage(""))
}

func TestDetectLanguage_Ugly_DockerfileVariant(t *core.T) {
	got := DetectLanguage("nested/Dockerfile.test")
	core.AssertEqual(t, "dockerfile", got)
	core.AssertNotEqual(t, "", got)
}

func TestScanDir_Good_FindsMatches(t *core.T) {
	dir := t.TempDir()

	// Create a Go file with a TODO.
	goFile := filepath.Join(dir, "main.go")
	err := os.WriteFile(goFile, []byte("package main\n\n// TODO: fix this\nfunc main() {}\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(dir)
	core.RequireNoError(t, err)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "test-001", findings[0].RuleID)
	core.AssertEqual(t, 3, findings[0].Line)
}

func TestScanDir_Good_ExcludesVendor(t *core.T) {
	dir := t.TempDir()

	// Create vendor directory with a matching file.
	vendorDir := filepath.Join(dir, "vendor")
	core.RequireNoError(t, os.MkdirAll(vendorDir, 0o755))
	err := os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("// TODO: vendor code\n"), 0o644)
	core.RequireNoError(t, err)

	// Create node_modules directory with a matching file.
	nodeDir := filepath.Join(dir, "node_modules")
	core.RequireNoError(t, os.MkdirAll(nodeDir, 0o755))
	err = os.WriteFile(filepath.Join(nodeDir, "index.js"), []byte("// TODO: node code\n"), 0o644)
	core.RequireNoError(t, err)

	// Create .git directory with a matching file.
	gitDir := filepath.Join(dir, ".git")
	core.RequireNoError(t, os.MkdirAll(gitDir, 0o755))
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("// TODO: git\n"), 0o644)
	core.RequireNoError(t, err)

	// Create testdata directory with a matching file.
	testdataDir := filepath.Join(dir, "testdata")
	core.RequireNoError(t, os.MkdirAll(testdataDir, 0o755))
	err = os.WriteFile(filepath.Join(testdataDir, "sample.go"), []byte("// TODO: testdata\n"), 0o644)
	core.RequireNoError(t, err)

	// Create .core directory with a matching file.
	coreDir := filepath.Join(dir, ".core")
	core.RequireNoError(t, os.MkdirAll(coreDir, 0o755))
	err = os.WriteFile(filepath.Join(coreDir, "build.go"), []byte("// TODO: build\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "low",
			Languages: []string{"go", "js"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(dir)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, findings, "should not find matches in excluded directories")
}

func TestScanDir_Good_LanguageFiltering(t *core.T) {
	dir := t.TempDir()

	// Create Go file with a match.
	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("// TODO: go\n"), 0o644)
	core.RequireNoError(t, err)

	// Create PHP file with a match — rule only targets Go.
	err = os.WriteFile(filepath.Join(dir, "index.php"), []byte("// TODO: php\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "go-only",
			Title:     "Go TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(dir)
	core.RequireNoError(t, err)
	RequireLen(t, findings, 1)
	core.AssertContains(t, findings[0].File, "main.go")
}

func TestScanFile_Good(t *core.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")
	err := os.WriteFile(file, []byte("package main\n\npanic(\"boom\")\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "test-panic",
			Title:     "Panic found",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   `\bpanic\(`,
			Fix:       "Return error",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanFile(file)
	core.RequireNoError(t, err)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "test-panic", findings[0].RuleID)
}

func TestScanFile_Good_Python(t *core.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.py")
	err := os.WriteFile(file, []byte("print('hello')\n# TODO: fix\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "python-todo",
			Title:     "Python TODO",
			Severity:  "low",
			Languages: []string{"python"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanFile(file)
	core.RequireNoError(t, err)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "python-todo", findings[0].RuleID)
	core.AssertEqual(t, "python", DetectLanguage(file))
}

func TestScanFile_Bad_NoMatchingLanguageRules(t *core.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.go")
	err := os.WriteFile(file, []byte("package main\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "php-only",
			Title:     "PHP TODO",
			Severity:  "low",
			Languages: []string{"php"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanFile(file)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, findings)
}

func TestScanFile_Ugly_UnsupportedExtension(t *core.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	err := os.WriteFile(file, []byte("TODO: this is not a recognised source file\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "go-only",
			Title:     "Go TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanFile(file)
	core.RequireNoError(t, err)
	core.AssertNil(t, findings)
}

func TestScanDir_Good_Subdirectories(t *core.T) {
	dir := t.TempDir()

	// Create a nested file.
	subDir := filepath.Join(dir, "pkg", "store")
	core.RequireNoError(t, os.MkdirAll(subDir, 0o755))
	err := os.WriteFile(filepath.Join(subDir, "db.go"), []byte("// TODO: deep\n"), 0o644)
	core.RequireNoError(t, err)

	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(dir)
	core.RequireNoError(t, err)
	RequireLen(t, findings, 1)
}

func TestScanDir_Good_SkipsHiddenRootDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".git")
	core.RequireNoError(t, os.MkdirAll(hiddenDir, 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(hiddenDir, "main.go"), []byte("// TODO: hidden\n"), 0o644))

	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(hiddenDir)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, findings)
}

func TestScanDir_Good_SkipsHiddenNestedDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, "services", ".generated")
	core.RequireNoError(t, os.MkdirAll(hiddenDir, 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(hiddenDir, "main.go"), []byte("// TODO: hidden\n"), 0o644))

	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	findings, err := s.ScanDir(dir)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, findings)
}

func TestScanDir_Bad_NonexistentDir(t *core.T) {
	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Test",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Fix",
			Detection: "regex",
		},
	}

	s, err := NewScanner(rules)
	core.RequireNoError(t, err)

	_, err = s.ScanDir("/nonexistent/path")
	core.AssertError(t, err)
}
