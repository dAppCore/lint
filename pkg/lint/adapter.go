package lint

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
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
		newCommandAdapter("gosec", []string{"gosec"}, []string{"go"}, "security", "lint.security", true, false, goProjectArguments("-fmt", "json"), parseJSONDiagnostics),
		newCommandAdapter("govulncheck", []string{"govulncheck"}, []string{"go"}, "security", "", false, false, goProjectArguments("-json"), parseGovulncheckDiagnostics),
		newCommandAdapter("staticcheck", []string{"staticcheck"}, []string{"go"}, "correctness", "", false, true, goProjectArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("revive", []string{"revive"}, []string{"go"}, "style", "", false, true, goProjectArguments("-formatter", "json"), parseJSONDiagnostics),
		newCommandAdapter("errcheck", []string{"errcheck"}, []string{"go"}, "correctness", "", false, true, goProjectArguments(), parseTextDiagnostics),
		newCommandAdapter("phpstan", []string{"phpstan"}, []string{"php"}, "correctness", "", false, true, projectPathArguments("analyse", "--error-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("psalm", []string{"psalm"}, []string{"php"}, "correctness", "", false, true, projectPathArguments("--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("phpcs", []string{"phpcs"}, []string{"php"}, "style", "", false, true, projectPathArguments("--report=json"), parseJSONDiagnostics),
		newCommandAdapter("phpmd", []string{"phpmd"}, []string{"php"}, "correctness", "", false, true, phpmdArguments(), parseJSONDiagnostics),
		newCommandAdapter("pint", []string{"pint"}, []string{"php"}, "style", "", false, true, projectPathArguments("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("biome", []string{"biome"}, []string{"js", "ts"}, "style", "", false, true, projectPathArguments("check", "--reporter", "json"), parseJSONDiagnostics),
		newCommandAdapter("oxlint", []string{"oxlint"}, []string{"js", "ts"}, "style", "", false, true, projectPathArguments("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("eslint", []string{"eslint"}, []string{"js"}, "style", "", false, true, projectPathArguments("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("typescript", []string{"tsc", "typescript"}, []string{"ts"}, "correctness", "", false, true, projectPathArguments("--pretty", "false"), parseTextDiagnostics),
		newCommandAdapter("ruff", []string{"ruff"}, []string{"python"}, "style", "", false, true, projectPathArguments("check", "--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("mypy", []string{"mypy"}, []string{"python"}, "correctness", "", false, true, projectPathArguments("--output", "json"), parseJSONDiagnostics),
		newCommandAdapter("bandit", []string{"bandit"}, []string{"python"}, "security", "lint.security", true, false, recursiveProjectPathArguments("-f", "json", "-r"), parseJSONDiagnostics),
		newCommandAdapter("pylint", []string{"pylint"}, []string{"python"}, "style", "", false, true, projectPathArguments("--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("shellcheck", []string{"shellcheck"}, []string{"shell"}, "correctness", "", false, true, filePathArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("hadolint", []string{"hadolint"}, []string{"dockerfile"}, "security", "", false, true, filePathArguments("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("yamllint", []string{"yamllint"}, []string{"yaml"}, "style", "", false, true, projectPathArguments("-f", "parsable"), parseTextDiagnostics),
		newCommandAdapter("jsonlint", []string{"jsonlint"}, []string{"json"}, "style", "", false, true, filePathArguments(), parseTextDiagnostics),
		newCommandAdapter("markdownlint", []string{"markdownlint", "markdownlint-cli"}, []string{"markdown"}, "style", "", false, true, projectPathArguments("--json"), parseJSONDiagnostics),
		newCommandAdapter("prettier", []string{"prettier"}, []string{"js"}, "style", "", false, true, projectPathArguments("--list-different"), parsePrettierDiagnostics),
		newCommandAdapter("gitleaks", []string{"gitleaks"}, []string{"*"}, "security", "lint.security", true, false, recursiveProjectPathArguments("detect", "--no-git", "--report-format", "json", "--source"), parseJSONDiagnostics),
		newCommandAdapter("trivy", []string{"trivy"}, []string{"*"}, "security", "lint.security", true, false, projectPathArguments("fs", "--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("semgrep", []string{"semgrep"}, []string{"*"}, "security", "lint.security", true, false, projectPathArguments("--json"), parseJSONDiagnostics),
		newCommandAdapter("syft", []string{"syft"}, []string{"*"}, "compliance", "lint.compliance", true, false, projectPathArguments("scan", "-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("grype", []string{"grype"}, []string{"*"}, "security", "lint.compliance", true, false, projectPathArguments("-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("scancode", []string{"scancode-toolkit", "scancode"}, []string{"*"}, "compliance", "lint.compliance", true, false, projectPathArguments("--json"), parseJSONDiagnostics),
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
	if len(adapter.languages) == 0 || len(languages) == 0 {
		return true
	}
	if len(adapter.languages) == 1 && adapter.languages[0] == "*" {
		return true
	}
	for _, language := range languages {
		if strings.EqualFold(language, adapter.category) {
			return true
		}
		for _, supported := range adapter.languages {
			if supported == language {
				return true
			}
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
			Message:  fmt.Sprintf("%s is not installed", missingName),
			Category: adapter.category,
		}}
		result.Tool.Findings = len(result.Findings)
		return result
	}

	result.Tool.Version = probeCommandVersion(binary, input.Path)

	runContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := adapter.buildArgs(input.Path, files)
	stdout, stderr, exitCode, runErr := runCommand(runContext, input.Path, binary, args)
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()

	if core.Is(runContext.Err(), context.DeadlineExceeded) {
		result.Tool.Status = "timeout"
		return result
	}

	if err := runContext.Err(); err != nil {
		if core.Is(err, context.DeadlineExceeded) {
			result.Tool.Status = "timeout"
		} else {
			result.Tool.Status = "canceled"
		}
		return result
	}

	if adapter.parseOutput != nil {
		if stdout != "" {
			result.Findings = append(result.Findings, adapter.parseOutput(adapter.name, adapter.category, stdout)...)
		}
		if len(result.Findings) == 0 && stderr != "" {
			result.Findings = append(result.Findings, adapter.parseOutput(adapter.name, adapter.category, stderr)...)
		}
	}
	if len(result.Findings) == 0 && (stdout != "" || stderr != "") {
		output := stdout
		if output != "" && stderr != "" {
			output += "\n" + stderr
		} else if output == "" {
			output = stderr
		}
		result.Findings = parseTextDiagnostics(adapter.name, adapter.category, output)
	}
	if len(result.Findings) == 0 && runErr != nil {
		result.Findings = []Finding{{
			Tool:     adapter.name,
			Severity: defaultSeverityForCategory(adapter.category),
			Code:     "command-failed",
			Message:  strings.TrimSpace(firstNonEmpty(stdout, stderr, runErr.Error())),
			Category: adapter.category,
		}}
	}

	for index := range result.Findings {
		if result.Findings[index].Tool == "" {
			result.Findings[index].Tool = adapter.name
		}
		if result.Findings[index].Category == "" {
			result.Findings[index].Category = adapter.category
		}
		if result.Findings[index].Severity == "" {
			result.Findings[index].Severity = defaultSeverityForCategory(adapter.category)
		} else {
			result.Findings[index].Severity = normaliseSeverity(result.Findings[index].Severity)
		}
	}

	result.Tool.Findings = len(result.Findings)
	switch {
	case runErr != nil || exitCode != 0 || len(result.Findings) > 0:
		result.Tool.Status = "failed"
	default:
		result.Tool.Status = "passed"
	}

	return result
}

func probeCommandVersion(binary string, workingDir string) string {
	for _, args := range [][]string{{"--version"}, {"-version"}, {"version"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		stdout, stderr, exitCode, err := runCommand(ctx, workingDir, binary, args)
		cancel()

		if err != nil && exitCode != 0 {
			continue
		}

		version := firstNonEmpty(stdout, stderr)
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
		path, err := exec.LookPath(binary)
		if err == nil {
			return path, true
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

	catalog, err := loadBuiltinCatalog()
	if err != nil {
		result.Tool.Status = "failed"
		result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
		result.Findings = []Finding{{
			Tool:     "catalog",
			Severity: "error",
			Code:     "catalog-load",
			Message:  err.Error(),
			Category: "correctness",
		}}
		result.Tool.Findings = len(result.Findings)
		return result
	}

	rules := catalog.Rules
	if input.Category != "" {
		rules = filterRulesByTag(rules, input.Category)
	}

	scanner, err := NewScanner(rules)
	if err != nil {
		result.Tool.Status = "failed"
		result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
		result.Findings = []Finding{{
			Tool:     "catalog",
			Severity: "error",
			Code:     "catalog-scan",
			Message:  err.Error(),
			Category: "correctness",
		}}
		result.Tool.Findings = len(result.Findings)
		return result
	}

	var findings []Finding
	if len(files) > 0 {
		for _, file := range files {
			if err := ctx.Err(); err != nil {
				break
			}
			scanPath := file
			if !filepath.IsAbs(scanPath) {
				scanPath = filepath.Join(input.Path, file)
			}
			fileFindings, scanErr := scanner.ScanFile(scanPath)
			if scanErr != nil {
				continue
			}
			findings = append(findings, fileFindings...)
		}
	} else {
		if ctx.Err() != nil {
			result.Tool.Status = "canceled"
			result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
			return result
		}
		findings, _ = scanner.ScanDir(input.Path)
	}

	if err := ctx.Err(); err != nil {
		result.Tool.Status = "canceled"
		result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
		result.Tool.Findings = len(findings)
		result.Findings = findings
		return result
	}

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

	result.Findings = findings
	result.Tool.Findings = len(findings)
	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()
	if len(findings) > 0 {
		result.Tool.Status = "failed"
	} else {
		result.Tool.Status = "passed"
	}

	return result
}

func loadBuiltinCatalog() (*Catalog, error) {
	rules, err := ParseRules([]byte(defaultCatalogRulesYAML))
	if err != nil {
		return nil, coreerr.E("loadBuiltinCatalog", "parse embedded fallback rules", err)
	}
	return &Catalog{Rules: rules}, nil
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
			target = strings.Join(files, ",")
		}
		return []string{target, "json", "cleancode,codesize,controversial,design,naming,unusedcode"}
	}
}

func runCommand(ctx context.Context, workingDir string, binary string, args []string) (string, string, int, error) {
	command := exec.CommandContext(ctx, binary, args...)
	if workingDir != "" {
		command.Dir = workingDir
	}

	stdout := core.NewBuilder()
	stderr := core.NewBuilder()
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0, nil
	}

	var exitErr *exec.ExitError
	if core.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode(), err
	}

	return stdout.String(), stderr.String(), -1, err
}

func parseGovulncheckDiagnostics(tool string, category string, output string) []Finding {
	result, err := ParseVulnCheckJSON(output, "")
	if err != nil || result == nil {
		return nil
	}

	var findings []Finding
	for _, vuln := range result.Findings {
		message := strings.TrimSpace(firstNonEmpty(vuln.Description, vuln.Package))
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
	trimmed := strings.TrimSpace(output)
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
		if inString {
			switch {
			case escaped:
				escaped = false
			case current == '\\':
				escaped = true
			case current == '"':
				inString = false
			}
			continue
		}

		switch current {
		case '"':
			if depth > 0 {
				inString = true
			}
		case '{', '[':
			if depth == 0 {
				start = index
			}
			depth++
		case '}', ']':
			if depth == 0 {
				return segments, strings.TrimSpace(output[index:])
			}
			depth--
			if depth == 0 && start >= 0 {
				segments = append(segments, output[start:index+1])
				start = -1
			}
		default:
			if depth == 0 && len(segments) == 0 && !isJSONWhitespace(current) {
				return nil, output
			}
			if depth == 0 && len(segments) > 0 && !isJSONWhitespace(current) {
				return segments, strings.TrimSpace(output[index:])
			}
		}
	}

	if depth != 0 && start >= 0 {
		return segments, strings.TrimSpace(output[start:])
	}

	return segments, ""
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
		Code:     "parse-error",
		Message:  fmt.Sprintf("failed to parse JSON output: %v", err),
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
	file := firstStringPath(fields,
		[]string{"file"},
		[]string{"File"},
		[]string{"filename"},
		[]string{"path"},
		[]string{"location", "path"},
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
	if file == "" && line == 0 && !strings.Contains(strings.ToLower(category), "security") && code == "" {
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

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if finding, ok := parseTextDiagnosticLine(tool, category, trimmed); ok {
			findings = append(findings, finding)
		}
	}

	if len(findings) == 0 && strings.TrimSpace(output) != "" {
		findings = append(findings, Finding{
			Tool:     tool,
			Severity: defaultSeverityForCategory(category),
			Code:     "diagnostic",
			Message:  strings.TrimSpace(output),
			Category: category,
		})
	}

	return dedupeFindings(findings)
}

func parsePrettierDiagnostics(tool string, category string, output string) []Finding {
	var findings []Finding

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		findings = append(findings, Finding{
			Tool:     tool,
			File:     filepath.ToSlash(trimmed),
			Severity: defaultSeverityForCategory(category),
			Code:     "prettier-format",
			Message:  "File is not formatted with Prettier",
			Category: category,
		})
	}

	return dedupeFindings(findings)
}

func parseTextDiagnosticLine(tool string, category string, line string) (Finding, bool) {
	segments := strings.Split(line, ":")
	if len(segments) < 3 {
		return Finding{}, false
	}

	lineNumber, lineErr := strconv.Atoi(strings.TrimSpace(segments[1]))
	if lineErr != nil {
		return Finding{}, false
	}

	columnNumber := 0
	messageIndex := 2
	if len(segments) > 3 {
		if parsedColumn, columnErr := strconv.Atoi(strings.TrimSpace(segments[2])); columnErr == nil {
			columnNumber = parsedColumn
			messageIndex = 3
		}
	}

	message := strings.TrimSpace(strings.Join(segments[messageIndex:], ":"))
	if message == "" {
		return Finding{}, false
	}

	severity := defaultSeverityForCategory(category)
	switch {
	case strings.Contains(strings.ToLower(message), "warning"):
		severity = "warning"
	case strings.Contains(strings.ToLower(message), "error"):
		severity = "error"
	}

	return Finding{
		Tool:     tool,
		File:     filepath.ToSlash(strings.TrimSpace(segments[0])),
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
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
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
				parsed, err := strconv.Atoi(strings.TrimSpace(typed))
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
	lowerKey := strings.ToLower(key)
	for fieldKey, value := range fields {
		if strings.ToLower(fieldKey) == lowerKey {
			return value, true
		}
	}
	return nil, false
}

func dedupeFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	var deduped []Finding
	for _, finding := range findings {
		key := strings.Join([]string{
			finding.Tool,
			finding.File,
			strconv.Itoa(finding.Line),
			strconv.Itoa(finding.Column),
			finding.Code,
			finding.Message,
		}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, finding)
	}
	return deduped
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
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high", "error", "errors":
		return "error"
	case "medium", "low", "warning", "warn":
		return "warning"
	case "info", "note":
		return "info"
	default:
		return strings.ToLower(strings.TrimSpace(severity))
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
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstVersionLine(output string) string {
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
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
  pattern: '^\s*_\s*=\s*\w+\.\w+\('
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
  title: "Path traversal via filepath.Join"
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
