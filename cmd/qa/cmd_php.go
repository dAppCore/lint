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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/lint/pkg/detect"
	"forge.lthn.ai/core/lint/pkg/php"
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
func addPHPCommands(parent *cli.Command) {
	addPHPFmtCommand(parent)
	addPHPStanCommand(parent)
	addPHPPsalmCommand(parent)
	addPHPAuditCommand(parent)
	addPHPSecurityCommand(parent)
	addPHPRectorCommand(parent)
	addPHPInfectionCommand(parent)
	addPHPTestCommand(parent)
}

// PHP fmt command flags.
var (
	phpFmtFix  bool
	phpFmtDiff bool
	phpFmtJSON bool
)

func addPHPFmtCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "fmt",
		Short: "Format PHP code with Laravel Pint",
		Long:  "Run Laravel Pint to check or fix PHP code style. Uses --test mode by default; pass --fix to apply changes.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			if !isMachineReadableOutput(phpFmtJSON) {
				cli.Print("%s %s\n", headerStyle.Render("PHP Format"), dimStyle.Render("(Pint)"))
				cli.Blank()
			}

			return php.Format(context.Background(), php.FormatOptions{
				Dir:  cwd,
				Fix:  phpFmtFix,
				Diff: phpFmtDiff,
				JSON: phpFmtJSON,
			})
		},
	}

	cmd.Flags().BoolVar(&phpFmtFix, "fix", false, "Apply formatting fixes")
	cmd.Flags().BoolVar(&phpFmtDiff, "diff", false, "Show diff of changes")
	cmd.Flags().BoolVar(&phpFmtJSON, "json", false, "Output results as JSON")

	parent.AddCommand(cmd)
}

// PHP stan command flags.
var (
	phpStanLevel  int
	phpStanMemory string
	phpStanJSON   bool
	phpStanSARIF  bool
)

func addPHPStanCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "stan",
		Short: "Run PHPStan static analysis",
		Long:  "Run PHPStan (or Larastan) to find bugs in PHP code through static analysis.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			analyser, found := php.DetectAnalyser(cwd)
			if !found {
				return cli.Err("no static analyser found (install PHPStan: composer require phpstan/phpstan --dev)")
			}

			if !isMachineReadableOutput(phpStanJSON, phpStanSARIF) {
				cli.Print("%s %s\n", headerStyle.Render("PHP Static Analysis"), dimStyle.Render(fmt.Sprintf("(%s)", analyser)))
				cli.Blank()
			}

			err = php.Analyse(context.Background(), php.AnalyseOptions{
				Dir:    cwd,
				Level:  phpStanLevel,
				Memory: phpStanMemory,
				JSON:   phpStanJSON,
				SARIF:  phpStanSARIF,
			})
			if err != nil {
				return cli.Err("static analysis found issues")
			}

			if !isMachineReadableOutput(phpStanJSON, phpStanSARIF) {
				cli.Blank()
				cli.Print("%s\n", successStyle.Render("Static analysis passed"))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&phpStanLevel, "level", 0, "Analysis level (0-9, 0 uses config default)")
	cmd.Flags().StringVar(&phpStanMemory, "memory", "", "Memory limit (e.g., 2G)")
	cmd.Flags().BoolVar(&phpStanJSON, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&phpStanSARIF, "sarif", false, "Output results in SARIF format")

	parent.AddCommand(cmd)
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

func addPHPPsalmCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "psalm",
		Short: "Run Psalm static analysis",
		Long:  "Run Psalm for deep type-level static analysis of PHP code.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			_, found := php.DetectPsalm(cwd)
			if !found {
				return cli.Err("Psalm not found (install: composer require vimeo/psalm --dev)")
			}

			if !isMachineReadableOutput(phpPsalmJSON, phpPsalmSARIF) {
				cli.Print("%s\n", headerStyle.Render("PHP Psalm Analysis"))
				cli.Blank()
			}

			err = php.RunPsalm(context.Background(), php.PsalmOptions{
				Dir:      cwd,
				Level:    phpPsalmLevel,
				Fix:      phpPsalmFix,
				Baseline: phpPsalmBaseline,
				ShowInfo: phpPsalmShowInfo,
				JSON:     phpPsalmJSON,
				SARIF:    phpPsalmSARIF,
			})
			if err != nil {
				return cli.Err("Psalm found issues")
			}

			if !isMachineReadableOutput(phpPsalmJSON, phpPsalmSARIF) {
				cli.Blank()
				cli.Print("%s\n", successStyle.Render("Psalm analysis passed"))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&phpPsalmLevel, "level", 0, "Error level (1=strictest, 8=most lenient)")
	cmd.Flags().BoolVar(&phpPsalmFix, "fix", false, "Auto-fix issues where possible")
	cmd.Flags().BoolVar(&phpPsalmBaseline, "baseline", false, "Generate/update baseline file")
	cmd.Flags().BoolVar(&phpPsalmShowInfo, "show-info", false, "Show info-level issues")
	cmd.Flags().BoolVar(&phpPsalmJSON, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&phpPsalmSARIF, "sarif", false, "Output results in SARIF format")

	parent.AddCommand(cmd)
}

// PHP audit command flags.
var (
	phpAuditJSON bool
	phpAuditFix  bool
)

func addPHPAuditCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "audit",
		Short: "Audit PHP and npm dependencies for vulnerabilities",
		Long:  "Run composer audit and npm audit to check dependencies for known security vulnerabilities.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			if !isMachineReadableOutput(phpAuditJSON) {
				cli.Print("%s\n", headerStyle.Render("Dependency Audit"))
				cli.Blank()
			}

			results, err := php.RunAudit(context.Background(), php.AuditOptions{
				Dir:  cwd,
				JSON: phpAuditJSON,
				Fix:  phpAuditFix,
			})
			if err != nil {
				return err
			}

			if phpAuditJSON {
				payload := mapAuditResultsForJSON(results)
				data, err := json.MarshalIndent(payload, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))

				if payload.HasVulnerabilities {
					return cli.Err("vulnerabilities found in dependencies")
				}
				return nil
			}

			hasVulns := false
			for _, result := range results {
				if result.Error != nil {
					cli.Print("%s %s: %s\n", warningStyle.Render("!"), result.Tool, result.Error)
					continue
				}

				if result.Vulnerabilities > 0 {
					hasVulns = true
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
				} else {
					cli.Print("%s %s: no vulnerabilities found\n",
						successStyle.Render(cli.Glyph(":check:")),
						result.Tool)
				}
			}

			if hasVulns {
				return cli.Err("vulnerabilities found in dependencies")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&phpAuditJSON, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&phpAuditFix, "fix", false, "Auto-fix vulnerabilities (npm only)")

	parent.AddCommand(cmd)
}

// PHP security command flags.
var (
	phpSecuritySeverity string
	phpSecurityJSON     bool
	phpSecuritySARIF    bool
	phpSecurityURL      string
)

func addPHPSecurityCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "security",
		Short: "Run security checks on the PHP project",
		Long:  "Check for common security issues including dependency vulnerabilities, .env exposure, debug mode, and more.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			if !isMachineReadableOutput(phpSecurityJSON, phpSecuritySARIF) {
				cli.Print("%s\n", headerStyle.Render("Security Checks"))
				cli.Blank()
			}

			result, err := php.RunSecurityChecks(context.Background(), php.SecurityOptions{
				Dir:      cwd,
				Severity: phpSecuritySeverity,
				JSON:     phpSecurityJSON,
				SARIF:    phpSecuritySARIF,
				URL:      phpSecurityURL,
			})
			if err != nil {
				return err
			}

			result.Checks = sortSecurityChecks(result.Checks)

			if phpSecuritySARIF {
				data, err := json.MarshalIndent(mapSecurityResultForSARIF(result), "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))

				summary := result.Summary
				if summary.Critical > 0 || summary.High > 0 {
					return cli.Err("security checks failed")
				}
				return nil
			}

			if phpSecurityJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))

				summary := result.Summary
				if summary.Critical > 0 || summary.High > 0 {
					return cli.Err("security checks failed")
				}
				return nil
			}

			// Print each check result
			for _, check := range result.Checks {
				if check.Passed {
					cli.Print("%s %s\n",
						successStyle.Render(cli.Glyph(":check:")),
						check.Name)
				} else {
					style := getSeverityStyle(check.Severity)
					cli.Print("%s %s %s\n",
						errorStyle.Render(cli.Glyph(":cross:")),
						check.Name,
						style.Render(fmt.Sprintf("[%s]", check.Severity)))
					if check.Message != "" {
						cli.Print("  %s %s\n", dimStyle.Render("->"), check.Message)
					}
					if check.Fix != "" {
						cli.Print("  %s Fix: %s\n", dimStyle.Render("->"), check.Fix)
					}
				}
			}

			// Print summary
			cli.Blank()
			summary := result.Summary
			cli.Print("%s: %d/%d checks passed\n",
				headerStyle.Render("Summary"),
				summary.Passed, summary.Total)

			if summary.Critical > 0 {
				cli.Print("  %s\n", criticalStyle.Render(fmt.Sprintf("%d critical", summary.Critical)))
			}
			if summary.High > 0 {
				cli.Print("  %s\n", highStyle.Render(fmt.Sprintf("%d high", summary.High)))
			}
			if summary.Medium > 0 {
				cli.Print("  %s\n", mediumStyle.Render(fmt.Sprintf("%d medium", summary.Medium)))
			}
			if summary.Low > 0 {
				cli.Print("  %s\n", lowStyle.Render(fmt.Sprintf("%d low", summary.Low)))
			}

			if summary.Critical > 0 || summary.High > 0 {
				return cli.Err("security checks failed")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&phpSecuritySeverity, "severity", "", "Minimum severity to report (critical, high, medium, low)")
	cmd.Flags().BoolVar(&phpSecurityJSON, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&phpSecuritySARIF, "sarif", false, "Output results in SARIF format")
	cmd.Flags().StringVar(&phpSecurityURL, "url", "", "URL to check HTTP security headers")

	parent.AddCommand(cmd)
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

func addPHPRectorCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "rector",
		Short: "Run Rector for automated PHP code refactoring",
		Long:  "Run Rector to apply automated code refactoring rules. Uses dry-run by default; pass --fix to apply changes.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			if !php.DetectRector(cwd) {
				return cli.Err("Rector not found (install: composer require rector/rector --dev)")
			}

			mode := "dry-run"
			if phpRectorFix {
				mode = "apply"
			}
			cli.Print("%s %s\n", headerStyle.Render("Rector Refactoring"), dimStyle.Render(fmt.Sprintf("(%s)", mode)))
			cli.Blank()

			err = php.RunRector(context.Background(), php.RectorOptions{
				Dir:        cwd,
				Fix:        phpRectorFix,
				Diff:       phpRectorDiff,
				ClearCache: phpRectorClearCache,
			})
			if err != nil {
				return cli.Err("Rector found refactoring suggestions")
			}

			cli.Blank()
			cli.Print("%s\n", successStyle.Render("Rector check passed"))
			return nil
		},
	}

	cmd.Flags().BoolVar(&phpRectorFix, "fix", false, "Apply refactoring changes")
	cmd.Flags().BoolVar(&phpRectorDiff, "diff", false, "Show detailed diff of changes")
	cmd.Flags().BoolVar(&phpRectorClearCache, "clear-cache", false, "Clear cache before running")

	parent.AddCommand(cmd)
}

// PHP infection command flags.
var (
	phpInfectionMinMSI        int
	phpInfectionMinCoveredMSI int
	phpInfectionThreads       int
	phpInfectionFilter        string
	phpInfectionOnlyCovered   bool
)

func addPHPInfectionCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "infection",
		Short: "Run Infection mutation testing",
		Long:  "Run Infection to test mutation coverage. Mutates code and verifies tests catch the mutations.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			if !php.DetectInfection(cwd) {
				return cli.Err("Infection not found (install: composer require infection/infection --dev)")
			}

			cli.Print("%s\n", headerStyle.Render("Mutation Testing"))
			cli.Blank()

			err = php.RunInfection(context.Background(), php.InfectionOptions{
				Dir:           cwd,
				MinMSI:        phpInfectionMinMSI,
				MinCoveredMSI: phpInfectionMinCoveredMSI,
				Threads:       phpInfectionThreads,
				Filter:        phpInfectionFilter,
				OnlyCovered:   phpInfectionOnlyCovered,
			})
			if err != nil {
				return cli.Err("mutation testing did not pass minimum thresholds")
			}

			cli.Blank()
			cli.Print("%s\n", successStyle.Render("Mutation testing passed"))
			return nil
		},
	}

	cmd.Flags().IntVar(&phpInfectionMinMSI, "min-msi", 0, "Minimum mutation score indicator (0-100, default 50)")
	cmd.Flags().IntVar(&phpInfectionMinCoveredMSI, "min-covered-msi", 0, "Minimum covered mutation score (0-100, default 70)")
	cmd.Flags().IntVar(&phpInfectionThreads, "threads", 0, "Number of parallel threads (default 4)")
	cmd.Flags().StringVar(&phpInfectionFilter, "filter", "", "Filter files by pattern")
	cmd.Flags().BoolVar(&phpInfectionOnlyCovered, "only-covered", false, "Only mutate covered code")

	parent.AddCommand(cmd)
}

// PHP test command flags.
var (
	phpTestParallel bool
	phpTestCoverage bool
	phpTestFilter   string
	phpTestGroup    string
	phpTestJUnit    bool
)

func addPHPTestCommand(parent *cli.Command) {
	cmd := &cli.Command{
		Use:   "test",
		Short: "Run PHP tests with Pest or PHPUnit",
		Long:  "Detect and run the PHP test suite. Automatically detects Pest or PHPUnit.",
		RunE: func(cmd *cli.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !detect.IsPHPProject(cwd) {
				return cli.Err("not a PHP project (no composer.json found)")
			}

			runner := php.DetectTestRunner(cwd)
			if !isMachineReadableOutput(phpTestJUnit) {
				cli.Print("%s %s\n", headerStyle.Render("PHP Tests"), dimStyle.Render(fmt.Sprintf("(%s)", runner)))
				cli.Blank()
			}

			var groups []string
			if phpTestGroup != "" {
				groups = strings.Split(phpTestGroup, ",")
			}

			err = php.RunTests(context.Background(), php.TestOptions{
				Dir:      cwd,
				Parallel: phpTestParallel,
				Coverage: phpTestCoverage,
				Filter:   phpTestFilter,
				Groups:   groups,
				JUnit:    phpTestJUnit,
			})
			if err != nil {
				return cli.Err("tests failed")
			}

			if !isMachineReadableOutput(phpTestJUnit) {
				cli.Blank()
				cli.Print("%s\n", successStyle.Render("All tests passed"))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&phpTestParallel, "parallel", false, "Run tests in parallel")
	cmd.Flags().BoolVar(&phpTestCoverage, "coverage", false, "Generate code coverage")
	cmd.Flags().StringVar(&phpTestFilter, "filter", "", "Filter tests by name pattern")
	cmd.Flags().StringVar(&phpTestGroup, "group", "", "Run only tests in specified groups (comma-separated)")
	cmd.Flags().BoolVar(&phpTestJUnit, "junit", false, "Output results in JUnit XML format")

	parent.AddCommand(cmd)
}

// getSeverityStyle returns a style for the given severity level.
func getSeverityStyle(severity string) *cli.AnsiStyle {
	switch strings.ToLower(severity) {
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
	switch strings.ToLower(severity) {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}
