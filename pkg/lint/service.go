package lint

import (
	"context"
	"io/fs"
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
//	input := lint.RunInput{Path: ".", Schedule: "nightly", Output: "json"}
//	report, err := lint.NewService().Run(ctx, input)
type RunInput struct {
	Path     string   `json:"path"`
	Output   string   `json:"output,omitempty"`
	Config   string   `json:"config,omitempty"`
	Schedule string   `json:"schedule,omitempty"`
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
//
//	svc := lint.NewService()
func NewService() *Service {
	return &Service{adapters: defaultAdapters()}
}

// Run executes the selected adapters and returns the merged report.
//
//	report, err := lint.NewService().Run(ctx, lint.RunInput{Path: ".", Output: "json"})
func (s *Service) Run(ctx context.Context, input RunInput) (Report, error) {
	startedAt := time.Now().UTC()
	input = normaliseRunInput(input)

	config, _, err := LoadProjectConfig(input.Path, input.Config)
	if err != nil {
		return Report{}, err
	}
	schedule, err := ResolveSchedule(config, input.Schedule)
	if err != nil {
		return Report{}, err
	}
	if input.FailOn == "" && schedule != nil && schedule.FailOn != "" {
		input.FailOn = schedule.FailOn
	}
	if input.FailOn == "" {
		input.FailOn = config.FailOn
	}

	files, scoped, err := s.scopeFiles(input.Path, config, input, schedule)
	if err != nil {
		return Report{}, err
	}
	if input.Hook && len(files) == 0 {
		report := Report{
			Project:   projectName(input.Path),
			Timestamp: startedAt,
			Duration:  time.Since(startedAt).Round(time.Millisecond).String(),
			Languages: []string{},
			Tools:     []ToolRun{},
			Findings:  []Finding{},
			Summary:   Summarise(nil),
		}
		report.Summary.Passed = passesThreshold(report.Summary, input.FailOn)
		return report, nil
	}
	if scoped && len(files) == 0 {
		report := Report{
			Project:   projectName(input.Path),
			Timestamp: startedAt,
			Duration:  time.Since(startedAt).Round(time.Millisecond).String(),
			Languages: []string{},
			Tools:     []ToolRun{},
			Findings:  []Finding{},
			Summary:   Summarise(nil),
		}
		report.Summary.Passed = passesThreshold(report.Summary, input.FailOn)
		return report, nil
	}

	languages := s.languagesForInput(input, files, scoped)
	selectedAdapters := s.selectAdapters(config, languages, input, schedule)

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
	if languages == nil {
		languages = []string{}
	}
	if toolRuns == nil {
		toolRuns = []ToolRun{}
	}
	if findings == nil {
		findings = []Finding{}
	}

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
//
//	tools := lint.NewService().Tools([]string{"go"})
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
	if tools == nil {
		return []ToolInfo{}
	}
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
//
//	_ = lint.NewService().InstallHook(".")
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
//
//	_ = lint.NewService().RemoveHook(".")
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

func (s *Service) languagesForInput(input RunInput, files []string, scoped bool) []string {
	if input.Lang != "" {
		return []string{input.Lang}
	}
	if scoped {
		return detectFromFiles(files)
	}
	return Detect(input.Path)
}

func (s *Service) scopeFiles(projectPath string, config LintConfig, input RunInput, schedule *Schedule) ([]string, bool, error) {
	if len(input.Files) > 0 {
		return slices.Clone(input.Files), true, nil
	}
	if input.Hook {
		files, err := s.stagedFiles(projectPath)
		return files, true, err
	}
	if schedule != nil && len(schedule.Paths) > 0 {
		files, err := collectConfiguredFiles(projectPath, schedule.Paths, config.Exclude)
		return files, true, err
	}
	if !slices.Equal(config.Paths, DefaultConfig().Paths) || !slices.Equal(config.Exclude, DefaultConfig().Exclude) {
		files, err := collectConfiguredFiles(projectPath, config.Paths, config.Exclude)
		return files, true, err
	}
	return nil, false, nil
}

func (s *Service) selectAdapters(config LintConfig, languages []string, input RunInput, schedule *Schedule) []Adapter {
	categories := selectedCategories(input, schedule)
	enabled := make(map[string]bool)
	for _, name := range enabledToolNames(config, languages, input, categories) {
		enabled[name] = true
	}

	var selected []Adapter
	for _, adapter := range s.adapters {
		if len(enabled) > 0 && !enabled[adapter.Name()] {
			continue
		}
		if len(categories) > 0 && !slices.Contains(categories, adapter.Category()) {
			continue
		}
		if !adapter.MatchesLanguage(languages) {
			continue
		}
		selected = append(selected, adapter)
	}

	if slices.Contains(languages, "go") && !slices.Contains(categories, "compliance") {
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

func collectConfiguredFiles(projectPath string, paths []string, excludes []string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string

	for _, path := range paths {
		if path == "" {
			continue
		}

		absolutePath := path
		if !filepath.IsAbs(absolutePath) {
			absolutePath = filepath.Join(projectPath, path)
		}

		info, err := os.Stat(absolutePath)
		if err != nil {
			return nil, coreerr.E("collectConfiguredFiles", "stat "+absolutePath, err)
		}
		if info.IsDir() && shouldSkipTraversalRoot(absolutePath) {
			continue
		}

		addFile := func(candidate string) {
			relativePath := relativeConfiguredPath(projectPath, candidate)
			if matchesConfiguredExclude(relativePath, excludes) || matchesConfiguredExclude(filepath.ToSlash(filepath.Clean(candidate)), excludes) {
				return
			}
			if seen[relativePath] {
				return
			}
			seen[relativePath] = true
			files = append(files, relativePath)
		}

		if !info.IsDir() {
			addFile(absolutePath)
			continue
		}

		walkErr := filepath.WalkDir(absolutePath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				relativeDir := relativeConfiguredPath(projectPath, currentPath)
				if matchesConfiguredExclude(relativeDir, excludes) || matchesConfiguredExclude(filepath.ToSlash(filepath.Clean(currentPath)), excludes) {
					return filepath.SkipDir
				}
				if currentPath != absolutePath && IsExcludedDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			addFile(currentPath)
			return nil
		})
		if walkErr != nil {
			return nil, coreerr.E("collectConfiguredFiles", "walk "+absolutePath, walkErr)
		}
	}

	slices.Sort(files)
	return files, nil
}

func relativeConfiguredPath(projectPath string, candidate string) string {
	relativePath := candidate
	if projectPath != "" {
		if rel, relErr := filepath.Rel(projectPath, candidate); relErr == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			relativePath = rel
		}
	}
	return filepath.ToSlash(filepath.Clean(relativePath))
}

func matchesConfiguredExclude(candidate string, excludes []string) bool {
	if candidate == "" || len(excludes) == 0 {
		return false
	}

	normalisedCandidate := filepath.ToSlash(filepath.Clean(candidate))
	for _, exclude := range excludes {
		normalisedExclude := filepath.ToSlash(filepath.Clean(strings.TrimSpace(exclude)))
		if normalisedExclude == "." || normalisedExclude == "" {
			continue
		}
		normalisedExclude = strings.TrimSuffix(normalisedExclude, "/")
		if normalisedCandidate == normalisedExclude {
			return true
		}
		if strings.HasPrefix(normalisedCandidate, normalisedExclude+"/") {
			return true
		}
	}
	return false
}

func enabledToolNames(config LintConfig, languages []string, input RunInput, categories []string) []string {
	var names []string

	if slices.Contains(categories, "security") {
		names = append(names, config.Lint.Security...)
	}
	if slices.Contains(categories, "compliance") {
		names = append(names, config.Lint.Compliance...)
	}

	if input.Lang != "" {
		names = append(names, groupForLanguage(config.Lint, input.Lang)...)
	} else if shouldIncludeLanguageGroups(categories) {
		for _, language := range languages {
			names = append(names, groupForLanguage(config.Lint, language)...)
		}
	}

	if shouldIncludeInfraGroups(categories) {
		names = append(names, config.Lint.Infra...)
	}
	if input.Lang == "" {
		if input.CI {
			names = append(names, config.Lint.Security...)
		}
		if input.SBOM {
			names = append(names, config.Lint.Compliance...)
		}
	}

	return dedupeStrings(names)
}

func selectedCategories(input RunInput, schedule *Schedule) []string {
	if input.Category != "" {
		return []string{input.Category}
	}
	if schedule == nil {
		return nil
	}
	return slices.Clone(schedule.Categories)
}

func shouldIncludeLanguageGroups(categories []string) bool {
	if len(categories) == 0 {
		return true
	}
	for _, category := range categories {
		switch category {
		case "security", "compliance":
			continue
		default:
			return true
		}
	}
	return false
}

func shouldIncludeInfraGroups(categories []string) bool {
	if len(categories) == 0 {
		return true
	}
	for _, category := range categories {
		switch category {
		case "security", "compliance":
			continue
		default:
			return true
		}
	}
	return false
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
