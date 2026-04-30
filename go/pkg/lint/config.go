package lint

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

// DefaultConfigPath is the repo-local config path used by core-lint.
const DefaultConfigPath = ".core/lint.yaml"

// LintConfig defines which tools run for each language and how results fail the build.
//
//	cfg := lint.DefaultConfig()
//	cfg.FailOn = "warning"
type LintConfig struct {
	Lint      ToolGroups          `yaml:"lint"      json:"lint"`
	Output    string              `yaml:"output"    json:"output"`
	FailOn    string              `yaml:"fail_on"   json:"fail_on"`
	Paths     []string            `yaml:"paths"     json:"paths"`
	Exclude   []string            `yaml:"exclude"   json:"exclude"`
	Schedules map[string]Schedule `yaml:"schedules,omitempty" json:"schedules,omitempty"`
}

// ToolGroups maps config groups to tool names.
type ToolGroups struct {
	Go         []string `yaml:"go,omitempty"         json:"go,omitempty"`
	PHP        []string `yaml:"php,omitempty"        json:"php,omitempty"`
	JS         []string `yaml:"js,omitempty"         json:"js,omitempty"`
	TS         []string `yaml:"ts,omitempty"         json:"ts,omitempty"`
	Python     []string `yaml:"python,omitempty"     json:"python,omitempty"`
	Infra      []string `yaml:"infra,omitempty"      json:"infra,omitempty"`
	Security   []string `yaml:"security,omitempty"   json:"security,omitempty"`
	Compliance []string `yaml:"compliance,omitempty" json:"compliance,omitempty"`
}

// Schedule declares a named lint run for external schedulers.
type Schedule struct {
	Cron       string   `yaml:"cron"                 json:"cron"`
	Categories []string `yaml:"categories,omitempty" json:"categories,omitempty"`
	Output     string   `yaml:"output,omitempty"     json:"output,omitempty"`
	Paths      []string `yaml:"paths,omitempty"      json:"paths,omitempty"`
	FailOn     string   `yaml:"fail_on,omitempty"    json:"fail_on,omitempty"`
}

type projectConfigResult struct {
	Config LintConfig
	Path   string
}

// DefaultConfig returns the RFC baseline config used when a repo has no local file yet.
//
//	cfg := lint.DefaultConfig()
//	cfg.Output = "sarif"
func DefaultConfig() LintConfig {
	return LintConfig{
		Lint: ToolGroups{
			Go: []string{
				"golangci-lint",
				"gosec",
				"govulncheck",
				"staticcheck",
				"revive",
				"errcheck",
			},
			PHP: []string{
				"phpstan",
				"psalm",
				"phpcs",
				"phpmd",
				"pint",
			},
			JS: []string{
				"biome",
				"oxlint",
				"eslint",
				"prettier",
			},
			TS: []string{
				"biome",
				"oxlint",
				"typescript",
			},
			Python: []string{
				"ruff",
				"mypy",
				"bandit",
				"pylint",
			},
			Infra: []string{
				"shellcheck",
				"hadolint",
				"yamllint",
				"jsonlint",
				"markdownlint",
			},
			Security: []string{
				"gitleaks",
				"trivy",
				"gosec",
				"bandit",
				"semgrep",
			},
			Compliance: []string{
				"syft",
				"grype",
				"scancode",
			},
		},
		Output:  "json",
		FailOn:  "error",
		Paths:   []string{"."},
		Exclude: []string{"vendor/", "node_modules/", ".core/"},
	}
}

// DefaultConfigYAML marshals the default config as the file content for `core-lint init`.
func DefaultConfigYAML() core.Result {
	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return core.Fail(core.E("DefaultConfigYAML", "marshal default config", err))
	}
	return core.Ok(string(data))
}

// ResolveConfigPath resolves an explicit config path or the repo-local default.
//
//	path := lint.ResolveConfigPath(".", "")
//	override := lint.ResolveConfigPath("/repo", ".core/lint.yaml")
func ResolveConfigPath(projectPath string, override string) string {
	if projectPath == "" {
		projectPath = "."
	}
	if override == "" {
		return core.CleanPath(core.JoinPath(projectPath, DefaultConfigPath), "/")
	}
	if core.PathIsAbs(override) {
		return override
	}
	return core.CleanPath(core.JoinPath(projectPath, override), "/")
}

// LoadProjectConfig reads `.core/lint.yaml` if present, otherwise returns the default config.
//
//	cfg, path, err := lint.LoadProjectConfig(".", "")
//	cfg, _, err = lint.LoadProjectConfig("/repo", ".core/lint.yaml")
func LoadProjectConfig(projectPath string, override string) core.Result {
	config := DefaultConfig()
	path := ResolveConfigPath(projectPath, override)

	_, err := coreio.Local.Stat(path)
	if err != nil {
		if isNotExistError(err) {
			return core.Ok(projectConfigResult{Config: config})
		}
		return core.Fail(core.E("LoadProjectConfig", "stat "+path, err))
	}

	raw, err := coreio.Local.Read(path)
	if err != nil {
		return core.Fail(core.E("LoadProjectConfig", "read "+path, err))
	}
	if err := yaml.Unmarshal([]byte(raw), &config); err != nil {
		return core.Fail(core.E("LoadProjectConfig", "parse "+path, err))
	}

	return core.Ok(projectConfigResult{Config: config, Path: path})
}

// ResolveSchedule returns a named schedule from the config.
//
//	schedule, err := lint.ResolveSchedule(cfg, "nightly")
func ResolveSchedule(config LintConfig, name string) core.Result {
	if name == "" {
		return core.Ok((*Schedule)(nil))
	}

	schedule, ok := config.Schedules[name]
	if !ok {
		return core.Fail(core.E("ResolveSchedule", "schedule "+name+" not found", nil))
	}

	return core.Ok(&schedule)
}
