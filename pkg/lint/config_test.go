package lint

import (
	"os"
	"path/filepath"
	"testing"

	core "dappco.re/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfig_DefaultConfig_Good(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "json", cfg.Output)
	assert.Equal(t, "error", cfg.FailOn)
	assert.Equal(t, []string{"."}, cfg.Paths)
	assert.Equal(t, []string{"vendor/", "node_modules/", ".core/"}, cfg.Exclude)
	assert.Contains(t, cfg.Lint.Go, "golangci-lint")
	assert.Contains(t, cfg.Lint.Security, "semgrep")
}

func TestConfig_DefaultConfig_Bad(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Lint.Go[0] = "mutated"
	cfg.Paths[0] = "mutated"

	fresh := DefaultConfig()
	assert.Equal(t, "golangci-lint", fresh.Lint.Go[0])
	assert.Equal(t, ".", fresh.Paths[0])
}

func TestConfig_DefaultConfig_Ugly(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Lint.Go = append(cfg.Lint.Go, "extra-tool")
	cfg.Exclude = append(cfg.Exclude, "build/")

	fresh := DefaultConfig()
	assert.NotContains(t, fresh.Lint.Go, "extra-tool")
	assert.NotContains(t, fresh.Exclude, "build/")
}

func TestConfig_DefaultConfigYAML_Good(t *testing.T) {
	raw, err := DefaultConfigYAML()
	require.NoError(t, err)
	assert.Contains(t, raw, "output: json")
	assert.Contains(t, raw, "fail_on: error")

	var cfg LintConfig
	require.NoError(t, yaml.Unmarshal([]byte(raw), &cfg))
	assert.Equal(t, DefaultConfig().Output, cfg.Output)
	assert.Equal(t, DefaultConfig().FailOn, cfg.FailOn)
	assert.Equal(t, DefaultConfig().Paths, cfg.Paths)
	assert.Equal(t, DefaultConfig().Exclude, cfg.Exclude)
}

func TestConfig_ResolveConfigPath_Good(t *testing.T) {
	projectPath := t.TempDir()

	assert.Equal(t, core.CleanPath(filepath.Join(projectPath, ".core", "lint.yaml"), "/"), ResolveConfigPath(projectPath, ""))
	assert.Equal(
		t,
		core.CleanPath(filepath.Join(projectPath, "config", "lint.yaml"), "/"),
		ResolveConfigPath(projectPath, filepath.Join("config", "lint.yaml")),
	)
}

func TestConfig_ResolveConfigPath_Bad(t *testing.T) {
	assert.Equal(t, core.CleanPath(filepath.Join(".core", "lint.yaml"), "/"), ResolveConfigPath("", ""))
}

func TestConfig_ResolveConfigPath_Ugly(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "nested", "lint.yaml")
	assert.Equal(t, absolute, ResolveConfigPath(t.TempDir(), absolute))
}

func TestConfig_LoadProjectConfig_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte(`output: sarif
fail_on: warning
paths:
  - src
exclude:
  - generated/
`), 0o644))

	cfg, path, err := LoadProjectConfig(dir, "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".core", "lint.yaml"), path)
	assert.Equal(t, "sarif", cfg.Output)
	assert.Equal(t, "warning", cfg.FailOn)
	assert.Equal(t, []string{"src"}, cfg.Paths)
	assert.Equal(t, []string{"generated/"}, cfg.Exclude)
	assert.Equal(t, "golangci-lint", cfg.Lint.Go[0])
}

func TestConfig_LoadProjectConfig_Bad(t *testing.T) {
	dir := t.TempDir()

	cfg, path, err := LoadProjectConfig(dir, "")
	require.NoError(t, err)
	assert.Empty(t, path)
	assert.Equal(t, DefaultConfig().Output, cfg.Output)
	assert.Equal(t, DefaultConfig().FailOn, cfg.FailOn)
}

func TestConfig_LoadProjectConfig_Ugly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".core", "lint.yaml"), []byte("{not: yaml"), 0o644))

	_, path, err := LoadProjectConfig(dir, "")
	assert.Error(t, err)
	assert.Empty(t, path)
}

func TestConfig_ResolveSchedule_Good(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedules = map[string]Schedule{
		"nightly": {
			Cron:   "0 2 * * *",
			Output: "json",
		},
	}

	schedule, err := ResolveSchedule(cfg, "nightly")
	require.NoError(t, err)
	require.NotNil(t, schedule)
	assert.Equal(t, "0 2 * * *", schedule.Cron)
	assert.Equal(t, "json", schedule.Output)
}

func TestConfig_ResolveSchedule_Bad(t *testing.T) {
	_, err := ResolveSchedule(DefaultConfig(), "missing")
	assert.Error(t, err)
}

func TestConfig_ResolveSchedule_Ugly(t *testing.T) {
	schedule, err := ResolveSchedule(DefaultConfig(), "")
	require.NoError(t, err)
	assert.Nil(t, schedule)
}
