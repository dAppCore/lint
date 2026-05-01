package lint

import (
	"context"
	"io/fs"
	"slices"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	serviceServiceInstallhooke9c9af        = "Service.InstallHook"
	serviceServiceRemovehook715b44         = "Service.RemoveHook"
	serviceServiceWritedefaultconfigd06c2e = "Service.WriteDefaultConfig"
	serviceWrite23fbc8                     = "write "
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
	Path     string   `json:"file_path"`
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
//
//	tools := lint.NewService().Tools([]string{"go"})
type ToolInfo struct {
	Name        string   `json:"name"`
	Available   bool     `json:"available"`
	Languages   []string `json:"languages"`
	Category    string   `json:"category"`
	Entitlement string   `json:"entitlement,omitempty"`
}

// Report aggregates every tool run into a single output document.
//
//	result := lint.NewService().Run(context.Background(), lint.RunInput{Path: ".", Output: "json"})
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

// LintConfigOptions configures the lint service. Empty config gives
// the built-in adapter registry; reserved for future tunables (extra
// adapters, default exclusions, custom output paths) without breaking
// existing callers.
//
// Usage example: `cfg := lint.LintConfigOptions{}`
type LintConfigOptions struct{}

// Service orchestrates the configured lint adapters for a project and
// hosts the canonical Core registration handle. Embeds
// *core.ServiceRuntime[LintConfigOptions] for typed options access
// when constructed via NewServiceFor / Register; library callers use
// NewService() (no Core attachment) which leaves ServiceRuntime nil.
//
//	// Library use (no Core)
//	svc := lint.NewService()
//	result := svc.Run(ctx, lint.RunInput{Path: ".", Output: "json"})
//
//	// Core registration
//	c, _ := core.New(core.WithName("lint", lint.NewServiceFor(lint.LintConfigOptions{})))
type Service struct {
	*core.ServiceRuntime[LintConfigOptions]
	adapters      []Adapter
	registrations core.Once
}

// NewService constructs a lint orchestrator with the built-in adapter
// registry — library use, no Core attachment. ServiceRuntime is nil
// on the returned Service; OnStartup / handler methods will be no-ops.
//
//	svc := lint.NewService()
func NewService() *Service {
	return &Service{adapters: defaultAdapters()}
}

// NewServiceFor returns a factory that builds a Core-attached lint
// Service with the supplied LintConfigOptions and produces a *Service
// ready for c.Service() registration.
//
// Usage example: `c, _ := core.New(core.WithName("lint", lint.NewServiceFor(lint.LintConfigOptions{})))`
func NewServiceFor(config LintConfigOptions) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, config),
			adapters:       defaultAdapters(),
		})
	}
}

// Register builds the lint service with default LintConfigOptions and
// returns the service Result directly — the imperative-style
// alternative to NewServiceFor for consumers wiring services without
// WithName options.
//
// Usage example: `r := lint.Register(c); svc := r.Value.(*lint.Service)`
func Register(c *core.Core) core.Result {
	return NewServiceFor(LintConfigOptions{})(c)
}

// OnStartup registers the lint action handlers on the attached Core.
// No-op when ServiceRuntime is nil (library construction). Implements
// core.Startable. Idempotent via core.Once.
//
// Usage example: `r := svc.OnStartup(ctx)`
func (service *Service) OnStartup(context.Context) core.Result {
	if service == nil || service.ServiceRuntime == nil {
		return core.Ok(nil)
	}
	service.registrations.Do(func() {
		c := service.Core()
		if c == nil {
			return
		}
		c.Action("lint.run", service.handleRun)
		c.Action("lint.tools", service.handleTools)
		c.Action("lint.install_hook", service.handleInstallHook)
		c.Action("lint.remove_hook", service.handleRemoveHook)
		c.Action("lint.write_default_config", service.handleWriteDefaultConfig)
	})
	return core.Ok(nil)
}

// OnShutdown is a no-op — adapter lifecycles are bounded by individual
// Run calls. Implements core.Stoppable.
//
// Usage example: `r := svc.OnShutdown(ctx)`
func (service *Service) OnShutdown(context.Context) core.Result {
	return core.Ok(nil)
}

// handleRun — `lint.run` action handler. Reads opts.{path, output,
// config, schedule, fail_on, category, lang, hook, ci, sbom} as a
// RunInput and returns the resulting Report in r.Value.
//
// Usage example: `r := c.Action("lint.run").Run(ctx, core.NewOptions(core.Option{Key: "path", Value: "."}, core.Option{Key: "output", Value: "json"}))`
func (service *Service) handleRun(ctx core.Context, opts core.Options) core.Result {
	if service == nil {
		return core.Fail(core.E("lint.run", "service not initialised", nil))
	}
	return service.Run(ctx, RunInput{
		Path:     opts.String("path"),
		Output:   opts.String("output"),
		Config:   opts.String("config"),
		Schedule: opts.String("schedule"),
		FailOn:   opts.String("fail_on"),
		Category: opts.String("category"),
		Lang:     opts.String("lang"),
		Hook:     opts.Bool("hook"),
		CI:       opts.Bool("ci"),
		SBOM:     opts.Bool("sbom"),
	})
}

// handleTools — `lint.tools` action handler. Reads opts.languages
// ([]string) and returns []ToolInfo describing the supported linter
// tools and PATH availability in r.Value.
//
// Usage example: `r := c.Action("lint.tools").Run(ctx, core.NewOptions(core.Option{Key: "languages", Value: []string{"go"}}))`
func (service *Service) handleTools(_ core.Context, opts core.Options) core.Result {
	if service == nil {
		return core.Fail(core.E("lint.tools", "service not initialised", nil))
	}
	languages, _ := opts.Get("languages").Value.([]string)
	return core.Ok(service.Tools(languages))
}

// handleInstallHook — `lint.install_hook` action handler. Reads
// opts.path and installs the lint pre-commit hook into the project's
// git hook directory.
//
// Usage example: `r := c.Action("lint.install_hook").Run(ctx, core.NewOptions(core.Option{Key: "path", Value: "."}))`
func (service *Service) handleInstallHook(_ core.Context, opts core.Options) core.Result {
	if service == nil {
		return core.Fail(core.E("lint.install_hook", "service not initialised", nil))
	}
	return service.InstallHook(opts.String("path"))
}

// handleRemoveHook — `lint.remove_hook` action handler. Reads
// opts.path and removes the lint pre-commit hook from the project's
// git hook directory.
//
// Usage example: `r := c.Action("lint.remove_hook").Run(ctx, core.NewOptions(core.Option{Key: "path", Value: "."}))`
func (service *Service) handleRemoveHook(_ core.Context, opts core.Options) core.Result {
	if service == nil {
		return core.Fail(core.E("lint.remove_hook", "service not initialised", nil))
	}
	return service.RemoveHook(opts.String("path"))
}

// handleWriteDefaultConfig — `lint.write_default_config` action
// handler. Reads opts.path + opts.force and writes the default
// .core/lint.yaml config to the project root.
//
// Usage example: `r := c.Action("lint.write_default_config").Run(ctx, core.NewOptions(core.Option{Key: "path", Value: "."}, core.Option{Key: "force", Value: true}))`
func (service *Service) handleWriteDefaultConfig(_ core.Context, opts core.Options) core.Result {
	if service == nil {
		return core.Fail(core.E("lint.write_default_config", "service not initialised", nil))
	}
	return service.WriteDefaultConfig(opts.String("path"), opts.Bool("force"))
}

// Run executes the selected adapters and returns the merged report.
//
//	result := lint.NewService().Run(ctx, lint.RunInput{Path: ".", Output: "json"})
func (service *Service) Run(ctx context.Context, input RunInput) core.Result {
	startedAt := time.Now().UTC()
	input = normaliseRunInput(input)

	configResult := LoadProjectConfig(input.Path, input.Config)
	if !configResult.OK {
		return configResult
	}
	config := configResult.Value.(projectConfigResult).Config
	scheduleResult := ResolveSchedule(config, input.Schedule)
	if !scheduleResult.OK {
		return scheduleResult
	}
	schedule, _ := scheduleResult.Value.(*Schedule)
	if input.FailOn == "" && schedule != nil && schedule.FailOn != "" {
		input.FailOn = schedule.FailOn
	}
	if input.FailOn == "" {
		input.FailOn = config.FailOn
	}

	scopeResult := service.scopeFiles(input.Path, config, input, schedule)
	if !scopeResult.OK {
		return scopeResult
	}
	scope := scopeResult.Value.(scopeFilesResult)
	files := scope.Files
	scoped := scope.Scoped
	if shouldReturnEmptyReport(input, files, scoped) {
		return core.Ok(emptyRunReport(input.Path, startedAt, input.FailOn))
	}

	languages := service.languagesForInput(input, files, scoped)
	selectedAdapters := service.selectAdapters(config, languages, input, schedule)
	findings, toolRuns, err := service.runAdapters(ctx, selectedAdapters, input, files)
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(buildRunReport(input, startedAt, languages, toolRuns, findings))
}

func shouldReturnEmptyReport(input RunInput, files []string, scoped bool) bool {
	return len(files) == 0 && (input.Hook || scoped)
}

func emptyRunReport(projectPath string, startedAt time.Time, failOn string) Report {
	report := Report{
		Project:   projectName(projectPath),
		Timestamp: startedAt,
		Duration:  time.Since(startedAt).Round(time.Millisecond).String(),
		Languages: []string{},
		Tools:     []ToolRun{},
		Findings:  []Finding{},
		Summary:   Summarise(nil),
	}
	report.Summary.Passed = passesThreshold(report.Summary, failOn)
	return report
}

func (service *Service) runAdapters(
	ctx context.Context,
	selectedAdapters []Adapter,
	input RunInput,
	files []string,
) ([]Finding, []ToolRun, error) {
	var findings []Finding
	var toolRuns []ToolRun

	for _, adapter := range selectedAdapters {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if input.Hook && !adapter.Fast() {
			toolRuns = append(toolRuns, skippedToolRun(adapter))
			continue
		}

		result := adapter.Run(ctx, input, files)
		toolRuns = append(toolRuns, result.Tool)
		findings = append(findings, normaliseReportFindings(result.Findings, input.Path)...)
	}
	return findings, toolRuns, nil
}

func skippedToolRun(adapter Adapter) ToolRun {
	return ToolRun{
		Name:     adapter.Name(),
		Status:   "skipped",
		Duration: "0s",
		Findings: 0,
	}
}

func buildRunReport(input RunInput, startedAt time.Time, languages []string, toolRuns []ToolRun, findings []Finding) Report {
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

	return report
}

// Tools returns the current adapter inventory for display in the CLI.
//
//	tools := lint.NewService().Tools([]string{"go"})
func (service *Service) Tools(languages []string) []ToolInfo {
	var tools []ToolInfo
	for _, adapter := range service.adapters {
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
		return core.Compare(left.Name, right.Name)
	})
	if tools == nil {
		return []ToolInfo{}
	}
	return tools
}

// WriteDefaultConfig creates `.core/lint.yaml` in the target project.
//
//	result := svc.WriteDefaultConfig(".", false)
func (service *Service) WriteDefaultConfig(projectPath string, force bool) core.Result {
	if projectPath == "" {
		projectPath = "."
	}

	targetPath := core.JoinPath(projectPath, DefaultConfigPath)
	if !force {
		if stat := core.Stat(targetPath); stat.OK {
			return core.Fail(core.E(serviceServiceWritedefaultconfigd06c2e, targetPath+" already exists", nil))
		}
	}

	if mkdir := core.MkdirAll(core.PathDir(targetPath), 0o755); !mkdir.OK {
		err, _ := mkdir.Value.(error)
		return core.Fail(core.E(serviceServiceWritedefaultconfigd06c2e, "mkdir "+core.PathDir(targetPath), err))
	}

	contentResult := DefaultConfigYAML()
	if !contentResult.OK {
		return contentResult
	}
	content := contentResult.Value.(string)
	if err := coreio.Local.Write(targetPath, content); err != nil {
		return core.Fail(core.E(serviceServiceWritedefaultconfigd06c2e, serviceWrite23fbc8+targetPath, err))
	}

	return core.Ok(targetPath)
}

// InstallHook adds a git pre-commit hook that runs `core-lint run --hook`.
//
//	result := lint.NewService().InstallHook(".")
func (service *Service) InstallHook(projectPath string) core.Result {
	hookPathResult := hookFilePath(projectPath)
	if !hookPathResult.OK {
		return hookPathResult
	}
	hookPath := hookPathResult.Value.(string)

	block := hookScriptBlock(false)
	content := "#!/bin/sh\n" + block

	raw, readErr := coreio.Local.Read(hookPath)
	if readErr == nil {
		if core.Contains(raw, hookStartMarker) {
			return core.Ok(nil)
		}

		trimmed := trimRightNewlines(raw)
		if trimmed == "" {
			content = "#!/bin/sh\n" + block
		} else {
			content = trimmed + "\n\n" + hookScriptBlock(true)
		}
	}

	if mkdir := core.MkdirAll(core.PathDir(hookPath), 0o755); !mkdir.OK {
		err, _ := mkdir.Value.(error)
		return core.Fail(core.E(serviceServiceInstallhooke9c9af, "mkdir "+core.PathDir(hookPath), err))
	}
	if err := coreio.Local.Write(hookPath, content); err != nil {
		return core.Fail(core.E(serviceServiceInstallhooke9c9af, serviceWrite23fbc8+hookPath, err))
	}
	fileResult := core.OpenFile(hookPath, core.O_RDWR, 0o755)
	if !fileResult.OK {
		return fileResult
	}
	file := fileResult.Value.(*core.OSFile)
	if err := file.Chmod(0o755); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return core.Fail(core.E(serviceServiceInstallhooke9c9af, "close "+hookPath, closeErr))
		}
		return core.Fail(core.E(serviceServiceInstallhooke9c9af, "chmod "+hookPath, err))
	}
	if err := file.Close(); err != nil {
		return core.Fail(core.E(serviceServiceInstallhooke9c9af, "close "+hookPath, err))
	}

	return core.Ok(nil)
}

// RemoveHook removes the block previously installed by InstallHook.
//
//	result := lint.NewService().RemoveHook(".")
func (service *Service) RemoveHook(projectPath string) core.Result {
	hookPathResult := hookFilePath(projectPath)
	if !hookPathResult.OK {
		return hookPathResult
	}
	hookPath := hookPathResult.Value.(string)

	raw, err := coreio.Local.Read(hookPath)
	if err != nil {
		if isNotExistError(err) {
			return core.Ok(nil)
		}
		return core.Fail(core.E(serviceServiceRemovehook715b44, "read "+hookPath, err))
	}

	startIndex := stringIndex(raw, hookStartMarker)
	endIndex := stringIndex(raw, hookEndMarker)
	if startIndex < 0 || endIndex < 0 || endIndex < startIndex {
		return core.Ok(nil)
	}

	endIndex += len(hookEndMarker)
	content := trimRightNewlines(raw[:startIndex] + raw[endIndex:])
	if core.Trim(content) == "" {
		if remove := core.Remove(hookPath); !remove.OK {
			err, _ := remove.Value.(error)
			if !core.IsNotExist(err) {
				return core.Fail(core.E(serviceServiceRemovehook715b44, "remove "+hookPath, err))
			}
		}
		return core.Ok(nil)
	}

	if err := coreio.Local.Write(hookPath, content); err != nil {
		return core.Fail(core.E(serviceServiceRemovehook715b44, serviceWrite23fbc8+hookPath, err))
	}
	return core.Ok(nil)
}

func trimRightNewlines(s string) string {
	for len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

func stringIndex(s string, needle string) int {
	if needle == "" {
		return 0
	}
	if len(needle) > len(s) {
		return -1
	}
	limit := len(s) - len(needle)
	for i := 0; i <= limit; i++ {
		if s[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func (service *Service) languagesForInput(input RunInput, files []string, scoped bool) []string {
	if input.Lang != "" {
		return []string{input.Lang}
	}
	if scoped {
		return detectFromFiles(files)
	}
	return Detect(input.Path)
}

type scopeFilesResult struct {
	Files  []string
	Scoped bool
}

func (service *Service) scopeFiles(projectPath string, config LintConfig, input RunInput, schedule *Schedule) core.Result {
	if input.Files != nil {
		return core.Ok(scopeFilesResult{Files: slices.Clone(input.Files), Scoped: true})
	}
	if input.Hook {
		filesResult := service.stagedFiles(projectPath)
		if !filesResult.OK {
			return filesResult
		}
		return core.Ok(scopeFilesResult{Files: filesResult.Value.([]string), Scoped: true})
	}
	if schedule != nil && len(schedule.Paths) > 0 {
		filesResult := collectConfiguredFiles(projectPath, schedule.Paths, config.Exclude)
		if !filesResult.OK {
			return filesResult
		}
		return core.Ok(scopeFilesResult{Files: filesResult.Value.([]string), Scoped: true})
	}
	if !slices.Equal(config.Paths, DefaultConfig().Paths) || !slices.Equal(config.Exclude, DefaultConfig().Exclude) {
		filesResult := collectConfiguredFiles(projectPath, config.Paths, config.Exclude)
		if !filesResult.OK {
			return filesResult
		}
		return core.Ok(scopeFilesResult{Files: filesResult.Value.([]string), Scoped: true})
	}
	return core.Ok(scopeFilesResult{})
}

func (service *Service) selectAdapters(config LintConfig, languages []string, input RunInput, schedule *Schedule) []Adapter {
	categories := selectedCategories(input, schedule)
	enabled := make(map[string]bool)
	for _, name := range enabledToolNames(config, languages, input, categories) {
		enabled[name] = true
	}

	var selected []Adapter
	for _, adapter := range service.adapters {
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

func (service *Service) stagedFiles(projectPath string) core.Result {
	toolkit := NewToolkit(projectPath)
	run := toolkit.Run("git", "diff", "--cached", "--name-only").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("Service.stagedFiles", "git diff --cached --name-only: "+core.Trim(run.Stderr), run.Err))
	}

	var files []string
	for _, line := range core.Split(core.Trim(run.Stdout), "\n") {
		line = core.Trim(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}
	return core.Ok(files)
}

func collectConfiguredFiles(projectPath string, paths []string, excludes []string) core.Result {
	collector := configuredFileCollector{
		projectPath: projectPath,
		excludes:    excludes,
		seen:        make(map[string]bool),
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if result := collector.collect(path); !result.OK {
			return result
		}
	}

	slices.Sort(collector.files)
	return core.Ok(collector.files)
}

type configuredFileCollector struct {
	projectPath string
	excludes    []string
	seen        map[string]bool
	files       []string
}

func (collector *configuredFileCollector) collect(path string) core.Result {
	absolutePath := collector.absolute(path)
	stat := core.Stat(absolutePath)
	if !stat.OK {
		err, _ := stat.Value.(error)
		return core.Fail(core.E("collectConfiguredFiles", "stat "+absolutePath, err))
	}
	info := stat.Value.(core.FsFileInfo)
	if info.IsDir() && shouldSkipTraversalRoot(absolutePath) {
		return core.Ok(nil)
	}
	if !info.IsDir() {
		collector.addFile(absolutePath)
		return core.Ok(nil)
	}
	return collector.walkDir(absolutePath)
}

func (collector *configuredFileCollector) absolute(path string) string {
	if core.PathIsAbs(path) {
		return path
	}
	return core.JoinPath(collector.projectPath, path)
}

func (collector *configuredFileCollector) addFile(candidate string) {
	relativePath := relativeConfiguredPath(collector.projectPath, candidate)
	if collector.shouldSkipFile(candidate, relativePath) {
		return
	}
	collector.seen[relativePath] = true
	collector.files = append(collector.files, relativePath)
}

func (collector *configuredFileCollector) shouldSkipFile(candidate string, relativePath string) bool {
	if hasHiddenDirectory(relativePath) || hasHiddenDirectory(cleanSlashPath(candidate)) {
		return true
	}
	if matchesConfiguredExclude(relativePath, collector.excludes) || matchesConfiguredExclude(cleanSlashPath(candidate), collector.excludes) {
		return true
	}
	return collector.seen[relativePath]
}

func (collector *configuredFileCollector) walkDir(absolutePath string) core.Result {
	walkErr := core.PathWalkDir(absolutePath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			result := collector.walkDirEntry(absolutePath, currentPath, entry)
			if !result.OK {
				if result.Value == fs.SkipDir {
					return fs.SkipDir
				}
				err, _ := result.Value.(error)
				return err
			}
			return nil
		}
		collector.addFile(currentPath)
		return nil
	})
	if walkErr != nil {
		return core.Fail(core.E("collectConfiguredFiles", "walk "+absolutePath, walkErr))
	}
	return core.Ok(nil)
}

func (collector *configuredFileCollector) walkDirEntry(absolutePath string, currentPath string, entry fs.DirEntry) core.Result {
	relativeDir := relativeConfiguredPath(collector.projectPath, currentPath)
	if matchesConfiguredExclude(relativeDir, collector.excludes) || matchesConfiguredExclude(cleanSlashPath(currentPath), collector.excludes) {
		return core.Fail(fs.SkipDir)
	}
	if currentPath != absolutePath && IsExcludedDir(entry.Name()) {
		return core.Fail(fs.SkipDir)
	}
	return core.Ok(nil)
}

func cleanSlashPath(path string) string {
	return core.CleanPath(core.Replace(path, "\\", "/"), "/")
}

func relativeConfiguredPath(projectPath string, candidate string) string {
	relativePath := candidate
	if projectPath != "" {
		relResult := core.PathRel(projectPath, candidate)
		if relResult.OK {
			rel := relResult.Value.(string)
			if rel != "" && !core.HasPrefix(rel, "..") {
				relativePath = rel
			}
		}
	}
	return cleanSlashPath(relativePath)
}

func matchesConfiguredExclude(candidate string, excludes []string) bool {
	if candidate == "" || len(excludes) == 0 {
		return false
	}

	normalisedCandidate := cleanSlashPath(candidate)
	for _, exclude := range excludes {
		normalisedExclude := cleanSlashPath(core.Trim(exclude))
		if normalisedExclude == "." || normalisedExclude == "" {
			continue
		}
		normalisedExclude = core.TrimSuffix(normalisedExclude, "/")
		if normalisedCandidate == normalisedExclude {
			return true
		}
		if core.HasPrefix(normalisedCandidate, normalisedExclude+"/") {
			return true
		}
	}
	return false
}

func hasHiddenDirectory(candidate string) bool {
	if candidate == "" {
		return false
	}

	for _, segment := range core.Split(cleanSlashPath(candidate), "/") {
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		if core.HasPrefix(segment, ".") {
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

	if input.Lang == "" && shouldIncludeInfraGroups(categories) {
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
	return shouldIncludeLanguageGroups(categories)
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

func hookFilePath(projectPath string) core.Result {
	if projectPath == "" {
		projectPath = "."
	}

	toolkit := NewToolkit(projectPath)
	run := toolkit.Run("git", "rev-parse", "--git-dir").Value.(CommandOutput)
	if run.Err != nil && run.ExitCode != 0 {
		return core.Fail(core.E("hookFilePath", "git rev-parse --git-dir: "+core.Trim(run.Stderr), run.Err))
	}

	gitDir := core.Trim(run.Stdout)
	if gitDir == "" {
		return core.Fail(core.E("hookFilePath", "git directory is empty", nil))
	}
	if !core.PathIsAbs(gitDir) {
		gitDir = core.JoinPath(projectPath, gitDir)
	}
	return core.Ok(core.JoinPath(gitDir, "hooks", "pre-commit"))
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
		normalised = append(normalised, normaliseReportFinding(finding, projectPath))
	}
	return normalised
}

func normaliseReportFinding(finding Finding, projectPath string) Finding {
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
	finding.File = normaliseFindingFile(finding.File, projectPath)
	return finding
}

func normaliseFindingFile(file string, projectPath string) string {
	if file == "" || projectPath == "" {
		return file
	}
	relative := core.PathRel(projectPath, file)
	if relative.OK {
		relativePath := relative.Value.(string)
		if relativePath != "" && !core.HasPrefix(relativePath, "..") {
			return cleanSlashPath(relativePath)
		}
	}
	return cleanSlashPath(file)
}

func projectName(path string) string {
	absolute := core.PathAbs(path)
	if !absolute.OK {
		return core.PathBase(path)
	}
	return core.PathBase(absolute.Value.(string))
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
	switch core.Lower(core.Trim(threshold)) {
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
			return core.Compare(left.File, right.File)
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
			return core.Compare(left.Tool, right.Tool)
		default:
			return core.Compare(left.Code, right.Code)
		}
	})
}

func sortToolRuns(toolRuns []ToolRun) {
	slices.SortFunc(toolRuns, func(left ToolRun, right ToolRun) int {
		return core.Compare(left.Name, right.Name)
	})
}
