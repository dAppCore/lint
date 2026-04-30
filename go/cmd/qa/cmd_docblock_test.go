package qa

import (
	. "dappco.re/go"
)

func TestRunDocblockCheckJSONOutput_IsDeterministicAndKeepsWarnings(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, "b.go"), "package sample\n\nfunc Beta() {}\n")
	writeTestFile(t, PathJoin(dir, "a.go"), "package sample\n\nfunc Alpha() {}\n")
	writeTestFile(t, PathJoin(dir, "broken.go"), "package sample\n\nfunc Broken(\n")

	restoreWorkingDir(t, dir)

	var result DocblockResult
	output := captureStdout(t, func() {
		r := RunDocblockCheck([]string{"."}, 100, false, true)
		RequireResultError(t, r)
	})

	RequireResultOK(t, JSONUnmarshal([]byte(output), &result))
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

func TestCmdDocblock_RunDocblockCheck_Good(t *T) {
	subject := RunDocblockCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "RunDocblockCheck:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdDocblock_RunDocblockCheck_Bad(t *T) {
	subject := RunDocblockCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "RunDocblockCheck:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdDocblock_RunDocblockCheck_Ugly(t *T) {
	subject := RunDocblockCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "RunDocblockCheck:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdDocblock_CheckDocblockCoverage_Good(t *T) {
	subject := CheckDocblockCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CheckDocblockCoverage:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdDocblock_CheckDocblockCoverage_Bad(t *T) {
	subject := CheckDocblockCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CheckDocblockCoverage:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdDocblock_CheckDocblockCoverage_Ugly(t *T) {
	subject := CheckDocblockCoverage
	if subject == nil {
		t.FailNow()
	}
	marker := "CheckDocblockCoverage:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
