package lint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
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

type parseFunc func(tool string, category string, output string) []Finding
type argsBuilder func(projectPath string, files []string) []string

// CommandAdapter runs an external binary and parses its stdout/stderr.
type CommandAdapter struct {
	name                string
	binaries            []string
	languages           []string
	category            string
	entitlement         string
	requiresEntitlement bool
	fast                bool
	buildArgs           argsBuilder
	parseOutput         parseFunc
}

// CatalogAdapter wraps the embedded regex rule catalog as a built-in linter.
type CatalogAdapter struct{}

func defaultAdapters() []Adapter {
	return []Adapter{
		newCommandAdapter("golangci-lint", []string{"golangci-lint"}, []string{"go"}, "correctness", "", false, true, goPackageArgs("run", "--out-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("gosec", []string{"gosec"}, []string{"go"}, "security", "lint.security", true, false, goPackageArgs("-fmt", "json"), parseJSONDiagnostics),
		newCommandAdapter("govulncheck", []string{"govulncheck"}, []string{"go"}, "security", "", false, false, goPackageArgs("-json"), parseGovulncheckDiagnostics),
		newCommandAdapter("staticcheck", []string{"staticcheck"}, []string{"go"}, "correctness", "", false, true, goPackageArgs("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("revive", []string{"revive"}, []string{"go"}, "style", "", false, true, goPackageArgs("-formatter", "json"), parseJSONDiagnostics),
		newCommandAdapter("errcheck", []string{"errcheck"}, []string{"go"}, "correctness", "", false, true, goPackageArgs(), parseTextDiagnostics),
		newCommandAdapter("phpstan", []string{"phpstan"}, []string{"php"}, "correctness", "", false, true, pathArgs("analyse", "--error-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("psalm", []string{"psalm"}, []string{"php"}, "correctness", "", false, true, pathArgs("--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("phpcs", []string{"phpcs"}, []string{"php"}, "style", "", false, true, pathArgs("--report=json"), parseJSONDiagnostics),
		newCommandAdapter("phpmd", []string{"phpmd"}, []string{"php"}, "correctness", "", false, true, phpmdArgs(), parseJSONDiagnostics),
		newCommandAdapter("pint", []string{"pint"}, []string{"php"}, "style", "", false, true, pathArgs("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("biome", []string{"biome"}, []string{"js", "ts"}, "style", "", false, true, pathArgs("check", "--reporter", "json"), parseJSONDiagnostics),
		newCommandAdapter("oxlint", []string{"oxlint"}, []string{"js", "ts"}, "style", "", false, true, pathArgs("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("eslint", []string{"eslint"}, []string{"js"}, "style", "", false, true, pathArgs("--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("typescript", []string{"tsc", "typescript"}, []string{"ts"}, "correctness", "", false, true, pathArgs("--pretty", "false"), parseTextDiagnostics),
		newCommandAdapter("ruff", []string{"ruff"}, []string{"python"}, "style", "", false, true, pathArgs("check", "--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("mypy", []string{"mypy"}, []string{"python"}, "correctness", "", false, true, pathArgs("--output", "json"), parseJSONDiagnostics),
		newCommandAdapter("bandit", []string{"bandit"}, []string{"python"}, "security", "lint.security", true, false, recursivePathArgs("-f", "json", "-r"), parseJSONDiagnostics),
		newCommandAdapter("pylint", []string{"pylint"}, []string{"python"}, "style", "", false, true, pathArgs("--output-format", "json"), parseJSONDiagnostics),
		newCommandAdapter("shellcheck", []string{"shellcheck"}, []string{"shell"}, "correctness", "", false, true, fileArgs("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("hadolint", []string{"hadolint"}, []string{"dockerfile"}, "security", "", false, true, fileArgs("-f", "json"), parseJSONDiagnostics),
		newCommandAdapter("yamllint", []string{"yamllint"}, []string{"yaml"}, "style", "", false, true, pathArgs("-f", "parsable"), parseTextDiagnostics),
		newCommandAdapter("jsonlint", []string{"jsonlint"}, []string{"json"}, "style", "", false, true, fileArgs(), parseTextDiagnostics),
		newCommandAdapter("markdownlint", []string{"markdownlint", "markdownlint-cli"}, []string{"markdown"}, "style", "", false, true, pathArgs("--json"), parseJSONDiagnostics),
		newCommandAdapter("gitleaks", []string{"gitleaks"}, []string{"*"}, "security", "lint.security", true, false, recursivePathArgs("detect", "--no-git", "--report-format", "json", "--source"), parseJSONDiagnostics),
		newCommandAdapter("trivy", []string{"trivy"}, []string{"*"}, "security", "lint.security", true, false, pathArgs("fs", "--format", "json"), parseJSONDiagnostics),
		newCommandAdapter("semgrep", []string{"semgrep"}, []string{"*"}, "security", "lint.security", true, false, pathArgs("--json"), parseJSONDiagnostics),
		newCommandAdapter("syft", []string{"syft"}, []string{"*"}, "compliance", "lint.compliance", true, false, pathArgs("scan", "-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("grype", []string{"grype"}, []string{"*"}, "security", "lint.compliance", true, false, pathArgs("-o", "json"), parseJSONDiagnostics),
		newCommandAdapter("scancode", []string{"scancode-toolkit", "scancode"}, []string{"*"}, "compliance", "lint.compliance", true, false, pathArgs("--json"), parseJSONDiagnostics),
	}
}

func newCatalogAdapter() Adapter {
	return CatalogAdapter{}
}

func newCommandAdapter(name string, binaries []string, languages []string, category string, entitlement string, requiresEntitlement bool, fast bool, builder argsBuilder, parser parseFunc) Adapter {
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
		return result
	}

	runContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := adapter.buildArgs(input.Path, files)
	stdout, stderr, exitCode, runErr := runCommand(runContext, input.Path, binary, args)

	result.Tool.Version = ""
	result.Tool.Duration = time.Since(startedAt).Round(time.Millisecond).String()

	if errors.Is(runContext.Err(), context.DeadlineExceeded) {
		result.Tool.Status = "timeout"
		return result
	}

	output := strings.TrimSpace(stdout)
	if strings.TrimSpace(stderr) != "" {
		if output != "" {
			output += "\n" + strings.TrimSpace(stderr)
		} else {
			output = strings.TrimSpace(stderr)
		}
	}

	if adapter.parseOutput != nil && output != "" {
		result.Findings = adapter.parseOutput(adapter.name, adapter.category, output)
	}
	if len(result.Findings) == 0 && output != "" {
		result.Findings = parseTextDiagnostics(adapter.name, adapter.category, output)
	}
	if len(result.Findings) == 0 && runErr != nil {
		result.Findings = []Finding{{
			Tool:     adapter.name,
			Severity: defaultSeverityForCategory(adapter.category),
			Code:     "command-failed",
			Message:  strings.TrimSpace(firstNonEmpty(output, runErr.Error())),
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
	return len(languages) == 0 || slicesContains(languages, "go")
}

func (CatalogAdapter) Category() string { return "correctness" }

func (CatalogAdapter) Fast() bool { return true }

func (CatalogAdapter) Run(_ context.Context, input RunInput, files []string) AdapterResult {
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
		findings, _ = scanner.ScanDir(input.Path)
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

func goPackageArgs(prefix ...string) argsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, "./...")
	}
}

func pathArgs(prefix ...string) argsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func recursivePathArgs(prefix ...string) argsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func fileArgs(prefix ...string) argsBuilder {
	return func(_ string, files []string) []string {
		args := append([]string(nil), prefix...)
		if len(files) > 0 {
			return append(args, files...)
		}
		return append(args, ".")
	}
}

func phpmdArgs() argsBuilder {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
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
	decoder := json.NewDecoder(strings.NewReader(output))
	var findings []Finding

	for {
		var value any
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil
		}
		findings = append(findings, collectJSONDiagnostics(tool, category, value)...)
	}

	return dedupeFindings(findings)
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
			case json.Number:
				return typed.String()
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
			case json.Number:
				parsed, _ := typed.Int64()
				return int(parsed)
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

func slicesContains(values []string, target string) bool {
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
