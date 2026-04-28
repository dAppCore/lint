package qa

import (
	"os"
	"path/filepath"

	. "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

func ax7DocblockFile(t *T, content string) string {
	t.Helper()
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "sample.go"), []byte(content), 0o644))
	return dir
}

func TestQA_AddQACommands_Good(t *T) {
	root := &cli.Command{Use: "root"}
	AddQACommands(root)
	AssertLen(t, root.Commands(), 1)
	AssertEqual(t, "qa", root.Commands()[0].Use)
}

func TestQA_AddQACommands_Bad(t *T) {
	root := &cli.Command{Use: "root"}
	root.AddCommand(&cli.Command{Use: "existing"})
	AddQACommands(root)
	AssertLen(t, root.Commands(), 2)
	AssertEqual(t, "existing", root.Commands()[0].Use)
}

func TestQA_AddQACommands_Ugly(t *T) {
	root := &cli.Command{Use: "root"}
	AddQACommands(root)
	AddQACommands(root)
	AssertLen(t, root.Commands(), 2)
	AssertEqual(t, "qa", root.Commands()[1].Use)
}

func TestQA_CheckDocblockCoverage_Good(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\n// Alpha is documented.\nfunc Alpha() {}\n")
	result, err := CheckDocblockCoverage([]string{dir})
	AssertNoError(t, err)
	AssertEqual(t, 100.0, result.Coverage)
	AssertTrue(t, result.Documented == result.Total)
}

func TestQA_CheckDocblockCoverage_Bad(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\nfunc Alpha() {}\n")
	result, err := CheckDocblockCoverage([]string{dir})
	AssertNoError(t, err)
	AssertEqual(t, 0.0, result.Coverage)
	AssertLen(t, result.Missing, 1)
}

func TestQA_CheckDocblockCoverage_Ugly(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\n// Alpha is documented.\nfunc Alpha() {}\n")
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package sample\nfunc"), 0o644))
	result, err := CheckDocblockCoverage([]string{dir})
	AssertNoError(t, err)
	AssertLen(t, result.Warnings, 1)
	AssertEqual(t, 100.0, result.Coverage)
}

func TestQA_RunDocblockCheck_Good(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\n// Alpha is documented.\nfunc Alpha() {}\n")
	err := RunDocblockCheck([]string{dir}, 100, false, false)
	AssertNoError(t, err)
	AssertTrue(t, true)
}

func TestQA_RunDocblockCheck_Bad(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\nfunc Alpha() {}\n")
	err := RunDocblockCheck([]string{dir}, 100, false, false)
	AssertError(t, err)
	AssertContains(t, err.Error(), "below threshold")
}

func TestQA_RunDocblockCheck_Ugly(t *T) {
	dir := ax7DocblockFile(t, "package sample\n\nfunc Alpha() {}\n")
	err := RunDocblockCheck([]string{dir}, 50, false, true)
	AssertError(t, err)
	AssertContains(t, err.Error(), "below threshold")
}
