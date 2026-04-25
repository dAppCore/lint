package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRunOutputFormat_Good_Precedence(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`output: text
schedules:
  nightly:
    output: json
`), 0o644))

	format, err := ResolveRunOutputFormat(RunInput{
		Path:   dir,
		Output: "sarif",
		CI:     true,
	})
	require.NoError(t, err)
	assert.Equal(t, "sarif", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
		CI:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, "github", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	require.NoError(t, err)
	assert.Equal(t, "json", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	require.NoError(t, err)
	assert.Equal(t, "text", format)
}

func TestResolveRunOutputFormat_Good_ExplicitOutputBypassesConfigLoading(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project-file")
	require.NoError(t, os.WriteFile(projectPath, []byte("not a directory"), 0o644))

	format, err := ResolveRunOutputFormat(RunInput{
		Path:     projectPath,
		Output:   "sarif",
		Config:   "broken/config.yaml",
		Schedule: "nightly",
	})
	require.NoError(t, err)
	assert.Equal(t, "sarif", format)
}

func TestResolveRunOutputFormat_Bad_BrokenConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("{not: yaml"), 0o644))

	_, err := ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	assert.Error(t, err)
}

func TestResolveRunOutputFormat_Ugly_MissingSchedule(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("output: text\n"), 0o644))

	_, err := ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	assert.Error(t, err)
}
