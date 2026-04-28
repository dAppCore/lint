package lint

import (
	core "dappco.re/go"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// setupMockCmd creates a shell script in a temp dir that echoes predetermined
// content, and prepends that dir to PATH so Run() picks it up.
func setupMockCmd(t *core.T, name, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	script := fmt.Sprintf("#!/bin/sh\ncat <<'MOCK_EOF'\n%s\nMOCK_EOF\n", content)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock command %s: %v", name, err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
}

// setupMockCmdExit creates a mock that echoes to stdout/stderr and exits with a code.
func setupMockCmdExit(t *core.T, name, stdout, stderr string, exitCode int) {
	t.Helper()
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	script := fmt.Sprintf("#!/bin/sh\ncat <<'MOCK_EOF'\n%s\nMOCK_EOF\ncat <<'MOCK_ERR' >&2\n%s\nMOCK_ERR\nexit %d\n", stdout, stderr, exitCode)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock command %s: %v", name, err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
}

func TestNewToolkit(t *core.T) {
	tk := NewToolkit("/tmp")
	core.AssertNotNil(t, tk)
	core.AssertEqual(t, "/tmp", tk.Dir)
}

func TestToolkit_Coverage_Good(t *core.T) {
	output := `?   	example.com/skipped	[no test files]
ok  	example.com/pkg1	0.5s	coverage: 85.0% of statements
ok  	example.com/pkg2	0.2s	coverage: 100.0% of statements`

	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	reports, err := tk.Coverage("./...")
	core.RequireNoError(t, err)
	RequireLen(t, reports, 2)
	core.AssertEqual(t, "example.com/pkg1", reports[0].Package)
	core.AssertEqual(t, 85.0, reports[0].Percentage)
	core.AssertEqual(t, "example.com/pkg2", reports[1].Package)
	core.AssertEqual(t, 100.0, reports[1].Percentage)
}

func TestToolkit_Coverage_Bad(t *core.T) {
	setupMockCmd(t, "go", "FAIL\texample.com/broken [build failed]")

	tk := NewToolkit(t.TempDir())
	reports, err := tk.Coverage("./...")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, reports)
}

func TestToolkit_GitLog_Good(t *core.T) {
	now := time.Now().Truncate(time.Second)
	nowStr := now.Format(time.RFC3339)

	output := fmt.Sprintf("abc123|Alice|%s|Fix the bug\ndef456|Bob|%s|Add feature", nowStr, nowStr)
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	commits, err := tk.GitLog(2)
	core.RequireNoError(t, err)
	RequireLen(t, commits, 2)
	core.AssertEqual(t, "abc123", commits[0].Hash)
	core.AssertEqual(t, "Alice", commits[0].Author)
	core.AssertEqual(t, "Fix the bug", commits[0].Message)
	core.AssertTrue(t, commits[0].Date.Equal(now))
}

func TestToolkit_GitLog_Bad(t *core.T) {
	setupMockCmd(t, "git", "incomplete|line\nabc|Bob|2025-01-01T00:00:00Z|Good commit")

	tk := NewToolkit(t.TempDir())
	commits, err := tk.GitLog(5)
	core.RequireNoError(t, err)
	core.AssertLen(t, commits, 1)
}

func TestToolkit_GocycloComplexity_Good(t *core.T) {
	output := "15 main ComplexFunc file.go:10:1\n20 pkg VeryComplex other.go:50:1"
	setupMockCmd(t, "gocyclo", output)

	tk := NewToolkit(t.TempDir())
	funcs, err := tk.GocycloComplexity(10)
	core.RequireNoError(t, err)
	RequireLen(t, funcs, 2)
	core.AssertEqual(t, 15, funcs[0].Score)
	core.AssertEqual(t, "ComplexFunc", funcs[0].FuncName)
	core.AssertEqual(t, "file.go", funcs[0].File)
	core.AssertEqual(t, 10, funcs[0].Line)
	core.AssertEqual(t, 20, funcs[1].Score)
	core.AssertEqual(t, "pkg", funcs[1].Package)
}

func TestToolkit_GocycloComplexity_Bad(t *core.T) {
	setupMockCmd(t, "gocyclo", "")

	tk := NewToolkit(t.TempDir())
	funcs, err := tk.GocycloComplexity(50)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, funcs)
}

func TestToolkit_DepGraph_Good(t *core.T) {
	output := "modA@v1 modB@v2\nmodA@v1 modC@v3\nmodB@v2 modD@v1"
	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	graph, err := tk.DepGraph("./...")
	core.RequireNoError(t, err)
	core.AssertLen(t, graph.Nodes, 4)
	core.AssertLen(t, graph.Edges["modA@v1"], 2)
}

func TestToolkit_DepGraph_SortsNodesAndEdges(t *core.T) {
	output := "modB@v2 modD@v1\nmodA@v1 modC@v3\nmodA@v1 modB@v2"
	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	graph, err := tk.DepGraph("./...")
	core.RequireNoError(t, err)

	core.AssertEqual(t, []string{"modA@v1", "modB@v2", "modC@v3", "modD@v1"}, graph.Nodes)
	core.AssertEqual(t, []string{"modB@v2", "modC@v3"}, graph.Edges["modA@v1"])
}

func TestToolkit_RaceDetect_Good(t *core.T) {
	setupMockCmd(t, "go", "ok\texample.com/safe\t0.1s")

	tk := NewToolkit(t.TempDir())
	races, err := tk.RaceDetect("./...")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, races)
}

func TestToolkit_RaceDetect_Bad(t *core.T) {
	stderrOut := `WARNING: DATA RACE
Read at 0x00c000123456 by goroutine 7:
      /home/user/project/main.go:42
Previous write at 0x00c000123456 by goroutine 6:
      /home/user/project/main.go:38`

	setupMockCmdExit(t, "go", "", stderrOut, 1)

	tk := NewToolkit(t.TempDir())
	races, err := tk.RaceDetect("./...")
	core.RequireNoError(t, err)
	RequireLen(t, races, 1)
	core.AssertEqual(t, "/home/user/project/main.go", races[0].File)
	core.AssertEqual(t, 42, races[0].Line)
}

func TestToolkit_DiffStat_Good(t *core.T) {
	output := ` file1.go | 10 +++++++---
 file2.go |  5 +++++
 2 files changed, 12 insertions(+), 3 deletions(-)`
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	s, err := tk.DiffStat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, s.FilesChanged)
	core.AssertEqual(t, 12, s.Insertions)
	core.AssertEqual(t, 3, s.Deletions)
}

func TestToolkit_CheckPerms_Good(t *core.T) {
	dir := t.TempDir()

	badFile := filepath.Join(dir, "bad.txt")
	core.RequireNoError(t, os.WriteFile(badFile, []byte("test"), 0644))
	core.RequireNoError(t, os.Chmod(badFile, 0666))

	goodFile := filepath.Join(dir, "good.txt")
	core.RequireNoError(t, os.WriteFile(goodFile, []byte("test"), 0644))

	tk := NewToolkit("/")
	issues, err := tk.CheckPerms(dir)
	core.RequireNoError(t, err)
	RequireLen(t, issues, 1)
	core.AssertEqual(t, "Group and world-writable", issues[0].Issue)
}

func TestToolkit_FindTrackedComments_Compatibility(t *core.T) {
	output := "pkg/file.go:12:TODO: fix this\n"
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	comments, err := tk.FindTrackedComments("pkg")
	core.RequireNoError(t, err)
	RequireLen(t, comments, 1)
	core.AssertEqual(t, "pkg/file.go", comments[0].File)
	core.AssertEqual(t, 12, comments[0].Line)
	core.AssertEqual(t, "TODO", comments[0].Type)
	core.AssertEqual(t, "fix this", comments[0].Message)

	legacyComments, err := tk.FindTODOs("pkg")
	core.RequireNoError(t, err)
	core.AssertEqual(t, comments, legacyComments)
}
