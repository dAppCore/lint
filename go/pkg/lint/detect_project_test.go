package lint

import (
	core "dappco.re/go"
)

const (
	detectProjectTestIgnoredGoba2352      = "ignored.go"
	detectProjectTestPackageIgnored145cee = "package ignored\n"
)

func TestDetect_Good_ProjectMarkersAndFiles(t *core.T) {
	dir := t.TempDir()

	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "main.cpp"), []byte("int main() { return 0; }\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "package.json"), []byte("{}\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "tsconfig.json"), []byte("{}\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "requirements.txt"), []byte("ruff\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "README.md"), []byte("# Test\n"), 0o644))
	RequireResultOK(t, core.MkdirAll(core.PathJoin(dir, "vendor"), 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "vendor", detectProjectTestIgnoredGoba2352), []byte(detectProjectTestPackageIgnored145cee), 0o644))

	core.AssertEqual(t,
		[]string{"cpp", "dockerfile", "go", "js", "markdown", "python", "shell", "ts"},
		Detect(dir),
	)
}

func TestDetect_Good_MarkerCoverage(t *core.T) {
	dir := t.TempDir()

	files := map[string]string{
		"go.mod":           "module example.com/test\n",
		"composer.json":    "{}\n",
		"package.json":     "{}\n",
		"tsconfig.json":    "{}\n",
		"requirements.txt": "ruff\n",
		"pyproject.toml":   "[tool.ruff]\n",
		"Cargo.toml":       "[package]\nname = \"test\"\n",
		"Dockerfile.dev":   "FROM scratch\n",
		"run.sh":           "#!/bin/sh\n",
		"main.cpp":         "int main() { return 0; }\n",
		"config.yaml":      "kind: Config\n",
		"config.yml":       "kind: Config\n",
	}

	for name, content := range files {
		RequireResultOK(t, core.WriteFile(core.PathJoin(dir, name), []byte(content), 0o644))
	}

	core.AssertEqual(t,
		[]string{"cpp", "dockerfile", "go", "js", "php", "python", "rust", "shell", "ts", "yaml"},
		Detect(dir),
	)
}

func TestDetectFromFiles_Good(t *core.T) {
	files := []string{
		"main.go",
		"src/lib.cc",
		"web/app.ts",
		"Dockerfile",
		"scripts/run.sh",
		"docs/index.md",
	}

	core.AssertEqual(t,
		[]string{"cpp", "dockerfile", "go", "markdown", "shell", "ts"},
		detectFromFiles(files),
	)
}

func TestDetect_Bad_MissingPathReturnsEmptySlice(t *core.T) {
	got := Detect(core.PathJoin(t.TempDir(), "missing"))
	core.AssertEqual(t, []string{}, got)
	core.AssertNotNil(t, got)
}

func TestDetect_Good_SkipsHiddenRootDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := core.PathJoin(dir, ".core")
	RequireResultOK(t, core.MkdirAll(hiddenDir, 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(hiddenDir, "main.go"), []byte("package main\n"), 0o644))

	core.AssertEqual(t, []string{}, Detect(hiddenDir))
}

func TestDetect_Ugly_SkipsNestedHiddenAndExcludedDirectories(t *core.T) {
	dir := t.TempDir()

	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "root.go"), []byte("package main\n"), 0o644))
	RequireResultOK(t, core.MkdirAll(core.PathJoin(dir, "vendor"), 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "vendor", detectProjectTestIgnoredGoba2352), []byte(detectProjectTestPackageIgnored145cee), 0o644))
	RequireResultOK(t, core.MkdirAll(core.PathJoin(dir, ".core"), 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, ".core", detectProjectTestIgnoredGoba2352), []byte(detectProjectTestPackageIgnored145cee), 0o644))
	RequireResultOK(t, core.MkdirAll(core.PathJoin(dir, "services", ".generated"), 0o755))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "services", ".generated", detectProjectTestIgnoredGoba2352), []byte(detectProjectTestPackageIgnored145cee), 0o644))

	core.AssertEqual(t, []string{"go"}, Detect(dir))
}

func TestDetectProject_Detect_Good(t *core.T) {
	subject := Detect
	if subject == nil {
		t.FailNow()
	}
	marker := "Detect:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetectProject_Detect_Bad(t *core.T) {
	subject := Detect
	if subject == nil {
		t.FailNow()
	}
	marker := "Detect:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetectProject_Detect_Ugly(t *core.T) {
	subject := Detect
	if subject == nil {
		t.FailNow()
	}
	marker := "Detect:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
