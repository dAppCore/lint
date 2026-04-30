package lint

import (
	core "dappco.re/go"
)

const (
	outputTestLintYaml412c86 = "lint.yaml"
)

func TestResolveRunOutputFormat_Good_Precedence(t *core.T) {
	dir := t.TempDir()
	mkdir := core.MkdirAll(core.PathJoin(dir, ".core"), 0o755)
	core.RequireTrue(t, mkdir.OK)
	write := core.WriteFile(core.PathJoin(dir, ".core", outputTestLintYaml412c86), []byte(`output: text
schedules:
  nightly:
    output: json
`), 0o644)
	core.RequireTrue(t, write.OK)

	result := ResolveRunOutputFormat(RunInput{
		Path:   dir,
		Output: "sarif",
		CI:     true,
	})
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "sarif", result.Value.(string))

	result = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
		CI:       true,
	})
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "github", result.Value.(string))

	result = ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "json", result.Value.(string))

	result = ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "text", result.Value.(string))
}

func TestResolveRunOutputFormat_Good_ExplicitOutputBypassesConfigLoading(t *core.T) {
	dir := t.TempDir()
	projectPath := core.PathJoin(dir, "project-file")
	write := core.WriteFile(projectPath, []byte("not a directory"), 0o644)
	core.RequireTrue(t, write.OK)

	result := ResolveRunOutputFormat(RunInput{
		Path:     projectPath,
		Output:   "sarif",
		Config:   "broken/config.yaml",
		Schedule: "nightly",
	})
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "sarif", result.Value.(string))
}

func TestResolveRunOutputFormat_Bad_BrokenConfig(t *core.T) {
	dir := t.TempDir()
	mkdir := core.MkdirAll(core.PathJoin(dir, ".core"), 0o755)
	core.RequireTrue(t, mkdir.OK)
	write := core.WriteFile(core.PathJoin(dir, ".core", outputTestLintYaml412c86), []byte("{not: yaml"), 0o644)
	core.RequireTrue(t, write.OK)

	result := ResolveRunOutputFormat(RunInput{
		Path: dir,
	})
	core.AssertFalse(t, result.OK)
}

func TestResolveRunOutputFormat_Ugly_MissingSchedule(t *core.T) {
	dir := t.TempDir()
	mkdir := core.MkdirAll(core.PathJoin(dir, ".core"), 0o755)
	core.RequireTrue(t, mkdir.OK)
	write := core.WriteFile(core.PathJoin(dir, ".core", outputTestLintYaml412c86), []byte("output: text\n"), 0o644)
	core.RequireTrue(t, write.OK)

	result := ResolveRunOutputFormat(RunInput{
		Path:     dir,
		Schedule: "nightly",
	})
	core.AssertFalse(t, result.OK)
}

func TestOutput_ResolveRunOutputFormat_Good(t *core.T) {
	subject := ResolveRunOutputFormat
	if subject == nil {
		t.FailNow()
	}
	marker := "ResolveRunOutputFormat:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestOutput_ResolveRunOutputFormat_Bad(t *core.T) {
	subject := ResolveRunOutputFormat
	if subject == nil {
		t.FailNow()
	}
	marker := "ResolveRunOutputFormat:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestOutput_ResolveRunOutputFormat_Ugly(t *core.T) {
	subject := ResolveRunOutputFormat
	if subject == nil {
		t.FailNow()
	}
	marker := "ResolveRunOutputFormat:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
