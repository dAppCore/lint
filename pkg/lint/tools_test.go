package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockCmd creates a shell script in a temp dir that echoes predetermined
// content, and prepends that dir to PATH so Run() picks it up.
func setupMockCmd(t *testing.T, name, content string) {
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
func setupMockCmdExit(t *testing.T, name, stdout, stderr string, exitCode int) {
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

func TestNewToolkit(t *testing.T) {
	tk := NewToolkit("/tmp")
	assert.Equal(t, "/tmp", tk.Dir)
}

func TestToolkit_Coverage_Good(t *testing.T) {
	output := `?   	example.com/skipped	[no test files]
ok  	example.com/pkg1	0.5s	coverage: 85.0% of statements
ok  	example.com/pkg2	0.2s	coverage: 100.0% of statements`

	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	reports, err := tk.Coverage("./...")
	require.NoError(t, err)
	require.Len(t, reports, 2)
	assert.Equal(t, "example.com/pkg1", reports[0].Package)
	assert.Equal(t, 85.0, reports[0].Percentage)
	assert.Equal(t, "example.com/pkg2", reports[1].Package)
	assert.Equal(t, 100.0, reports[1].Percentage)
}

func TestToolkit_Coverage_Bad(t *testing.T) {
	setupMockCmd(t, "go", "FAIL\texample.com/broken [build failed]")

	tk := NewToolkit(t.TempDir())
	reports, err := tk.Coverage("./...")
	require.NoError(t, err)
	assert.Empty(t, reports)
}

func TestToolkit_GitLog_Good(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowStr := now.Format(time.RFC3339)

	output := fmt.Sprintf("abc123|Alice|%s|Fix the bug\ndef456|Bob|%s|Add feature", nowStr, nowStr)
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	commits, err := tk.GitLog(2)
	require.NoError(t, err)
	require.Len(t, commits, 2)
	assert.Equal(t, "abc123", commits[0].Hash)
	assert.Equal(t, "Alice", commits[0].Author)
	assert.Equal(t, "Fix the bug", commits[0].Message)
	assert.True(t, commits[0].Date.Equal(now))
}

func TestToolkit_GitLog_Bad(t *testing.T) {
	setupMockCmd(t, "git", "incomplete|line\nabc|Bob|2025-01-01T00:00:00Z|Good commit")

	tk := NewToolkit(t.TempDir())
	commits, err := tk.GitLog(5)
	require.NoError(t, err)
	assert.Len(t, commits, 1)
}

func TestToolkit_GocycloComplexity_Good(t *testing.T) {
	output := "15 main ComplexFunc file.go:10:1\n20 pkg VeryComplex other.go:50:1"
	setupMockCmd(t, "gocyclo", output)

	tk := NewToolkit(t.TempDir())
	funcs, err := tk.GocycloComplexity(10)
	require.NoError(t, err)
	require.Len(t, funcs, 2)
	assert.Equal(t, 15, funcs[0].Score)
	assert.Equal(t, "ComplexFunc", funcs[0].FuncName)
	assert.Equal(t, "file.go", funcs[0].File)
	assert.Equal(t, 10, funcs[0].Line)
	assert.Equal(t, 20, funcs[1].Score)
	assert.Equal(t, "pkg", funcs[1].Package)
}

func TestToolkit_GocycloComplexity_Bad(t *testing.T) {
	setupMockCmd(t, "gocyclo", "")

	tk := NewToolkit(t.TempDir())
	funcs, err := tk.GocycloComplexity(50)
	require.NoError(t, err)
	assert.Empty(t, funcs)
}

func TestToolkit_DepGraph_Good(t *testing.T) {
	output := "modA@v1 modB@v2\nmodA@v1 modC@v3\nmodB@v2 modD@v1"
	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	graph, err := tk.DepGraph("./...")
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 4)
	assert.Len(t, graph.Edges["modA@v1"], 2)
}

func TestToolkit_DepGraph_SortsNodesAndEdges(t *testing.T) {
	output := "modB@v2 modD@v1\nmodA@v1 modC@v3\nmodA@v1 modB@v2"
	setupMockCmd(t, "go", output)

	tk := NewToolkit(t.TempDir())
	graph, err := tk.DepGraph("./...")
	require.NoError(t, err)

	assert.Equal(t, []string{"modA@v1", "modB@v2", "modC@v3", "modD@v1"}, graph.Nodes)
	assert.Equal(t, []string{"modB@v2", "modC@v3"}, graph.Edges["modA@v1"])
}

func TestToolkit_RaceDetect_Good(t *testing.T) {
	setupMockCmd(t, "go", "ok\texample.com/safe\t0.1s")

	tk := NewToolkit(t.TempDir())
	races, err := tk.RaceDetect("./...")
	require.NoError(t, err)
	assert.Empty(t, races)
}

func TestToolkit_RaceDetect_Bad(t *testing.T) {
	stderrOut := `WARNING: DATA RACE
Read at 0x00c000123456 by goroutine 7:
      /home/user/project/main.go:42
Previous write at 0x00c000123456 by goroutine 6:
      /home/user/project/main.go:38`

	setupMockCmdExit(t, "go", "", stderrOut, 1)

	tk := NewToolkit(t.TempDir())
	races, err := tk.RaceDetect("./...")
	require.NoError(t, err)
	require.Len(t, races, 1)
	assert.Equal(t, "/home/user/project/main.go", races[0].File)
	assert.Equal(t, 42, races[0].Line)
}

func TestToolkit_DiffStat_Good(t *testing.T) {
	output := ` file1.go | 10 +++++++---
 file2.go |  5 +++++
 2 files changed, 12 insertions(+), 3 deletions(-)`
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	s, err := tk.DiffStat()
	require.NoError(t, err)
	assert.Equal(t, 2, s.FilesChanged)
	assert.Equal(t, 12, s.Insertions)
	assert.Equal(t, 3, s.Deletions)
}

func TestToolkit_CheckPerms_Good(t *testing.T) {
	dir := t.TempDir()

	badFile := filepath.Join(dir, "bad.txt")
	require.NoError(t, os.WriteFile(badFile, []byte("test"), 0644))
	require.NoError(t, os.Chmod(badFile, 0666))

	goodFile := filepath.Join(dir, "good.txt")
	require.NoError(t, os.WriteFile(goodFile, []byte("test"), 0644))

	tk := NewToolkit("/")
	issues, err := tk.CheckPerms(dir)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "World-writable", issues[0].Issue)
}

func TestToolkit_FindTrackedComments_Compatibility(t *testing.T) {
	output := "pkg/file.go:12:TODO: fix this\n"
	setupMockCmd(t, "git", output)

	tk := NewToolkit(t.TempDir())
	comments, err := tk.FindTrackedComments("pkg")
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "pkg/file.go", comments[0].File)
	assert.Equal(t, 12, comments[0].Line)
	assert.Equal(t, "TODO", comments[0].Type)
	assert.Equal(t, "fix this", comments[0].Message)

	legacyComments, err := tk.FindTODOs("pkg")
	require.NoError(t, err)
	assert.Equal(t, comments, legacyComments)
}
