package lint

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

const (
	hookStartMarker = "# core-lint hook start"
	hookEndMarker   = "# core-lint hook end"
)

// RunInput is the DTO for `core-lint run` and the language/category shortcuts.
//
//	input := lint.RunInput{Path: ".", Output: "json", FailOn: "warning"}
type RunInput struct {
	Path     string   `json:"path"`
	Output   string   `json:"output,omitempty"`
	Config   string   `json:"config,omitempty"`
	FailOn   string   `json:"fail_on,omitempty"`
	Category string   `json:"category,omitempty"`
	Lang     string   `json:"lang,omitempty"`
	Hook     bool     `json:"hook,omitempty"`
	CI       bool     `json:"ci,omitempty"`
	Files    []string `json:"files,omitempty"`
	SBOM     bool     `json:"sbom,omitempty"`
}

// ToolInfo describes a supported linter tool and whether it is available in PATH.
type ToolInfo struct {
	Name        string   `json:"name"`
	Available   bool     `json:"available"`
	Languages   []string `json:"languages"`
	Category    string   `json:"category"`
	Entitlement string   `json:"entitlement,omitempty"`
}

// Report aggregates every tool run into a single output document.
type Report struct {
	Project   string    `json:"project"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration"`
	Languages []string  `json:"languages"`
	Tools     []ToolRun `json:"tools"`
	Findings  []Finding `json:"findings"`
	Summary   Summary   `json:"summary"`
}

// ToolRun records the execution status of one adapter.
type ToolRun struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Findings int    `json:"findings"`
}

// Service orchestrates the configured lint adapters for a project.
//
//	svc := lint.NewService()
//	report, err := svc.Run(ctx, lint.RunInput{Path: ".", Output: "json"})
type Service struct {
	adapters []Adapter
}

// NewService constructs a lint orchestrator with the built-in adapter registry.
func NewService() *Service {
	return &Service{adapters: defaultAdapters()}
}

// Run executes the selected adapters and returns the merged report.
func (s *Service) Run(ctx context.Context, input RunInput) (Report, error) {
	startedAt := time.Now().UTC()
	input = normaliseRunInput(input)

	config, _, err := LoadProjectConfig(input.Path, input.Config)
	if err != nil {
		return Report{}, err
	}
	if input.FailOn == "" {
		input.FailOn = config.FailOn
	}

	files := slices.Clone(input.Files)
	if input.Hook && len(files) == 0 {
		files, err = s.stagedFiles(input.Path)
		if err != nil {
			return Report{}, err
		}
	}

	languages := s.languagesForInput(input, files)
	selectedAdapters := s.selectAdapters(config, languages, input)

	var findings []Finding
	var toolRuns []ToolRun

	for _, adapter := range selectedAdapters {
		if input.Hook && !adapter.Fast() {
			toolRuns = append(toolRuns, ToolRun{
				Name:     adapter.Name(),
				Status:   "skipped",
				Duration: "0s",
				Findings: 0,
			})
			continue
		}

		result := adapter.Run(ctx, input, files)
		toolRuns = append(toolRuns, result.Tool)
		findings = append(findings, normaliseReportFindings(result.Findings, input.Path)...)
	}

	findings = dedupeFindings(findings)
	sortToolRuns(toolRuns)
	sortFindings(findings)

	report := Report{
		Project:   projectName(input.Path),
		Timestamp: startedAt,
		Duration:  time.Since(startedAt).Round(time.Millisecond).String(),
		Languages: slices.Clone(languages),
		Tools:     toolRuns,
		Findings:  findings,
		Summary:   Summarise(findings),
	}
	report.Summary.Passed = passesThreshold(report.Summary, input.FailOn)

	return report, nil
}

// Tools returns the current adapter inventory for display in the CLI.
func (s *Service) Tools(languages []string) []ToolInfo {
	var tools []ToolInfo
	for _, adapter := range s.adapters {
		if len(languages) > 0 && !adapter.MatchesLanguage(languages) {
			continue
		}
		tools = append(tools, ToolInfo{
			Name:        adapter.Name(),
			Available:   adapter.Available(),
			Languages:   slices.Clone(adapter.Languages()),
			Category:    adapter.Category(),
			Entitlement: adapter.Entitlement(),
		})
	}
	slices.SortFunc(tools, func(left ToolInfo, right ToolInfo) int {
		return strings.Compare(left.Name, right.Name)
	})
	return tools
}

// WriteDefaultConfig creates `.core/lint.yaml` in the target project.
//
//	path, err := svc.WriteDefaultConfig(".", false)
func (s *Service) WriteDefaultConfig(projectPath string, force bool) (string, error) {
	if projectPath == "" {
		projectPath = "."
	}

	targetPath := filepath.Join(projectPath, DefaultConfigPath)
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return "", coreerr.E("Service.WriteDefaultConfig", targetPath+" already exists", nil)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", coreerr.E("Service.WriteDefaultConfig", "mkdir "+filepath.Dir(targetPath), err)
	}

	content, err := DefaultConfigYAML()
	if err != nil {
		return "", err
	}
	if err := coreio.Local.Write(targetPath, content); err != nil {
		return "", coreerr.E("Service.WriteDefaultConfig", "write "+targetPath, err)
	}

	return targetPath, nil
}

// InstallHook adds a git pre-commit hook that runs `core-lint run --hook`.
func (s *Service) InstallHook(projectPath string) error {
	hookPath, err := hookFilePath(projectPath)
	if err != nil {
		return err
	}

	block := hookScriptBlock(false)
	content := "#!/bin/sh\n" + block

	raw, readErr := coreio.Local.Read(hookPath)
	if readErr == nil {
		if strings.Contains(raw, hookStartMarker) {
			return nil
		}

		trimmed := strings.TrimRight(raw, "\n")
		if trimmed == "" {
			content = "#!/bin/sh\n" + block
		} else {
			content = trimmed + "\n\n" + hookScriptBlock(true)
		}
	}

	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		return coreerr.E("Service.InstallHook", "mkdir "+filepath.Dir(hookPath), err)
	}
	if err := coreio.Local.Write(hookPath, content); err != nil {
		return coreerr.E("Service.InstallHook", "write "+hookPath, err)
	}
	if err := os.Chmod(hookPath, 0o755); err != nil {
		return coreerr.E("Service.InstallHook", "chmod "+hookPath, err)
	}

	return nil
}

// RemoveHook removes the block previously installed by InstallHook.
func (s *Service) RemoveHook(projectPath string) error {
	hookPath, err := hookFilePath(projectPath)
	if err != nil {
		return err
	}

	raw, err := coreio.Local.Read(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return coreerr.E("Service.RemoveHook", "read "+hookPath, err)
	}

	startIndex := strings.Index(raw, hookStartMarker)
	endIndex := strings.Index(raw, hookEndMarker)
	if startIndex < 0 || endIndex < 0 || endIndex < startIndex {
		return nil
	}

	endIndex += len(hookEndMarker)
	content := strings.TrimRight(raw[:startIndex]+raw[endIndex:], "\n")
	if strings.TrimSpace(content) == "" {
		if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
			return coreerr.E("Service.RemoveHook", "remove "+hookPath, err)
		}
		return nil
	}

	if err := coreio.Local.Write(hookPath, content); err != nil {
		return coreerr.E("Service.RemoveHook", "write "+hookPath, err)
	}
	return nil
}

func (s *Service) languagesForInput(input RunInput, files []string) []string {
	if input.Lang != "" {
		return []string{input.Lang}
	}
	if len(files) > 0 {
		return detectFromFiles(files)
	}
	return Detect(input.Path)
}

func (s *Service) selectAdapters(config LintConfig, languages []string, input RunInput) []Adapter {
	enabled := make(map[string]bool)
	for _, name := range enabledToolNames(config, languages, input) {
		enabled[name] = true
	}

	var selected []Adapter
	for _, adapter := range s.adapters {
		if len(enabled) > 0 && !enabled[adapter.Name()] {
			continue
		}
		if input.Category != "" && adapter.Category() != input.Category {
			continue
		}
		if !adapter.MatchesLanguage(languages) {
			continue
		}
		selected = append(selected, adapter)
	}

	if slices.Contains(languages, "go") && input.Category != "compliance" {
		if !hasAdapter(selected, "catalog") {
			selected = append([]Adapter{newCatalogAdapter()}, selected...)
		}
	}

	return selected
}

func (s *Service) stagedFiles(projectPath string) ([]string, error) {
	toolkit := NewToolkit(projectPath)
	stdout, stderr, exitCode, err := toolkit.Run("git", "diff", "--cached", "--name-only")
	if err != nil && exitCode != 0 {
		return nil, coreerr.E("Service.stagedFiles", "git diff --cached --name-only: "+strings.TrimSpace(stderr), err)
	}

	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}
	return files, nil
}

func enabledToolNames(config LintConfig, languages []string, input RunInput) []string {
	var names []string

	if input.Category == "security" {
		names = append(names, config.Lint.Security...)
		return dedupeStrings(names)
	}
	if input.Category == "compliance" {
		names = append(names, config.Lint.Compliance...)
		return dedupeStrings(names)
	}

	if input.Lang != "" {
		names = append(names, groupForLanguage(config.Lint, input.Lang)...)
		return dedupeStrings(names)
	}

	for _, language := range languages {
		names = append(names, groupForLanguage(config.Lint, language)...)
	}
	names = append(names, config.Lint.Infra...)
	if input.CI || input.Category == "security" {
		names = append(names, config.Lint.Security...)
	}
	if input.SBOM {
		names = append(names, config.Lint.Compliance...)
	}

	return dedupeStrings(names)
}

func groupForLanguage(groups ToolGroups, language string) []string {
	switch language {
	case "go":
		return groups.Go
	case "php":
		return groups.PHP
	case "js":
		return groups.JS
	case "ts":
		return groups.TS
	case "python":
		return groups.Python
	case "shell", "dockerfile", "yaml", "json", "markdown":
		return groups.Infra
	default:
		return nil
	}
}

func hookFilePath(projectPath string) (string, error) {
	if projectPath == "" {
		projectPath = "."
	}

	toolkit := NewToolkit(projectPath)
	stdout, stderr, exitCode, err := toolkit.Run("git", "rev-parse", "--git-dir")
	if err != nil && exitCode != 0 {
		return "", coreerr.E("hookFilePath", "git rev-parse --git-dir: "+strings.TrimSpace(stderr), err)
	}

	gitDir := strings.TrimSpace(stdout)
	if gitDir == "" {
		return "", coreerr.E("hookFilePath", "git directory is empty", nil)
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectPath, gitDir)
	}
	return filepath.Join(gitDir, "hooks", "pre-commit"), nil
}

func hookScriptBlock(appended bool) string {
	command := "exec core-lint run --hook"
	if appended {
		command = "core-lint run --hook || exit $?"
	}

	return hookStartMarker + "\n# Installed by core-lint\n" + command + "\n" + hookEndMarker + "\n"
}

func normaliseRunInput(input RunInput) RunInput {
	if input.Path == "" {
		input.Path = "."
	}
	if input.CI && input.Output == "" {
		input.Output = "github"
	}
	return input
}

func normaliseReportFindings(findings []Finding, projectPath string) []Finding {
	normalised := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if finding.Code == "" {
			finding.Code = finding.RuleID
		}
		if finding.Message == "" {
			finding.Message = finding.Title
		}
		if finding.Tool == "" {
			finding.Tool = "catalog"
		}
		if finding.Severity == "" {
			finding.Severity = "warning"
		} else {
			finding.Severity = normaliseSeverity(finding.Severity)
		}
		if finding.File != "" && projectPath != "" {
			if relativePath, err := filepath.Rel(projectPath, finding.File); err == nil && relativePath != "" && !strings.HasPrefix(relativePath, "..") {
				finding.File = filepath.ToSlash(relativePath)
			} else {
				finding.File = filepath.ToSlash(finding.File)
			}
		}
		normalised = append(normalised, finding)
	}
	return normalised
}

func projectName(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.Base(absolutePath)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool)
	var deduped []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		deduped = append(deduped, value)
	}
	return deduped
}

func hasAdapter(adapters []Adapter, name string) bool {
	for _, adapter := range adapters {
		if adapter.Name() == name {
			return true
		}
	}
	return false
}

func passesThreshold(summary Summary, threshold string) bool {
	switch strings.ToLower(strings.TrimSpace(threshold)) {
	case "", "error":
		return summary.Errors == 0
	case "warning":
		return summary.Errors == 0 && summary.Warnings == 0
	case "info":
		return summary.Total == 0
	default:
		return summary.Errors == 0
	}
}

func sortFindings(findings []Finding) {
	slices.SortFunc(findings, func(left Finding, right Finding) int {
		switch {
		case left.File != right.File:
			return strings.Compare(left.File, right.File)
		case left.Line != right.Line:
			if left.Line < right.Line {
				return -1
			}
			return 1
		case left.Column != right.Column:
			if left.Column < right.Column {
				return -1
			}
			return 1
		case left.Tool != right.Tool:
			return strings.Compare(left.Tool, right.Tool)
		default:
			return strings.Compare(left.Code, right.Code)
		}
	})
}

func sortToolRuns(toolRuns []ToolRun) {
	slices.SortFunc(toolRuns, func(left ToolRun, right ToolRun) int {
		return strings.Compare(left.Name, right.Name)
	})
}
