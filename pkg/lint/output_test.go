package lint

import (
	core "dappco.re/go"
	"os"
	"path/filepath"
)

func TestResolveRunOutputFormat_Good_Precedence(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`output: text
schedules:
  nightly:
    output: json
`), 0o644))

	format, err := ResolveRunOutputFormat(RunInput{
		Path:   dir,
		Output: "sarif",
		CI:     true,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "sarif", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
		CI:       true,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "github", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "json", format)

	format, err = ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "text", format)
}

func TestResolveRunOutputFormat_Good_ExplicitOutputBypassesConfigLoading(t *core.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project-file")
	core.RequireNoError(t, os.WriteFile(projectPath, []byte("not a directory"), 0o644))

	format, err := ResolveRunOutputFormat(RunInput{
		Path:     projectPath,
		Output:   "sarif",
		Config:   "broken/config.yaml",
		Schedule: "nightly",
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "sarif", format)
}

func TestResolveRunOutputFormat_Bad_BrokenConfig(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("{not: yaml"), 0o644))

	_, err := ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	core.AssertError(t, err)
}

func TestResolveRunOutputFormat_Ugly_MissingSchedule(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("output: text\n"), 0o644))

	_, err := ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	core.AssertError(t, err)
}
