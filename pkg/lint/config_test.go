package lint

import (
	core "dappco.re/go"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configTestGolangciLint3dd70e = "golangci-lint"
	configTestLintYaml34d630     = "lint.yaml"
)

func TestConfig_DefaultConfig_Good(t *core.T) {
	cfg := DefaultConfig()

	core.AssertEqual(t, "json", cfg.Output)
	core.AssertEqual(t, "error", cfg.FailOn)
	core.AssertEqual(t, []string{"."}, cfg.Paths)
	core.AssertEqual(t, []string{"vendor/", "node_modules/", ".core/"}, cfg.Exclude)
	core.AssertContains(t, cfg.Lint.Go, configTestGolangciLint3dd70e)
	core.AssertContains(t, cfg.Lint.Security, "semgrep")
}

func TestConfig_DefaultConfig_Bad(t *core.T) {
	cfg := DefaultConfig()
	cfg.Lint.Go[0] = "mutated"
	cfg.Paths[0] = "mutated"

	fresh := DefaultConfig()
	core.AssertEqual(t, configTestGolangciLint3dd70e, fresh.Lint.Go[0])
	core.AssertEqual(t, ".", fresh.Paths[0])
}

func TestConfig_DefaultConfig_Ugly(t *core.T) {
	cfg := DefaultConfig()

	cfg.Lint.Go = append(cfg.Lint.Go, "extra-tool")
	cfg.Exclude = append(cfg.Exclude, "build/")

	fresh := DefaultConfig()
	core.AssertNotContains(t, fresh.Lint.Go, "extra-tool")
	core.AssertNotContains(t, fresh.Exclude, "build/")
}

func TestConfig_DefaultConfigYAML_Good(t *core.T) {
	raw, err := DefaultConfigYAML()
	core.RequireNoError(t, err)
	core.AssertContains(t, raw, "output: json")
	core.AssertContains(t, raw, "fail_on: error")

	var cfg LintConfig
	core.RequireNoError(t, yaml.Unmarshal([]byte(raw), &cfg))
	core.AssertEqual(t, DefaultConfig().Output, cfg.Output)
	core.AssertEqual(t, DefaultConfig().FailOn, cfg.FailOn)
	core.AssertEqual(t, DefaultConfig().Paths, cfg.Paths)
	core.AssertEqual(t, DefaultConfig().Exclude, cfg.Exclude)
}

func TestConfig_ResolveConfigPath_Good(t *core.T) {
	projectPath := t.TempDir()

	core.AssertEqual(t, core.CleanPath(filepath.Join(projectPath, ".core", configTestLintYaml34d630), "/"), ResolveConfigPath(projectPath, ""))
	core.AssertEqual(
		t,
		core.CleanPath(filepath.Join(projectPath, "config", configTestLintYaml34d630), "/"),
		ResolveConfigPath(projectPath, filepath.Join("config", configTestLintYaml34d630)),
	)
}

func TestConfig_ResolveConfigPath_Bad(t *core.T) {
	path := ResolveConfigPath("", "")
	core.AssertEqual(t, core.CleanPath(filepath.Join(".core", configTestLintYaml34d630), "/"), path)
	core.AssertContains(t, path, DefaultConfigPath)
}

func TestConfig_ResolveConfigPath_Ugly(t *core.T) {
	absolute := filepath.Join(t.TempDir(), "nested", configTestLintYaml34d630)
	path := ResolveConfigPath(t.TempDir(), absolute)
	core.AssertEqual(t, absolute, path)
	core.AssertContains(t, path, configTestLintYaml34d630)
}

func TestConfig_LoadProjectConfig_Good(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", configTestLintYaml34d630), []byte(`output: sarif
fail_on: warning
paths:
  - src
exclude:
  - generated/
`), 0o644))

	cfg, path, err := LoadProjectConfig(dir, "")
	core.RequireNoError(t, err)
	core.AssertEqual(t, filepath.Join(dir, ".core", configTestLintYaml34d630), path)
	core.AssertEqual(t, "sarif", cfg.Output)
	core.AssertEqual(t, "warning", cfg.FailOn)
	core.AssertEqual(t, []string{"src"}, cfg.Paths)
	core.AssertEqual(t, []string{"generated/"}, cfg.Exclude)
	core.AssertEqual(t, configTestGolangciLint3dd70e, cfg.Lint.Go[0])
}

func TestConfig_LoadProjectConfig_Bad(t *core.T) {
	dir := t.TempDir()

	cfg, path, err := LoadProjectConfig(dir, "")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, path)
	core.AssertEqual(t, DefaultConfig().Output, cfg.Output)
	core.AssertEqual(t, DefaultConfig().FailOn, cfg.FailOn)
}

func TestConfig_LoadProjectConfig_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", configTestLintYaml34d630), []byte("{not: yaml"), 0o644))

	_, path, err := LoadProjectConfig(dir, "")
	core.AssertError(t, err)
	core.AssertEmpty(t, path)
}

func TestConfig_ResolveSchedule_Good(t *core.T) {
	cfg := DefaultConfig()
	cfg.Schedules = map[string]Schedule{
		"nightly": {
			Cron:   "0 2 * * *",
			Output: "json",
		},
	}

	schedule, err := ResolveSchedule(cfg, "nightly")
	core.RequireNoError(t, err)
	RequireNotNil(t, schedule)
	core.AssertEqual(t, "0 2 * * *", schedule.Cron)
	core.AssertEqual(t, "json", schedule.Output)
}

func TestConfig_ResolveSchedule_Bad(t *core.T) {
	schedule, err := ResolveSchedule(DefaultConfig(), "missing")
	core.AssertError(t, err)
	core.AssertNil(t, schedule)
}

func TestConfig_ResolveSchedule_Ugly(t *core.T) {
	schedule, err := ResolveSchedule(DefaultConfig(), "")
	core.RequireNoError(t, err)
	core.AssertNil(t, schedule)
	core.AssertEqual(t, (*Schedule)(nil), schedule)
}
