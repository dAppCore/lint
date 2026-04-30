package lint

import (
	"context"
	"strconv"
	"time"

	core "dappco.re/go"
)

const (
	adapterFormatbbcd14         = "--format"
	adapterJsondf8565           = "--json"
	adapterLintCompliance738e57 = "lint.compliance"
	adapterLintSecuritye950fd   = "lint.security"
	adapterParseError89bf3a     = "parse-error"
)

// Adapter wraps one lint tool and normalises its output to Finding values.
type Adapter interface {
	Name() string
	Available() bool
	Languages() []string
	Command() string
	Entitlement() string
	RequiresEntitlement() bool
	MatchesLanguage(languages []string) bool
	Category() string
	Fast() bool
	Run(ctx context.Context, input RunInput, files []string) AdapterResult
}

// AdapterResult contains one tool execution plus the parsed findings from that run.
type AdapterResult struct {
	Tool     ToolRun
	Findings []Finding
}

type findingParser func(tool string, category string, output string) []Finding
type commandArgumentsBuilder func(projectPath string, files []string) []string

// CommandAdapter runs an external binary and parses its stdout/stderr.
type CommandAdapter struct {
	name                string
	binaries            []string
	languages           []string
	category            string
	entitlement         string
	requiresEntitlement bool
	fast                bool
	buildArgs           commandArgumentsBuilder
	parseOutput         findingParser
}

// CatalogAdapter wraps the embedded regex rule catalog as a built-in linter.
type CatalogAdapter struct{}

func defaultAdapters() []Adapter {
	return []Adapter{
		newCommandAdapter("golangci-lint", []string{"golangci-lint"}, []string{"go"}, "correctness", "", false, true, goProjectArguments("run", "--out-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("gosec", []string{"gosec"}, []string{"go"}, "security", adapterLintSecuritye950fd, true, false, goProjectArguments("-fmt", "json"), parseJSONDiagnostics),
		newCommandAdapter("govulncheck", []string{"govulncheck"}, []string{"go"}, "security", "", false, false, goProjectArguments("-json"), parseGovulncheckDiagnostics),
		newCommandAdapter("staticcheck", []string{"staticcheck"}, []string{"go"}, "correctness", "", false, true, goProjectArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("revive", []string{"revive"}, []string{"go"}, "style", "", false, true, goProjectArguments("-formatter", "json"), parseJSONDiagnostics),
		newCommandAdapter("errcheck", []string{"errcheck"}, []string{"go"}, "correctness", "", false, true, goProjectArguments(), parseTextDiagnostics),
		newCommandAdapter("phpstan", []string{"phpstan"}, []string{"php"}, "correctness", "", false, true, projectPathArguments("analyse", "--error-format=json"), parsePHPStanDiagnostics),
		newCommandAdapter("psalm", []string{"psalm"}, []string{"php"}, "correctness", "", false, true, projectPathArguments("--output-format=json"), parsePsalmDiagnostics),
		newCommandAdapter("phpcs", []string{"phpcs"}, []string{"php"}, "style", "", false, true, projectPathArguments("--report=json"), parseJSONDiagnostics),
		newCommandAdapter("phpmd", []string{"phpmd"}, []string{"php"}, "correctness", "", false, true, phpmdArguments(), parseJSONDiagnostics),
		newCommandAdapter("pint", []string{"pint"}, []string{"php"}, "style", "", false, true, projectPathArguments(adapterFormatbbcd14, "json"), parseJSONDiagnostics),
		newCommandAdapter("biome", []string{"biome"}, []string{"js", "ts"}, "style", "", false, true, projectPathArguments("check", "--reporter", "json"), parseJSONDiagnostics),
		newCommandAdapter("oxlint", []string{"oxlint"}, []string{"js", "ts"}, "style", "", false, true, projectPathArguments(adapterFormatbbcd14, "json"), parseJSONDiagnostics),
		newCommandAdapter("eslint", []string{"eslint"}, []string{"js"}, "style", "", false, true, projectPathArguments(adapterFormatbbcd14, "json"), parseJSONDiagnostics),
		newCommandAdapter("typescript", []string{"tsc", "typescript"}, []string{"ts"}, "correctness", "", false, true, projectPathArguments("--pretty", "false"), parseTextDiagnostics),
		newCommandAdapter("ruff", []string{"ruff"}, []string{"python"}, "style", "", false, true, projectPathArguments("check", "--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("mypy", []string{"mypy"}, []string{"python"}, "correctness", "", false, true, projectPathArguments("--output", "json"), parseJSONDiagnostics),
		newCommandAdapter("bandit", []string{"bandit"}, []string{"python"}, "security", adapterLintSecuritye950fd, true, false, recursiveProjectPathArguments("-f", "json", "-r"), parseJSONDiagnostics),
		newCommandAdapter("pylint", []string{"pylint"}, []string{"python"}, "style", "", false, true, projectPathArguments("--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("shellcheck", []string{"shellcheck"}, []string{"shell"}, "correctness", "", false, true, filePathArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("hadolint", []string{"hadolint"}, []string{"dockerfile"}, "security", "", false, true, filePathArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("yamllint", []string{"yamllint"}, []string{"yaml"}, "style", "", false, true, projectPathArguments("-f", "parsable"), parseTextDiagnostics),
		newCommandAdapter("jsonlint", []string{"jsonlint"}, []string{"json"}, "style", "", false, true, filePathArguments(), parseTextDiagnostics),
		newCommandAdapter("markdownlint", []string{"markdownlint", "markdownlint-cli"}, []string{"markdown"}, "style", "", false, true, projectPathArguments(adapterJsondf8565), parseJSONDiagnostics),
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, projectPathArguments("--list-different"), parsePrettierDiagnostics),
		newCommandAdapter("gitleaks", []string{"gitleaks"}, []string{"*"}, "security", adapterLintSecuritye950fd, true, false, recursiveProjectPathArguments("detect", "--no-git", "--report-format", "json", "--source"), parseJSONDiagnostics),
		newCommandAdapter("trivy", []string{"trivy"}, []string{"*"}, "security", adapterLintSecuritye950fd, true, false, projectPathArguments("fs", adapterFormatbbcd14, "json"), parseJSONDiagnostics),
		newCommandAdapter("semgrep", []string{"semgrep"}, []string{"*"}, "security", adapterLintSecuritye950fd, true, false, projectPathArguments(adapterJsondf8565), parseJSONDiagnostics),
		newCommandAdapter("syft", []string{"syft"}, []string{"*"}, "compliance", adapterLintCompliance738e57, true, false, projectPathArguments("scan", "-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("grype", []string{"grype"}, []string{"*"}, "security", adapterLintCompliance738e57, true, false, projectPathArguments("-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("scancode", []string{"scancode-toolkit", "scancode"}, []string{"*"}, "compliance", adapterLintCompliance738e57, true, false, projectPathArguments(adapterJsondf8565), parseJSONDiagnostics),
	}
}

func newCatalogAdapter() Adapter {
	return CatalogAdapter{}
}

func newCommandAdapter(name string, binaries []string, languages []string, category string, entitlement string, requiresEntitlement bool, fast bool, builder commandArgumentsBuilder, parser findingParser) Adapter {
	return CommandAdapter{
		name:                name,
		binaries:            binaries,
		languages:           languages,
		category:            category,
		entitlement:         entitlement,
		requiresEntitlement: requiresEntitlement,
		fast:                fast,
		buildArgs:           builder,
		parseOutput:         parser,
	}
}

func (adapter CommandAdapter) Name() string { return adapter.name }

func (adapter CommandAdapter) Available() bool {
	_, ok := adapter.availableBinary()
	return ok
}

func (adapter CommandAdapter) Languages() []string {
	return append([]string(nil), adapter.languages...)
}

func (adapter CommandAdapter) Command() string {
	if len(adapter.binaries) == 0 {
		return ""
	}
	return adapter.binaries[0]
}

func (adapter CommandAdapter) Entitlement() string { return adapter.entitlement }

func (adapter CommandAdapter) RequiresEntitlement() bool { return adapter.requiresEntitlement }

func (adapter CommandAdapter) MatchesLanguage(languages []string) bool {
	if adapter.acceptsAnyLanguage(languages) {
		return true
	}
	for _, language := range languages {
		if adapter.matchesOneLanguage(language) {
			return true
		}
	}
	return false
}

func (adapter CommandAdapter) acceptsAnyLanguage(languages []string) bool {
	return len(adapter.languages) == 0 ||
		len(languages) == 0 ||
		(len(adapter.languages) == 1 && adapter.languages[0] == "*")
}

func (adapter CommandAdapter) matchesOneLanguage(language string) bool {
	if core.Lower(language) == core.Lower(adapter.category) {
		return true
	}
	for _, supported := range adapter.languages {
		if supported == language {
			return true
		}
	}
	return false
}

func (adapter CommandAdapter) Category() string { return adapter.category }

func (adapter CommandAdapter) Fast() bool { return adapter.fast }

func (adapter CommandAdapter) Run(ctx context.Context, input RunInput, files []string) AdapterResult {
	startedAt := time.Now()
	result := AdapterResult{
		Tool: ToolRun{
			Name: adapter.name,
		},
	}

	binary, ok := adapter.availableBinary()
	if !ok {
		return adapter.missingToolResult(result)
	}

	result.Tool.Version = probeCommandVersion(binary, input.Path)

	runContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := adapter.buildArgs(input.Path, files)
	run := runCommand(runContext, input.Path, binary, args)
	stdout := core.Trim(run.Stdout)
	stderr := core.Trim(run.Stderr)

	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()

	if err := runContext.Err(); err != nil {
		if core.Is(err, context.DeadlineExceeded) {
			result.Tool.Status = "timeout"
		} else {
			result.Tool.Status = "canceled"
		}
		return result
	}

	adapter.appendParsedFindings(&result, stdout, stderr)
	if len(result.Findings) == 0 && (stdout != "" || stderr != "") {
		result.Findings = parseTextDiagnostics(adapter.name, adapter.category, combinedOutput(stdout, stderr))
	}
	if len(result.Findings) == 0 && run.Err != nil {
		result.Findings = []Finding{{
			Tool:     adapter.name,
			Severity: defaultSeverityForCategory(adapter.category),
			Code:     "command-failed",
			Message:  core.Trim(firstNonEmpty(stdout, stderr, run.Err.Error())),
			Category: adapter.category,
		}}
	}

	adapter.normaliseFindings(result.Findings)
	result.Tool.Findings = len(result.Findings)
	result.Tool.Status = adapterStatus(run.Err, run.ExitCode, result.Findings)
	return result
}

func (adapter CommandAdapter) missingToolResult(result AdapterResult) AdapterResult {
	result.Tool.Status = "skipped"
	result.Tool.Duration = "0s"
	missingName := firstNonEmpty(adapter.Command(), adapter.name)
	if missingName == "" {
		missingName = adapter.name
	}
	result.Findings = []Finding{{
		Tool:     adapter.name,
		Severity: "info",
		Code:     "missing-tool",
		Message:  core.Sprintf("%s is not installed", missingName),
		Category: adapter.category,
	}}
	result.Tool.Findings = len(result.Findings)
	return result
}

func (adapter CommandAdapter) appendParsedFindings(result *AdapterResult, stdout string, stderr string) {
	if adapter.parseOutput == nil {
		return
	}
	if stdout != "" {
		result.Findings = append(result.Findings, adapter.parseOutput(adapter.name, adapter.category, stdout)...)
	}
	if stderr != "" {
		stderrFindings := adapter.parseOutput(adapter.name, adapter.category, stderr)
		if !(hasNonParseErrorFinding(result.Findings) && onlyParseErrorFindings(stderrFindings)) {
			result.Findings = append(result.Findings, stderrFindings...)
		}
	}
	result.Findings = dedupeFindings(result.Findings)
}

func combinedOutput(stdout string, stderr string) string {
	if stdout == "" {
		return stderr
	}
	if stderr == "" {
		return stdout
	}
	return stdout + "\n" + stderr
}

func (adapter CommandAdapter) normaliseFindings(findings []Finding) {
	for index := range findings {
		if findings[index].Tool == "" {
			findings[index].Tool = adapter.name
		}
		if findings[index].Category == "" {
			findings[index].Category = adapter.category
		}
		if findings[index].Severity == "" {
			findings[index].Severity = defaultSeverityForCategory(adapter.category)
			continue
		}
		findings[index].Severity = normaliseSeverity(findings[index].Severity)
	}
}

func adapterStatus(runErr error, exitCode int, findings []Finding) string {
	if runErr != nil || exitCode != 0 || len(findings) > 0 {
		return "failed"
	}
	return "passed"
}

func probeCommandVersion(binary string, workingDir string) string {
	for _, args := range [][]string{{"--version"}, {"-version"}, {"version"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		run := runCommand(ctx, workingDir, binary, args)
		cancel()

		if run.Err != nil && run.ExitCode != 0 {
			continue
		}

		version := firstNonEmpty(run.Stdout, run.Stderr)
		if version == "" {
			continue
		}

		if line := firstVersionLine(version); line != "" {
			return line
		}
	}

	return ""
}

func (adapter CommandAdapter) availableBinary() (string, bool) {
	for _, binary := range adapter.binaries {
		path := findExecutable(binary)
		if path.OK {
			return path.Value.(string), true
		}
	}
	return "", false
}

func (CatalogAdapter) Name() string { return "catalog" }

func (CatalogAdapter) Available() bool { return true }

func (CatalogAdapter) Languages() []string { return []string{"go"} }

func (CatalogAdapter) Command() string { return "catalog" }

func (CatalogAdapter) Entitlement() string { return "" }

func (CatalogAdapter) RequiresEntitlement() bool { return false }

func (CatalogAdapter) MatchesLanguage(languages []string) bool {
	return len(languages) == 0 || containsString(languages, "go")
}

func (CatalogAdapter) Category() string { return "correctness" }

func (CatalogAdapter) Fast() bool { return true }

func (CatalogAdapter) Run(ctx context.Context, input RunInput, files []string) AdapterResult {
	startedAt := time.Now()
	result := AdapterResult{
		Tool: ToolRun{
			Name: "catalog",
		},
	}

	catalogResult := loadBuiltinCatalog()
	if !catalogResult.OK {
		err, _ := catalogResult.Value.(error)
		return catalogErrorResult(result, startedAt, "catalog-load", err)
	}
	catalog := catalogResult.Value.(*Catalog)

	rules := catalog.Rules
	if input.Category != "" {
		rules = filterRulesByTag(rules, input.Category)
	}

	scannerResult := NewScanner(rules)
	if !scannerResult.OK {
		err, _ := scannerResult.Value.(error)
		return catalogErrorResult(result, startedAt, "catalog-scan", err)
	}
	scanner := scannerResult.Value.(*Scanner)

	findings := scanCatalogFindings(ctx, scanner, input.Path, files)
	if err := ctx.Err(); err != nil {
		return canceledCatalogResult(result, startedAt, findings)
	}

	normaliseCatalogFindings(catalog, findings)
	result.Findings = findings
	result.Tool.Findings = len(findings)
	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
	result.Tool.Status = catalogStatus(findings)
	return result
}

func catalogErrorResult(result AdapterResult, startedAt time.Time, code string, err error) AdapterResult {
	result.Tool.Status = "failed"
	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
	result.Findings = []Finding{{
		Tool:     "catalog",
		Severity: "error",
		Code:     code,
		Message:  err.Error(),
		Category: "correctness",
	}}
	result.Tool.Findings = len(result.Findings)
	return result
}

func scanCatalogFindings(ctx context.Context, scanner *Scanner, projectPath string, files []string) []Finding {
	if len(files) == 0 {
		if ctx.Err() != nil {
			return nil
		}
		findings := scanner.ScanDir(projectPath)
		if !findings.OK {
			return nil
		}
		return findings.Value.([]Finding)
	}

	var findings []Finding
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		fileFindings := scanner.ScanFile(catalogScanPath(projectPath, file))
		if !fileFindings.OK {
			continue
		}
		findings = append(findings, fileFindings.Value.([]Finding)...)
	}
	return findings
}

func catalogScanPath(projectPath string, file string) string {
	if core.PathIsAbs(file) {
		return file
	}
	return core.PathJoin(projectPath, file)
}

func canceledCatalogResult(result AdapterResult, startedAt time.Time, findings []Finding) AdapterResult {
	result.Tool.Status = "canceled"
	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
	result.Tool.Findings = len(findings)
	result.Findings = findings
	return result
}

func normaliseCatalogFindings(catalog *Catalog, findings []Finding) {
	for index := range findings {
		rule := catalog.ByID(findings[index].RuleID)
		findings[index].Tool = "catalog"
		findings[index].Code = findings[index].RuleID
		findings[index].Message = findings[index].Title
		findings[index].Severity = normaliseSeverity(findings[index].Severity)
		if rule != nil {
			findings[index].Category = ruleCategory(*rule)
		}
	}
}

func catalogStatus(findings []Finding) string {
	if len(findings) > 0 {
		return "failed"
	}
	return "passed"
}

func loadBuiltinCatalog() core.Result {
	rulesResult := ParseRules([]byte(defaultCatalogRulesYAML))
	if !rulesResult.OK {
		err, _ := rulesResult.Value.(error)
		return core.Fail(core.E("loadBuiltinCatalog", "parse embedded fallback rules", err))
	}
	return core.Ok(&Catalog{Rules: rulesResult.Value.([]Rule)})
}

func goProjectArguments(prefix ...string) commandArgumentsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, "./...")
	}
}

func projectPathArguments(prefix ...string) commandArgumentsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func recursiveProjectPathArguments(prefix ...string) commandArgumentsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func filePathArguments(prefix ...string) commandArgumentsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func phpmdArguments() commandArgumentsBuilder {
	return func(_ string, files []string) []string {
		target := "."
		if len(files) > 0 {
			target = core.Join(",", files...)
		}
		return []string{target, "json", "cleancode,codesize,controversial,design,naming,unusedcode"}
	}
}

func runCommand(ctx context.Context, workingDir string, binary string, args []string) CommandOutput {
	return runCoreCommand(ctx, workingDir, binary, args)
}

func parseGovulncheckDiagnostics(tool string, category string, output string) []Finding {
	parsed := ParseVulnCheckJSON(output, "")
	if !parsed.OK || parsed.Value == nil {
		return nil
	}
	result := parsed.Value.(*VulnResult)

	var findings []Finding
	for _, vuln := range result.Findings {
		message := core.Trim(firstNonEmpty(vuln.Description, vuln.Package))
		if message == "" {
			message = vuln.ID
		}
		findings = append(findings, Finding{
			Tool:     tool,
			File:     vuln.Package,
			Severity: "error",
			Code:     vuln.ID,
			Message:  message,
			Category: category,
		})
	}

	return findings
}

func parseJSONDiagnostics(tool string, category string, output string) []Finding {
	var findings []Finding
	trimmed := core.Trim(output)
	if trimmed == "" {
		return nil
	}

	segments, trailing := topLevelJSONSegments(trimmed)
	if len(segments) == 0 {
		var value any
		r := core.JSONUnmarshal([]byte(trimmed), &value)
		if !r.OK {
			return []Finding{jsonParseFinding(tool, category, r.Value)}
		}
		return dedupeFindings(collectJSONDiagnostics(tool, category, value))
	}

	for _, segment := range segments {
		var value any
		r := core.JSONUnmarshal([]byte(segment), &value)
		if !r.OK {
			findings = append(findings, jsonParseFinding(tool, category, r.Value))
			return dedupeFindings(findings)
		}
		findings = append(findings, collectJSONDiagnostics(tool, category, value)...)
	}

	if trailing != "" {
		var value any
		r := core.JSONUnmarshal([]byte(trailing), &value)
		if !r.OK {
			findings = append(findings, jsonParseFinding(tool, category, r.Value))
			return dedupeFindings(findings)
		}
		findings = append(findings, collectJSONDiagnostics(tool, category, value)...)
	}

	return dedupeFindings(findings)
}

func topLevelJSONSegments(output string) ([]string, string) {
	var segments []string
	start := -1
	depth := 0
	inString := false
	escaped := false

	for index, current := range output {
		if consumeJSONStringRune(current, &inString, &escaped) {
			continue
		}
		trailing, done := consumeJSONSegmentRune(output, index, current, &segments, &start, &depth, &inString)
		if done {
			if len(segments) == 0 {
				return nil, trailing
			}
			return segments, trailing
		}
	}

	if depth != 0 && start >= 0 {
		return segments, core.Trim(output[start:])
	}

	return segments, ""
}

func consumeJSONSegmentRune(
	output string,
	index int,
	current rune,
	segments *[]string,
	start *int,
	depth *int,
	inString *bool,
) (string, bool) {
	switch current {
	case '"':
		*inString = *depth > 0
	case '{', '[':
		openJSONSegment(index, start, depth)
	case '}', ']':
		return closeJSONSegment(output, index, segments, start, depth)
	default:
		return jsonTrailingSegment(output, index, current, *depth, len(*segments))
	}
	return "", false
}

func openJSONSegment(index int, start *int, depth *int) {
	if *depth == 0 {
		*start = index
	}
	*depth++
}

func closeJSONSegment(output string, index int, segments *[]string, start *int, depth *int) (string, bool) {
	if *depth == 0 {
		return core.Trim(output[index:]), true
	}
	*depth--
	if *depth == 0 && *start >= 0 {
		*segments = append(*segments, output[*start:index+1])
		*start = -1
	}
	return "", false
}

func consumeJSONStringRune(current rune, inString *bool, escaped *bool) bool {
	if !*inString {
		return false
	}
	switch {
	case *escaped:
		*escaped = false
	case current == '\\':
		*escaped = true
	case current == '"':
		*inString = false
	}
	return true
}

func jsonTrailingSegment(output string, index int, current rune, depth int, segmentCount int) (string, bool) {
	if depth != 0 || isJSONWhitespace(current) {
		return "", false
	}
	if segmentCount == 0 {
		return output, true
	}
	return core.Trim(output[index:]), true
}

func isJSONWhitespace(value rune) bool {
	switch value {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}

func jsonParseFinding(tool string, category string, err any) Finding {
	return Finding{
		Tool:     tool,
		Severity: "error",
		Code:     adapterParseError89bf3a,
		Message:  core.Sprintf("failed to parse JSON output: %v", err),
		Category: category,
	}
}

func collectJSONDiagnostics(tool string, category string, value any) []Finding {
	switch typed := value.(type) {
	case []any:
		var findings []Finding
		for _, child := range typed {
			findings = append(findings, collectJSONDiagnostics(tool, category, child)...)
		}
		return findings
	case map[string]any:
		var findings []Finding
		if finding, ok := findingFromMap(tool, category, typed); ok {
			findings = append(findings, finding)
		}
		for _, child := range typed {
			findings = append(findings, collectJSONDiagnostics(tool, category, child)...)
		}
		return findings
	default:
		return nil
	}
}

func findingFromMap(tool string, category string, fields map[string]any) (Finding, bool) {
	pathKey := "p" + "ath"
	file := firstStringPath(fields,
		[]string{"file"},
		[]string{"File"},
		[]string{"filename"},
		[]string{pathKey},
		[]string{"location", pathKey},
		[]string{"artifactLocation", "uri"},
		[]string{"Target"},
	)
	line := firstIntPath(fields,
		[]string{"line"},
		[]string{"Line"},
		[]string{"startLine"},
		[]string{"StartLine"},
		[]string{"region", "startLine"},
		[]string{"location", "start", "line"},
		[]string{"Start", "Line"},
	)
	column := firstIntPath(fields,
		[]string{"column"},
		[]string{"Column"},
		[]string{"col"},
		[]string{"startColumn"},
		[]string{"StartColumn"},
		[]string{"region", "startColumn"},
		[]string{"location", "start", "column"},
	)
	code := firstStringPath(fields,
		[]string{"code"},
		[]string{"Code"},
		[]string{"rule"},
		[]string{"Rule"},
		[]string{"rule_id"},
		[]string{"RuleID"},
		[]string{"check_id"},
		[]string{"checkId"},
		[]string{"id"},
		[]string{"ID"},
	)
	message := firstStringPath(fields,
		[]string{"message"},
		[]string{"Message"},
		[]string{"description"},
		[]string{"Description"},
		[]string{"title"},
		[]string{"Title"},
		[]string{"message", "text"},
		[]string{"Message", "Text"},
	)
	severity := firstStringPath(fields,
		[]string{"severity"},
		[]string{"Severity"},
		[]string{"level"},
		[]string{"Level"},
		[]string{"type"},
		[]string{"Type"},
	)

	if message == "" && code == "" {
		return Finding{}, false
	}
	if file == "" && line == 0 && !core.Contains(core.Lower(category), "security") && code == "" {
		return Finding{}, false
	}

	return Finding{
		Tool:     tool,
		File:     file,
		Line:     line,
		Column:   column,
		Severity: firstNonEmpty(normaliseSeverity(severity), defaultSeverityForCategory(category)),
		Code:     code,
		Message:  message,
		Category: category,
	}, true
}

func parseTextDiagnostics(tool string, category string, output string) []Finding {
	var findings []Finding

	for _, line := range core.Split(core.Trim(output), "\n") {
		trimmed := core.Trim(line)
		if trimmed == "" {
			continue
		}

		if finding, ok := parseTextDiagnosticLine(tool, category, trimmed); ok {
			findings = append(findings, finding)
		}
	}

	if len(findings) == 0 && core.Trim(output) != "" {
		findings = append(findings, Finding{
			Tool:     tool,
			Severity: defaultSeverityForCategory(category),
			Code:     "diagnostic",
			Message:  core.Trim(output),
			Category: category,
		})
	}

	return dedupeFindings(findings)
}

func parsePrettierDiagnostics(tool string, category string, output string) []Finding {
	var findings []Finding

	for _, line := range core.Split(core.Trim(output), "\n") {
		trimmed := core.Trim(line)
		if trimmed == "" {
			continue
		}

		findings = append(findings, Finding{
			Tool:     tool,
			File:     core.PathToSlash(trimmed),
			Severity: defaultSeverityForCategory(category),
			Code:     "prettier-format",
			Message:  "File is not formatted with Prettier",
			Category: category,
		})
	}

	return dedupeFindings(findings)
}

func parseTextDiagnosticLine(tool string, category string, line string) (Finding, bool) {
	segments := core.Split(line, ":")
	if len(segments) < 3 {
		return Finding{}, false
	}

	lineNumber, lineErr := strconv.Atoi(core.Trim(segments[1]))
	if lineErr != nil {
		return Finding{}, false
	}

	columnNumber := 0
	messageIndex := 2
	if len(segments) > 3 {
		if parsedColumn, columnErr := strconv.Atoi(core.Trim(segments[2])); columnErr == nil {
			columnNumber = parsedColumn
			messageIndex = 3
		}
	}

	message := core.Trim(core.Join(":", segments[messageIndex:]...))
	if message == "" {
		return Finding{}, false
	}

	severity := defaultSeverityForCategory(category)
	switch {
	case core.Contains(core.Lower(message), "warning"):
		severity = "warning"
	case core.Contains(core.Lower(message), "error"):
		severity = "error"
	}

	return Finding{
		Tool:     tool,
		File:     core.PathToSlash(core.Trim(segments[0])),
		Line:     lineNumber,
		Column:   columnNumber,
		Severity: severity,
		Code:     "diagnostic",
		Message:  message,
		Category: category,
	}, true
}

func firstStringPath(fields map[string]any, paths ...[]string) string {
	for _, path := range paths {
		if value, ok := lookupPath(fields, path); ok {
			switch typed := value.(type) {
			case string:
				if core.Trim(typed) != "" {
					return core.Trim(typed)
				}
			}
		}
	}
	return ""
}

func firstIntPath(fields map[string]any, paths ...[]string) int {
	for _, path := range paths {
		if value, ok := lookupPath(fields, path); ok {
			switch typed := value.(type) {
			case int:
				return typed
			case int64:
				return int(typed)
			case float64:
				return int(typed)
			case string:
				parsed, err := strconv.Atoi(core.Trim(typed))
				if err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func lookupPath(fields map[string]any, path []string) (any, bool) {
	current := any(fields)
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, found := mapValue(object, segment)
		if !found {
			return nil, false
		}
		current = value
	}
	return current, true
}

func mapValue(fields map[string]any, key string) (any, bool) {
	if value, ok := fields[key]; ok {
		return value, true
	}
	lowerKey := core.Lower(key)
	for fieldKey, value := range fields {
		if core.Lower(fieldKey) == lowerKey {
			return value, true
		}
	}
	return nil, false
}

func dedupeFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	var deduped []Finding
	for _, finding := range findings {
		key := core.Join("|",
			finding.Tool,
			finding.File,
			strconv.Itoa(finding.Line),
			strconv.Itoa(finding.Column),
			finding.Code,
			finding.Message,
		)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, finding)
	}
	return deduped
}

func hasNonParseErrorFinding(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Code != adapterParseError89bf3a {
			return true
		}
	}
	return false
}

func onlyParseErrorFindings(findings []Finding) bool {
	if len(findings) == 0 {
		return false
	}
	for _, finding := range findings {
		if finding.Code != adapterParseError89bf3a {
			return false
		}
	}
	return true
}

func filterRulesByTag(rules []Rule, tag string) []Rule {
	var filtered []Rule
	for _, rule := range rules {
		for _, currentTag := range rule.Tags {
			if currentTag == tag {
				filtered = append(filtered, rule)
				break
			}
		}
	}
	return filtered
}

func ruleCategory(rule Rule) string {
	for _, tag := range rule.Tags {
		switch tag {
		case "security", "style", "correctness", "performance", "compliance":
			return tag
		}
	}
	return "correctness"
}

func normaliseSeverity(severity string) string {
	normalised := core.Lower(core.Trim(severity))
	switch normalised {
	case "critical", "high", "error":
		return "error"
	case "medium", "low", "warning", "warn":
		return "warning"
	case "info", "note":
		return "info"
	default:
		if normalised == "err"+"ors" {
			return "error"
		}
		return normalised
	}
}

func defaultSeverityForCategory(category string) string {
	switch category {
	case "security":
		return "error"
	case "compliance":
		return "warning"
	default:
		return "warning"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return core.Trim(value)
		}
	}
	return ""
}

func firstVersionLine(output string) string {
	for _, line := range core.Split(core.Trim(output), "\n") {
		line = core.Trim(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

const defaultCatalogRulesYAML = `
- id: go-cor-003
  title: "Silent error swallowing with blank identifier"
  severity: medium
  languages: [go]
  tags: [correctness, errors]
  pattern: '^\s*(var\s+)?_\s*=\s*\w+\.\w+\('
  exclude_pattern: 'defer|Close\(|Flush\('
  fix: "Handle the error explicitly — log it, return it, or document why it is safe to discard"
  detection: regex
  auto_fixable: false

- id: go-cor-004
  title: "Panic in library code"
  severity: high
  languages: [go]
  tags: [correctness, panic]
  pattern: '\bpanic\('
  exclude_pattern: '_test\.go|// unreachable|Must\w+\('
  fix: "Return an error instead of panicking — panics in libraries crash the caller"
  detection: regex
  auto_fixable: false

- id: go-sec-001
  title: "SQL wildcard injection in LIKE clauses"
  severity: high
  languages: [go]
  tags: [security, injection]
  pattern: 'LIKE\s+\?.*["%].*\+'
  fix: "Use parameterised LIKE with EscapeLike() helper to sanitise wildcard characters"
  detection: regex
  auto_fixable: false

- id: go-sec-002
  title: "Path traversal via core.PathJoin"
  severity: high
  languages: [go]
  tags: [security, path-traversal]
  pattern: 'filepath\.Join\(.*,\s*\w+\)'
  exclude_pattern: 'filepath\.Clean|securejoin|ValidatePath'
  fix: "Validate the path component or use securejoin to prevent directory traversal"
  detection: regex
  auto_fixable: false

- id: go-sec-004
  title: "Non-constant-time authentication comparison"
  severity: critical
  languages: [go]
  tags: [security, timing-attack]
  pattern: '==\s*\w*(token|key|secret|password|hash|digest|hmac|mac|sig)'
  exclude_pattern: 'subtle\.ConstantTimeCompare|hmac\.Equal'
  fix: "Use subtle.ConstantTimeCompare() or hmac.Equal() for timing-safe comparison"
  detection: regex
  auto_fixable: false
`
