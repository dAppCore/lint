// cmd_php.go adds PHP quality assurance subcommands to the qa parent command.
//
// Commands:
//   - fmt: Format PHP code with Laravel Pint
//   - stan: Run PHPStan static analysis
//   - psalm: Run Psalm static analysis
//   - audit: Check dependency security
//   - security: Run security checks
//   - rector: Automated code refactoring
//   - infection: Mutation testing
//   - test: Run PHPUnit/Pest tests

package qa

import (
	"context"
	"sort"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/lint/pkg/detect"
	"dappco.re/go/lint/pkg/php"
)

const (
	cmdPhpNotAPhpProjectNoComposerJsonFound469a5f = "not a PHP project (no composer.json found)"
	cmdPhpOutputResultsAsJson30f08f               = "Output results as JSON"
	cmdPhpOutputResultsInSarifFormat29b025        = "Output results in SARIF format"
	cmdPhpSSc5e701                                = "%s %s\n"
	cmdPhpSb4dc47                                 = "  %s\n"
	cmdPhpSecurityChecksFailed47bd21              = "security checks failed"
)

// Severity styles for security output.
var (
	headerStyle   = cli.HeaderStyle
	criticalStyle = cli.NewStyle().Bold().Foreground(cli.ColourRed500)
	highStyle     = cli.NewStyle().Bold().Foreground(cli.ColourOrange500)
	mediumStyle   = cli.NewStyle().Foreground(cli.ColourAmber500)
	lowStyle      = cli.NewStyle().Foreground(cli.ColourGray500)
)

// addPHPCommands registers all PHP QA subcommands.
func addPHPCommands(c *core.Core) core.Result {
	for _, register := range []func(*core.Core) core.Result{
		addPHPFmtCommand,
		addPHPStanCommand,
		addPHPPsalmCommand,
		addPHPAuditCommand,
		addPHPSecurityCommand,
		addPHPRectorCommand,
		addPHPInfectionCommand,
		addPHPTestCommand,
	} {
		if r := register(c); !r.OK {
			return r
		}
	}
	return core.Ok(nil)
}

// PHP fmt command flags.
var (
	phpFmtFix  bool
	phpFmtDiff bool
	phpFmtJSON bool
)

func addPHPFmtCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/fmt", "Format PHP code with Laravel Pint", runPHPFmt)
}

func runPHPFmt() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	output := core.NewBuilder()
	if !isMachineReadableOutput(phpFmtJSON) {
		cli.Print(cmdPhpSSc5e701, headerStyle.Render("PHP Format"), dimStyle.Render("(Pint)"))
		cli.Blank()
	}
	result := php.Format(context.Background(), php.FormatOptions{
		Dir:    dir,
		Fix:    phpFmtFix,
		Diff:   phpFmtDiff,
		JSON:   phpFmtJSON,
		Output: output,
	})
	printPHPToolOutput(output)
	return result
}

// PHP stan command flags.
var (
	phpStanLevel  int
	phpStanMemory string
	phpStanJSON   bool
	phpStanSARIF  bool
)

func addPHPStanCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/stan", "Run PHPStan static analysis", runPHPStan)
}

func runPHPStan() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	analyser, found := php.DetectAnalyser(dir)
	if !found {
		return core.Fail(cli.Err("no static analyser found (install PHPStan: composer require phpstan/phpstan --dev)"))
	}
	if !isMachineReadableOutput(phpStanJSON, phpStanSARIF) {
		cli.Print(cmdPhpSSc5e701, headerStyle.Render("PHP Static Analysis"), dimStyle.Render(core.Sprintf("(%s)", analyser)))
		cli.Blank()
	}
	output := core.NewBuilder()
	if r := php.Analyse(context.Background(), php.AnalyseOptions{
		Dir:    dir,
		Level:  phpStanLevel,
		Memory: phpStanMemory,
		JSON:   phpStanJSON,
		SARIF:  phpStanSARIF,
		Output: output,
	}); !r.OK {
		printPHPToolOutput(output)
		return core.Fail(cli.Err("static analysis found issues"))
	}
	printPHPToolOutput(output)
	if !isMachineReadableOutput(phpStanJSON, phpStanSARIF) {
		cli.Blank()
		cli.Print("%s\n", successStyle.Render("Static analysis passed"))
	}
	return core.Ok(nil)
}

// PHP psalm command flags.
var (
	phpPsalmLevel    int
	phpPsalmFix      bool
	phpPsalmBaseline bool
	phpPsalmShowInfo bool
	phpPsalmJSON     bool
	phpPsalmSARIF    bool
)

func addPHPPsalmCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/psalm", "Run Psalm static analysis", runPHPPsalm)
}

func runPHPPsalm() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	if _, found := php.DetectPsalm(dir); !found {
		return core.Fail(cli.Err("Psalm not found (install: composer require vimeo/psalm --dev)"))
	}
	if !isMachineReadableOutput(phpPsalmJSON, phpPsalmSARIF) {
		cli.Print("%s\n", headerStyle.Render("PHP Psalm Analysis"))
		cli.Blank()
	}
	output := core.NewBuilder()
	if r := php.RunPsalm(context.Background(), php.PsalmOptions{
		Dir:      dir,
		Level:    phpPsalmLevel,
		Fix:      phpPsalmFix,
		Baseline: phpPsalmBaseline,
		ShowInfo: phpPsalmShowInfo,
		JSON:     phpPsalmJSON,
		SARIF:    phpPsalmSARIF,
		Output:   output,
	}); !r.OK {
		printPHPToolOutput(output)
		return core.Fail(cli.Err("Psalm found issues"))
	}
	printPHPToolOutput(output)
	if !isMachineReadableOutput(phpPsalmJSON, phpPsalmSARIF) {
		cli.Blank()
		cli.Print("%s\n", successStyle.Render("Psalm analysis passed"))
	}
	return core.Ok(nil)
}

// PHP audit command flags.
var (
	phpAuditJSON bool
	phpAuditFix  bool
)

func addPHPAuditCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/audit", "Audit PHP and npm dependencies for vulnerabilities", runPHPAudit)
}

func runPHPAudit() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	if !isMachineReadableOutput(phpAuditJSON) {
		cli.Print("%s\n", headerStyle.Render("Dependency Audit"))
		cli.Blank()
	}

	results := php.RunAudit(context.Background(), php.AuditOptions{
		Dir:  cwd.Value.(string),
		JSON: phpAuditJSON,
		Fix:  phpAuditFix,
	})
	if !results.OK {
		return results
	}
	if phpAuditJSON {
		return printPHPAuditJSON(results.Value.([]php.AuditResult))
	}
	return printPHPAuditText(results.Value.([]php.AuditResult))
}

func printPHPAuditJSON(results []php.AuditResult) core.Result {
	payload := mapAuditResultsForJSON(results)
	data := core.JSONMarshalIndent(payload, "", "  ")
	if !data.OK {
		return data
	}
	cli.Print("%s\n", string(data.Value.([]byte)))
	if payload.HasVulnerabilities {
		return core.Fail(cli.Err("vulnerabilities found in dependencies"))
	}
	return core.Ok(nil)
}

func printPHPAuditText(results []php.AuditResult) core.Result {
	hasVulns := false
	for _, result := range results {
		if printPHPAuditResult(result) {
			hasVulns = true
		}
	}
	if hasVulns {
		return core.Fail(cli.Err("vulnerabilities found in dependencies"))
	}
	return core.Ok(nil)
}

func printPHPAuditResult(result php.AuditResult) bool {
	if result.Error != nil {
		cli.Print("%s %s: %s\n", warningStyle.Render("!"), result.Tool, result.Error)
		return false
	}
	if result.Vulnerabilities == 0 {
		cli.Print("%s %s: no vulnerabilities found\n",
			successStyle.Render(cli.Glyph(":check:")),
			result.Tool)
		return false
	}
	cli.Print("%s %s: %d vulnerabilities found\n",
		errorStyle.Render(cli.Glyph(":cross:")),
		result.Tool,
		result.Vulnerabilities)
	for _, adv := range result.Advisories {
		cli.Print("  %s %s: %s\n",
			dimStyle.Render("->"),
			adv.Package,
			adv.Title)
	}
	return true
}

// PHP security command flags.
var (
	phpSecuritySeverity string
	phpSecurityJSON     bool
	phpSecuritySARIF    bool
	phpSecurityURL      string
)

func addPHPSecurityCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/security", "Run security checks on the PHP project", runPHPSecurity)
}

func runPHPSecurity() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	if !isMachineReadableOutput(phpSecurityJSON, phpSecuritySARIF) {
		cli.Print("%s\n", headerStyle.Render("Security Checks"))
		cli.Blank()
	}

	securityResult := php.RunSecurityChecks(context.Background(), php.SecurityOptions{
		Dir:      cwd.Value.(string),
		Severity: phpSecuritySeverity,
		JSON:     phpSecurityJSON,
		SARIF:    phpSecuritySARIF,
		URL:      phpSecurityURL,
	})
	if !securityResult.OK {
		return securityResult
	}
	result := securityResult.Value.(*php.SecurityResult)

	result.Checks = sortSecurityChecks(result.Checks)
	if phpSecuritySARIF {
		return printPHPSecurityJSON(mapSecurityResultForSARIF(result), result.Summary)
	}
	if phpSecurityJSON {
		return printPHPSecurityJSON(result, result.Summary)
	}
	printPHPSecurityText(result)
	return failOnBlockingSecurity(result.Summary)
}

func requirePHPProjectDir() core.Result {
	cwd := core.Getwd()
	if !cwd.OK {
		return cwd
	}
	if !detect.IsPHPProject(cwd.Value.(string)) {
		return core.Fail(cli.Err(cmdPhpNotAPhpProjectNoComposerJsonFound469a5f))
	}
	return cwd
}

func printPHPSecurityJSON(payload any, summary php.SecuritySummary) core.Result {
	data := core.JSONMarshalIndent(payload, "", "  ")
	if !data.OK {
		return data
	}
	cli.Print("%s\n", string(data.Value.([]byte)))
	return failOnBlockingSecurity(summary)
}

func printPHPSecurityText(result *php.SecurityResult) {
	for _, check := range result.Checks {
		printPHPSecurityCheck(check)
	}
	cli.Blank()
	printPHPSecuritySummary(result.Summary)
}

func printPHPSecurityCheck(check php.SecurityCheck) {
	if check.Passed {
		cli.Print(cmdPhpSSc5e701,
			successStyle.Render(cli.Glyph(":check:")),
			check.Name)
		return
	}
	style := getSeverityStyle(check.Severity)
	cli.Print("%s %s %s\n",
		errorStyle.Render(cli.Glyph(":cross:")),
		check.Name,
		style.Render(core.Sprintf("[%s]", check.Severity)))
	if check.Message != "" {
		cli.Print("  %s %s\n", dimStyle.Render("->"), check.Message)
	}
	if check.Fix != "" {
		cli.Print("  %s Fix: %s\n", dimStyle.Render("->"), check.Fix)
	}
}

func printPHPSecuritySummary(summary php.SecuritySummary) {
	cli.Print("%s: %d/%d checks passed\n",
		headerStyle.Render("Summary"),
		summary.Passed, summary.Total)
	if summary.Critical > 0 {
		cli.Print(cmdPhpSb4dc47, criticalStyle.Render(core.Sprintf("%d critical", summary.Critical)))
	}
	if summary.High > 0 {
		cli.Print(cmdPhpSb4dc47, highStyle.Render(core.Sprintf("%d high", summary.High)))
	}
	if summary.Medium > 0 {
		cli.Print(cmdPhpSb4dc47, mediumStyle.Render(core.Sprintf("%d medium", summary.Medium)))
	}
	if summary.Low > 0 {
		cli.Print(cmdPhpSb4dc47, lowStyle.Render(core.Sprintf("%d low", summary.Low)))
	}
}

func failOnBlockingSecurity(summary php.SecuritySummary) core.Result {
	if summary.Critical > 0 || summary.High > 0 {
		return core.Fail(cli.Err(cmdPhpSecurityChecksFailed47bd21))
	}
	return core.Ok(nil)
}

type auditJSONOutput struct {
	Results            []auditResultJSON `json:"results"`
	HasVulnerabilities bool              `json:"has_vulnerabilities"`
	Vulnerabilities    int               `json:"vulnerabilities"`
}

type auditResultJSON struct {
	Tool            string              `json:"tool"`
	Vulnerabilities int                 `json:"vulnerabilities"`
	Advisories      []auditAdvisoryJSON `json:"advisories"`
	Error           string              `json:"error,omitempty"`
}

type auditAdvisoryJSON struct {
	Package     string   `json:"package"`
	Severity    string   `json:"severity,omitempty"`
	Title       string   `json:"title,omitempty"`
	URL         string   `json:"url,omitempty"`
	Identifiers []string `json:"identifiers,omitempty"`
}

func mapAuditResultsForJSON(results []php.AuditResult) auditJSONOutput {
	output := auditJSONOutput{
		Results: make([]auditResultJSON, 0, len(results)),
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Tool < results[j].Tool
	})

	for _, result := range results {
		entry := auditResultJSON{
			Tool:            result.Tool,
			Vulnerabilities: result.Vulnerabilities,
		}
		if result.Error != nil {
			entry.Error = result.Error.Error()
		}
		entry.Advisories = make([]auditAdvisoryJSON, 0, len(result.Advisories))
		for _, advisory := range result.Advisories {
			entry.Advisories = append(entry.Advisories, auditAdvisoryJSON{
				Package:     advisory.Package,
				Severity:    advisory.Severity,
				Title:       advisory.Title,
				URL:         advisory.URL,
				Identifiers: append([]string(nil), advisory.Identifiers...),
			})
		}
		sort.Slice(entry.Advisories, func(i, j int) bool {
			if entry.Advisories[i].Package == entry.Advisories[j].Package {
				return entry.Advisories[i].Title < entry.Advisories[j].Title
			}
			return entry.Advisories[i].Package < entry.Advisories[j].Package
		})
		output.Results = append(output.Results, entry)
		output.Vulnerabilities += entry.Vulnerabilities
	}

	output.HasVulnerabilities = output.Vulnerabilities > 0
	return output
}

func sortSecurityChecks(checks []php.SecurityCheck) []php.SecurityCheck {
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].ID < checks[j].ID
	})
	return checks
}

// PHP rector command flags.
var (
	phpRectorFix        bool
	phpRectorDiff       bool
	phpRectorClearCache bool
)

func addPHPRectorCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/rector", "Run Rector for automated PHP code refactoring", runPHPRector)
}

func runPHPRector() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	if !php.DetectRector(dir) {
		return core.Fail(cli.Err("Rector not found (install: composer require rector/rector --dev)"))
	}
	mode := "dry-run"
	if phpRectorFix {
		mode = "apply"
	}
	cli.Print(cmdPhpSSc5e701, headerStyle.Render("Rector Refactoring"), dimStyle.Render(core.Sprintf("(%s)", mode)))
	cli.Blank()
	output := core.NewBuilder()
	if r := php.RunRector(context.Background(), php.RectorOptions{
		Dir:        dir,
		Fix:        phpRectorFix,
		Diff:       phpRectorDiff,
		ClearCache: phpRectorClearCache,
		Output:     output,
	}); !r.OK {
		printPHPToolOutput(output)
		return core.Fail(cli.Err("Rector found refactoring suggestions"))
	}
	printPHPToolOutput(output)
	cli.Blank()
	cli.Print("%s\n", successStyle.Render("Rector check passed"))
	return core.Ok(nil)
}

// PHP infection command flags.
var (
	phpInfectionMinMSI        int
	phpInfectionMinCoveredMSI int
	phpInfectionThreads       int
	phpInfectionFilter        string
	phpInfectionOnlyCovered   bool
)

func addPHPInfectionCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/infection", "Run Infection mutation testing", runPHPInfection)
}

func runPHPInfection() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	if !php.DetectInfection(dir) {
		return core.Fail(cli.Err("Infection not found (install: composer require infection/infection --dev)"))
	}
	cli.Print("%s\n", headerStyle.Render("Mutation Testing"))
	cli.Blank()
	output := core.NewBuilder()
	if r := php.RunInfection(context.Background(), php.InfectionOptions{
		Dir:           dir,
		MinMSI:        phpInfectionMinMSI,
		MinCoveredMSI: phpInfectionMinCoveredMSI,
		Threads:       phpInfectionThreads,
		Filter:        phpInfectionFilter,
		OnlyCovered:   phpInfectionOnlyCovered,
		Output:        output,
	}); !r.OK {
		printPHPToolOutput(output)
		return core.Fail(cli.Err("mutation testing did not pass minimum thresholds"))
	}
	printPHPToolOutput(output)
	cli.Blank()
	cli.Print("%s\n", successStyle.Render("Mutation testing passed"))
	return core.Ok(nil)
}

// PHP test command flags.
var (
	phpTestParallel bool
	phpTestCoverage bool
	phpTestFilter   string
	phpTestGroup    string
	phpTestJUnit    bool
)

func addPHPTestCommand(c *core.Core) core.Result {
	return registerQACommand(c, "qa/test", "Run PHP tests with Pest or PHPUnit", runPHPTest)
}

func runPHPTest() core.Result {
	cwd := requirePHPProjectDir()
	if !cwd.OK {
		return cwd
	}
	dir := cwd.Value.(string)
	runner := php.DetectTestRunner(dir)
	if !isMachineReadableOutput(phpTestJUnit) {
		cli.Print(cmdPhpSSc5e701, headerStyle.Render("PHP Tests"), dimStyle.Render(core.Sprintf("(%s)", runner)))
		cli.Blank()
	}
	var groups []string
	if phpTestGroup != "" {
		groups = core.Split(phpTestGroup, ",")
	}
	output := core.NewBuilder()
	if r := php.RunTests(context.Background(), php.TestOptions{
		Dir:      dir,
		Parallel: phpTestParallel,
		Coverage: phpTestCoverage,
		Filter:   phpTestFilter,
		Groups:   groups,
		JUnit:    phpTestJUnit,
		Output:   output,
	}); !r.OK {
		printPHPToolOutput(output)
		return core.Fail(cli.Err("tests failed"))
	}
	printPHPToolOutput(output)
	if !isMachineReadableOutput(phpTestJUnit) {
		cli.Blank()
		cli.Print("%s\n", successStyle.Render("All tests passed"))
	}
	return core.Ok(nil)
}

func printPHPToolOutput(output interface{ String() string }) {
	text := output.String()
	if text != "" {
		cli.Print("%s", text)
	}
}

// getSeverityStyle returns a style for the given severity level.
func getSeverityStyle(severity string) *cli.AnsiStyle {
	switch core.Lower(severity) {
	case "critical":
		return criticalStyle
	case "high":
		return highStyle
	case "medium":
		return mediumStyle
	case "low":
		return lowStyle
	default:
		return dimStyle
	}
}

func isMachineReadableOutput(flags ...bool) bool {
	for _, flag := range flags {
		if flag {
			return true
		}
	}
	return false
}

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	ShortDescription sarifMessage `json:"shortDescription"`
	FullDescription  sarifMessage `json:"fullDescription"`
	Help             sarifMessage `json:"help,omitempty"`
	Properties       any          `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID     string       `json:"ruleId"`
	Level      string       `json:"level"`
	Message    sarifMessage `json:"message"`
	Properties any          `json:"properties,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

func mapSecurityResultForSARIF(result *php.SecurityResult) sarifLog {
	rules := make([]sarifRule, 0, len(result.Checks))
	sarifResults := make([]sarifResult, 0, len(result.Checks))

	for _, check := range result.Checks {
		rule := sarifRule{
			ID:               check.ID,
			Name:             check.Name,
			ShortDescription: sarifMessage{Text: check.Name},
			FullDescription:  sarifMessage{Text: check.Description},
		}
		if check.Fix != "" {
			rule.Help = sarifMessage{Text: check.Fix}
		}
		if check.CWE != "" {
			rule.Properties = map[string]any{"cwe": check.CWE}
		}
		rules = append(rules, rule)

		if check.Passed {
			continue
		}

		message := check.Message
		if message == "" {
			message = check.Description
		}

		properties := map[string]any{
			"severity": check.Severity,
		}
		if check.CWE != "" {
			properties["cwe"] = check.CWE
		}
		if check.Fix != "" {
			properties["fix"] = check.Fix
		}

		sarifResults = append(sarifResults, sarifResult{
			RuleID:     check.ID,
			Level:      sarifLevel(check.Severity),
			Message:    sarifMessage{Text: message},
			Properties: properties,
		})
	}

	return sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:  "core qa security",
					Rules: rules,
				},
			},
			Results: sarifResults,
		}},
	}
}

func sarifLevel(severity string) string {
	switch core.Lower(severity) {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}
