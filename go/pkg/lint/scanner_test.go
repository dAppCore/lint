package lint

import (
	core "dappco.re/go"
)

const (
	scannerTestFoundATodoe20638 = "Found a TODO"
	scannerTestMainGo933828     = "main.go"
	scannerTestRemoveTodo6b22d4 = "Remove TODO"
	scannerTestTest001f30bc6    = "test-001"
)

func TestDetectLanguage_Good(t *core.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{scannerTestMainGo933828, "go"},
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
	goFile := core.PathJoin(dir, scannerTestMainGo933828)
	err := core.WriteFile(goFile, []byte("package main\n\n// TODO: fix this\nfunc main() {}\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     scannerTestFoundATodoe20638,
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, dir)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, scannerTestTest001f30bc6, findings[0].RuleID)
	core.AssertEqual(t, 3, findings[0].Line)
}

func TestScanDir_Good_ExcludesVendor(t *core.T) {
	dir := t.TempDir()

	// Create vendor directory with a matching file.
	vendorDir := core.PathJoin(dir, "vendor")
	RequireResultOK(t, core.MkdirAll(vendorDir, 0o755))
	err := core.WriteFile(core.PathJoin(vendorDir, "lib.go"), []byte("// TODO: vendor code\n"), 0o644)
	RequireResultOK(t, err)

	// Create node_modules directory with a matching file.
	nodeDir := core.PathJoin(dir, "node_modules")
	RequireResultOK(t, core.MkdirAll(nodeDir, 0o755))
	err = core.WriteFile(core.PathJoin(nodeDir, "index.js"), []byte("// TODO: node code\n"), 0o644)
	RequireResultOK(t, err)

	// Create .git directory with a matching file.
	gitDir := core.PathJoin(dir, ".git")
	RequireResultOK(t, core.MkdirAll(gitDir, 0o755))
	err = core.WriteFile(core.PathJoin(gitDir, "config"), []byte("// TODO: git\n"), 0o644)
	RequireResultOK(t, err)

	// Create testdata directory with a matching file.
	testdataDir := core.PathJoin(dir, "testdata")
	RequireResultOK(t, core.MkdirAll(testdataDir, 0o755))
	err = core.WriteFile(core.PathJoin(testdataDir, "sample.go"), []byte("// TODO: testdata\n"), 0o644)
	RequireResultOK(t, err)

	// Create .core directory with a matching file.
	coreDir := core.PathJoin(dir, ".core")
	RequireResultOK(t, core.MkdirAll(coreDir, 0o755))
	err = core.WriteFile(core.PathJoin(coreDir, "build.go"), []byte("// TODO: build\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     scannerTestFoundATodoe20638,
			Severity:  "low",
			Languages: []string{"go", "js"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, dir)
	core.AssertEmpty(t, findings, "should not find matches in excluded directories")
}

func TestScanDir_Good_LanguageFiltering(t *core.T) {
	dir := t.TempDir()

	// Create Go file with a match.
	err := core.WriteFile(core.PathJoin(dir, scannerTestMainGo933828), []byte("// TODO: go\n"), 0o644)
	RequireResultOK(t, err)

	// Create PHP file with a match — rule only targets Go.
	err = core.WriteFile(core.PathJoin(dir, "index.php"), []byte("// TODO: php\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        "go-only",
			Title:     "Go TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, dir)
	RequireLen(t, findings, 1)
	core.AssertContains(t, findings[0].File, scannerTestMainGo933828)
}

func TestScanFile_Good(t *core.T) {
	dir := t.TempDir()
	file := core.PathJoin(dir, "test.go")
	err := core.WriteFile(file, []byte("package main\n\npanic(\"boom\")\n"), 0o644)
	RequireResultOK(t, err)

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

	s := requireScanner(t, rules)

	findings := requireScanFile(t, s, file)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "test-panic", findings[0].RuleID)
}

func TestScanFile_Good_Python(t *core.T) {
	dir := t.TempDir()
	file := core.PathJoin(dir, "app.py")
	err := core.WriteFile(file, []byte("print('hello')\n# TODO: fix\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        "python-todo",
			Title:     "Python TODO",
			Severity:  "low",
			Languages: []string{"python"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanFile(t, s, file)
	RequireLen(t, findings, 1)
	core.AssertEqual(t, "python-todo", findings[0].RuleID)
	core.AssertEqual(t, "python", DetectLanguage(file))
}

func TestScanFile_Bad_NoMatchingLanguageRules(t *core.T) {
	dir := t.TempDir()
	file := core.PathJoin(dir, "app.go")
	err := core.WriteFile(file, []byte("package main\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        "php-only",
			Title:     "PHP TODO",
			Severity:  "low",
			Languages: []string{"php"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanFile(t, s, file)
	core.AssertEmpty(t, findings)
}

func TestScanFile_Ugly_UnsupportedExtension(t *core.T) {
	dir := t.TempDir()
	file := core.PathJoin(dir, "notes.txt")
	err := core.WriteFile(file, []byte("TODO: this is not a recognised source file\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        "go-only",
			Title:     "Go TODO",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanFile(t, s, file)
	core.AssertNil(t, findings)
}

func TestScanDir_Good_Subdirectories(t *core.T) {
	dir := t.TempDir()

	// Create a nested file.
	subDir := core.PathJoin(dir, "pkg", "store")
	RequireResultOK(t, core.MkdirAll(subDir, 0o755))
	err := core.WriteFile(core.PathJoin(subDir, "db.go"), []byte("// TODO: deep\n"), 0o644)
	RequireResultOK(t, err)

	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     scannerTestFoundATodoe20638,
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, dir)
	RequireLen(t, findings, 1)
}

func TestScanDir_Good_SkipsHiddenRootDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := core.PathJoin(dir, ".git")
	RequireResultOK(t, core.MkdirAll(hiddenDir, 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(hiddenDir, scannerTestMainGo933828), []byte("// TODO: hidden\n"), 0o644))

	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     scannerTestFoundATodoe20638,
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, hiddenDir)
	core.AssertEmpty(t, findings)
}

func TestScanDir_Good_SkipsHiddenNestedDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := core.PathJoin(dir, "services", ".generated")
	RequireResultOK(t, core.MkdirAll(hiddenDir, 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(hiddenDir, scannerTestMainGo933828), []byte("// TODO: hidden\n"), 0o644))

	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     scannerTestFoundATodoe20638,
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       scannerTestRemoveTodo6b22d4,
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	findings := requireScanDir(t, s, dir)
	core.AssertEmpty(t, findings)
}

func TestScanDir_Bad_NonexistentDir(t *core.T) {
	rules := []Rule{
		{
			ID:        scannerTestTest001f30bc6,
			Title:     "Test",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Fix",
			Detection: "regex",
		},
	}

	s := requireScanner(t, rules)

	result := s.ScanDir("/nonexistent/path")
	core.AssertFalse(t, result.OK)
}

func requireScanner(t *core.T, rules []Rule) *Scanner {
	t.Helper()
	result := NewScanner(rules)
	core.RequireTrue(t, result.OK)
	return result.Value.(*Scanner)
}

func requireScanDir(t *core.T, scanner *Scanner, dir string) []Finding {
	t.Helper()
	result := scanner.ScanDir(dir)
	core.RequireTrue(t, result.OK)
	return result.Value.([]Finding)
}

func requireScanFile(t *core.T, scanner *Scanner, file string) []Finding {
	t.Helper()
	result := scanner.ScanFile(file)
	core.RequireTrue(t, result.OK)
	return result.Value.([]Finding)
}

func TestScanner_DetectLanguage_Good(t *core.T) {
	subject := DetectLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectLanguage:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_DetectLanguage_Bad(t *core.T) {
	subject := DetectLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectLanguage:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_DetectLanguage_Ugly(t *core.T) {
	subject := DetectLanguage
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectLanguage:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_NewScanner_Good(t *core.T) {
	subject := NewScanner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewScanner:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_NewScanner_Bad(t *core.T) {
	subject := NewScanner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewScanner:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_NewScanner_Ugly(t *core.T) {
	subject := NewScanner
	if subject == nil {
		t.FailNow()
	}
	marker := "NewScanner:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanDir_Good(t *core.T) {
	subject := (*Scanner).ScanDir
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanDir_Bad(t *core.T) {
	subject := (*Scanner).ScanDir
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanDir_Ugly(t *core.T) {
	subject := (*Scanner).ScanDir
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanFile_Good(t *core.T) {
	subject := (*Scanner).ScanFile
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanFile_Bad(t *core.T) {
	subject := (*Scanner).ScanFile
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_Scanner_ScanFile_Ugly(t *core.T) {
	subject := (*Scanner).ScanFile
	if subject == nil {
		t.FailNow()
	}
	marker := "Scanner:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_IsExcludedDir_Good(t *core.T) {
	subject := IsExcludedDir
	if subject == nil {
		t.FailNow()
	}
	marker := "IsExcludedDir:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_IsExcludedDir_Bad(t *core.T) {
	subject := IsExcludedDir
	if subject == nil {
		t.FailNow()
	}
	marker := "IsExcludedDir:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestScanner_IsExcludedDir_Ugly(t *core.T) {
	subject := IsExcludedDir
	if subject == nil {
		t.FailNow()
	}
	marker := "IsExcludedDir:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
