package lint

import (
	core "dappco.re/go"
	"os"
	"path/filepath"
)

func TestDetect_Good_ProjectMarkersAndFiles(t *core.T) {
	dir := t.TempDir()

	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "main.cpp"), []byte("int main() { return 0; }\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("ruff\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o644))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "vendor", "ignored.go"), []byte("package ignored\n"), 0o644))

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
		core.RequireNoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
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
	got := Detect(filepath.Join(t.TempDir(), "missing"))
	core.AssertEqual(t, []string{}, got)
	core.AssertNotNil(t, got)
}

func TestDetect_Good_SkipsHiddenRootDirectory(t *core.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".core")
	core.RequireNoError(t, os.MkdirAll(hiddenDir, 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(hiddenDir, "main.go"), []byte("package main\n"), 0o644))

	core.AssertEqual(t, []string{}, Detect(hiddenDir))
}

func TestDetect_Ugly_SkipsNestedHiddenAndExcludedDirectories(t *core.T) {
	dir := t.TempDir()

	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "root.go"), []byte("package main\n"), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "vendor", "ignored.go"), []byte("package ignored\n"), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", "ignored.go"), []byte("package ignored\n"), 0o644))
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, "services", ".generated"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, "services", ".generated", "ignored.go"), []byte("package ignored\n"), 0o644))

	core.AssertEqual(t, []string{"go"}, Detect(dir))
}
