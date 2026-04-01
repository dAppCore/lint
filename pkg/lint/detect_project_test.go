package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_Good_ProjectMarkersAndFiles(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("ruff\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vendor", "ignored.go"), []byte("package ignored\n"), 0o644))

	assert.Equal(t,
		[]string{"dockerfile", "go", "js", "markdown", "python", "shell", "ts"},
		Detect(dir),
	)
}

func TestDetectFromFiles_Good(t *testing.T) {
	files := []string{
		"main.go",
		"web/app.ts",
		"Dockerfile",
		"scripts/run.sh",
		"docs/index.md",
	}

	assert.Equal(t,
		[]string{"dockerfile", "go", "markdown", "shell", "ts"},
		detectFromFiles(files),
	)
}

func TestDetect_MissingPathReturnsEmptySlice(t *testing.T) {
	assert.Equal(t, []string{}, Detect(filepath.Join(t.TempDir(), "missing")))
}
