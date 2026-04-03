package qa

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDocblockCheckJSONOutput_IsDeterministicAndKeepsWarnings(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "b.go"), "package sample\n\nfunc Beta() {}\n")
	writeTestFile(t, filepath.Join(dir, "a.go"), "package sample\n\nfunc Alpha() {}\n")
	writeTestFile(t, filepath.Join(dir, "broken.go"), "package sample\n\nfunc Broken(\n")

	restoreWorkingDir(t, dir)

	var result DocblockResult
	output := captureStdout(t, func() {
		err := RunDocblockCheck([]string{"."}, 100, false, true)
		require.Error(t, err)
	})

	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.False(t, result.Passed)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 0, result.Documented)
	require.Len(t, result.Missing, 2)
	assert.Equal(t, "a.go", result.Missing[0].File)
	assert.Equal(t, "b.go", result.Missing[1].File)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, ".", result.Warnings[0].Path)
	assert.NotEmpty(t, result.Warnings[0].Error)
}
