package qa

import (
	. "dappco.re/go"
	"encoding/json"
	"path/filepath"
)

func TestRunDocblockCheckJSONOutput_IsDeterministicAndKeepsWarnings(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "b.go"), "package sample\n\nfunc Beta() {}\n")
	writeTestFile(t, filepath.Join(dir, "a.go"), "package sample\n\nfunc Alpha() {}\n")
	writeTestFile(t, filepath.Join(dir, "broken.go"), "package sample\n\nfunc Broken(\n")

	restoreWorkingDir(t, dir)

	var result DocblockResult
	output := captureStdout(t, func() {
		err := RunDocblockCheck([]string{"."}, 100, false, true)
		RequireError(t, err)
	})

	RequireNoError(t, json.Unmarshal([]byte(output), &result))
	AssertFalse(t, result.Passed)
	AssertEqual(t, 2, result.Total)
	AssertEqual(t, 0, result.Documented)
	RequireLen(t, result.Missing, 2)
	AssertEqual(t, "a.go", result.Missing[0].File)
	AssertEqual(t, "b.go", result.Missing[1].File)
	RequireLen(t, result.Warnings, 1)
	AssertEqual(t, ".", result.Warnings[0].Path)
	AssertNotEmpty(t, result.Warnings[0].Error)
}
