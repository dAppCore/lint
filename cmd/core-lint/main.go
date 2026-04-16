package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	cataloglint "dappco.re/go/core/lint"
	lintpkg "dappco.re/go/core/lint/pkg/lint"
	"forge.lthn.ai/core/cli/pkg/cli"
	coreerr "forge.lthn.ai/core/go-log"
)

func main() {
	cli.WithAppName("core-lint")
	cli.Main(cli.WithCommands("lint", addLintCommands))
}

func addLintCommands(root *cli.Command) {
	addRFCCommands(root)

	lintCmd := cli.NewGroup("lint", "Pattern-based code linter", "")
	lintCmd.AddCommand(newCheckCommand(), newCatalogCommand())
	addRFCCommands(lintCmd)

	root.AddCommand(lintCmd)
}

func addRFCCommands(parent *cli.Command) {
	parent.AddCommand(
		newRunCommand("run", "Run configured linters", lintpkg.RunInput{}),
		newDetectCommand("detect", "Detect project languages"),
		newToolsCommand("tools", "List supported linter tools"),
		newInitCommand("init", "Generate .core/lint.yaml"),
		newRunCommand("go", "Run Go linters", lintpkg.RunInput{Lang: "go"}),
		newRunCommand("php", "Run PHP linters", lintpkg.RunInput{Lang: "php"}),
		newRunCommand("js", "Run JS/TS linters", lintpkg.RunInput{Lang: "js"}),
		newRunCommand("python", "Run Python linters", lintpkg.RunInput{Lang: "python"}),
		newRunCommand("security", "Run security linters", lintpkg.RunInput{Category: "security"}),
		newRunCommand("compliance", "Run compliance linters", lintpkg.RunInput{Category: "compliance"}),
		newHookCommand(),
	)
}

func newRunCommand(commandName string, summary string, defaults lintpkg.RunInput) *cli.Command {
	var (
		outputFormat string
		configPath   string
		scheduleName string
		failOnLevel  string
		categoryName string
		languageName string
		filePaths    []string
		hookMode     bool
		ciMode       bool
		sbomMode     bool
	)

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		input := defaults
		input.Output = outputFormat
		input.Config = configPath
		input.Schedule = scheduleName
		input.FailOn = failOnLevel
		input.Category = categoryName
		input.Lang = languageName
		input.Files = filePaths
		input.Hook = hookMode
		input.CI = ciMode
		input.SBOM = sbomMode

		if len(args) > 0 {
			input.Path = args[0]
		}
		if input.Path == "" {
			input.Path = "."
		}

		resolvedOutputFormat, err := lintpkg.ResolveRunOutputFormat(input)
		if err != nil {
			return err
		}
		input.Output = resolvedOutputFormat

		service := lintpkg.NewService()
		report, err := service.Run(context.Background(), input)
		if err != nil {
			return err
		}

		if err := writeReport(command.OutOrStdout(), input.Output, report); err != nil {
			return err
		}
		if !report.Summary.Passed {
			return coreerr.E(
				"cmd."+commandName,
				fmt.Sprintf(
					"lint failed (fail-on=%s): %d error(s), %d warning(s), %d info finding(s)",
					input.FailOn,
					report.Summary.Errors,
					report.Summary.Warnings,
					report.Summary.Info,
				),
				nil,
			)
		}
		return nil
	})

	cli.StringFlag(command, &outputFormat, "output", "o", defaults.Output, "Output format: json, text, github, sarif")
	cli.StringFlag(command, &configPath, "config", "c", defaults.Config, "Config path (default: .core/lint.yaml)")
	cli.StringFlag(command, &scheduleName, "schedule", "", "", "Run a named schedule from the config")
	cli.StringFlag(command, &failOnLevel, "fail-on", "", defaults.FailOn, "Fail threshold: error, warning, info")
	cli.StringFlag(command, &categoryName, "category", "", defaults.Category, "Restrict to one category")
	cli.StringFlag(command, &languageName, "lang", "l", defaults.Lang, "Restrict to one language")
	cli.StringSliceFlag(command, &filePaths, "files", "", defaults.Files, "Restrict scanning to specific files")
	cli.BoolFlag(command, &hookMode, "hook", "", defaults.Hook, "Run in pre-commit mode against staged files")
	cli.BoolFlag(command, &ciMode, "ci", "", defaults.CI, "GitHub Actions mode (github annotations)")
	cli.BoolFlag(command, &sbomMode, "sbom", "", defaults.SBOM, "Enable compliance/SBOM tools")

	return command
}

func newDetectCommand(commandName string, summary string) *cli.Command {
	var output string

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		projectPath := "."
		if len(args) > 0 {
			projectPath = args[0]
		}

		languages := lintpkg.Detect(projectPath)
		switch output {
		case "", "text":
			for _, language := range languages {
				fmt.Fprintln(command.OutOrStdout(), language)
			}
			return nil
		case "json":
			return writeIndentedJSON(command.OutOrStdout(), languages)
		default:
			return coreerr.E("cmd.detect", "unsupported output format "+output, nil)
		}
	})

	cli.StringFlag(command, &output, "output", "o", "text", "Output format: text, json")
	return command
}

func newToolsCommand(commandName string, summary string) *cli.Command {
	var output string
	var languageFilter string

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		service := lintpkg.NewService()

		var languages []string
		if languageFilter != "" {
			languages = []string{languageFilter}
		}

		tools := service.Tools(languages)
		switch output {
		case "", "text":
			for _, tool := range tools {
				status := "missing"
				if tool.Available {
					status = "available"
				}
				line := fmt.Sprintf("%-14s [%-11s] %s langs=%s", tool.Name, tool.Category, status, strings.Join(tool.Languages, ","))
				if tool.Entitlement != "" {
					line += " entitlement=" + tool.Entitlement
				}
				fmt.Fprintln(command.OutOrStdout(), line)
			}
			return nil
		case "json":
			return writeIndentedJSON(command.OutOrStdout(), tools)
		default:
			return coreerr.E("cmd.tools", "unsupported output format "+output, nil)
		}
	})

	cli.StringFlag(command, &output, "output", "o", "text", "Output format: text, json")
	cli.StringFlag(command, &languageFilter, "lang", "l", "", "Filter by language")
	return command
}

func newInitCommand(commandName string, summary string) *cli.Command {
	var force bool

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		projectPath := "."
		if len(args) > 0 {
			projectPath = args[0]
		}

		service := lintpkg.NewService()
		writtenPath, err := service.WriteDefaultConfig(projectPath, force)
		if err != nil {
			return err
		}
		fmt.Fprintln(command.OutOrStdout(), writtenPath)
		return nil
	})

	cli.BoolFlag(command, &force, "force", "f", false, "Overwrite an existing config")
	return command
}

func newHookCommand() *cli.Command {
	hookCmd := cli.NewGroup("hook", "Install or remove the git pre-commit hook", "")

	installCmd := cli.NewCommand("install", "Install the pre-commit hook", "", func(command *cli.Command, args []string) error {
		projectPath := "."
		if len(args) > 0 {
			projectPath = args[0]
		}

		service := lintpkg.NewService()
		if err := service.InstallHook(projectPath); err != nil {
			return err
		}
		fmt.Fprintln(command.OutOrStdout(), "installed")
		return nil
	})

	removeCmd := cli.NewCommand("remove", "Remove the pre-commit hook", "", func(command *cli.Command, args []string) error {
		projectPath := "."
		if len(args) > 0 {
			projectPath = args[0]
		}

		service := lintpkg.NewService()
		if err := service.RemoveHook(projectPath); err != nil {
			return err
		}
		fmt.Fprintln(command.OutOrStdout(), "removed")
		return nil
	})

	hookCmd.AddCommand(installCmd, removeCmd)
	return hookCmd
}

func newCheckCommand() *cli.Command {
	var (
		format   string
		language string
		severity string
	)

	command := cli.NewCommand("check", "Scan files for pattern matches", "", func(command *cli.Command, args []string) error {
		catalog, err := cataloglint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.check", "loading catalog", err)
		}

		rules := catalog.Rules
		if language != "" {
			rules = catalog.ForLanguage(language)
			if len(rules) == 0 {
				fmt.Fprintf(os.Stderr, "no rules for language %q\n", language)
				return nil
			}
		}
		if severity != "" {
			filtered := (&lintpkg.Catalog{Rules: rules}).AtSeverity(severity)
			if len(filtered) == 0 {
				fmt.Fprintf(os.Stderr, "no rules at severity %q or above\n", severity)
				return nil
			}
			rules = filtered
		}

		scanner, err := lintpkg.NewScanner(rules)
		if err != nil {
			return coreerr.E("cmd.check", "creating scanner", err)
		}

		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		var findings []lintpkg.Finding
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				return coreerr.E("cmd.check", "stat "+path, err)
			}

			if info.IsDir() {
				pathFindings, err := scanner.ScanDir(path)
				if err != nil {
					return err
				}
				findings = append(findings, pathFindings...)
				continue
			}

			pathFindings, err := scanner.ScanFile(path)
			if err != nil {
				return err
			}
			findings = append(findings, pathFindings...)
		}

		switch format {
		case "json":
			return lintpkg.WriteJSON(command.OutOrStdout(), findings)
		case "jsonl":
			return lintpkg.WriteJSONL(command.OutOrStdout(), findings)
		case "sarif":
			report := lintpkg.Report{
				Findings: findings,
				Summary:  lintpkg.Summarise(findings),
			}
			return lintpkg.WriteReportSARIF(command.OutOrStdout(), report)
		default:
			if err := lintpkg.WriteText(command.OutOrStdout(), findings); err != nil {
				return err
			}
			if format == "text" && len(findings) > 0 {
				if err := writeCatalogSummary(command.OutOrStdout(), findings); err != nil {
					return err
				}
			}
			return nil
		}
	})

	cli.StringFlag(command, &format, "format", "f", "text", "Output format: text, json, jsonl, sarif")
	cli.StringFlag(command, &language, "lang", "l", "", "Filter rules by language")
	cli.StringFlag(command, &severity, "severity", "s", "", "Minimum severity threshold (info, low, medium, high, critical)")

	return command
}

func newCatalogCommand() *cli.Command {
	catalogCmd := cli.NewGroup("catalog", "Browse the pattern catalog", "")

	var listLanguage string
	listCmd := cli.NewCommand("list", "List all rules in the catalog", "", func(command *cli.Command, args []string) error {
		catalog, err := cataloglint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.catalog.list", "loading catalog", err)
		}

		rules := catalog.Rules
		if listLanguage != "" {
			rules = catalog.ForLanguage(listLanguage)
		}
		if len(rules) == 0 {
			fmt.Fprintln(command.OutOrStdout(), "No rules found.")
			return nil
		}

		rules = append([]lintpkg.Rule(nil), rules...)
		sort.Slice(rules, func(left int, right int) bool {
			if rules[left].Severity == rules[right].Severity {
				return strings.Compare(rules[left].ID, rules[right].ID) < 0
			}
			return strings.Compare(rules[left].Severity, rules[right].Severity) < 0
		})

		for _, rule := range rules {
			fmt.Fprintf(command.OutOrStdout(), "%-14s [%-8s] %s\n", rule.ID, rule.Severity, rule.Title)
		}
		fmt.Fprintf(os.Stderr, "\n%d rule(s)\n", len(rules))
		return nil
	})
	cli.StringFlag(listCmd, &listLanguage, "lang", "l", "", "Filter by language")

	showCmd := cli.NewCommand("show", "Show details of a specific rule", "", func(command *cli.Command, args []string) error {
		if len(args) == 0 {
			return coreerr.E("cmd.catalog.show", "rule ID required", nil)
		}

		catalog, err := cataloglint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.catalog.show", "loading catalog", err)
		}

		rule := catalog.ByID(args[0])
		if rule == nil {
			return coreerr.E("cmd.catalog.show", "rule "+args[0]+" not found", nil)
		}

		data, err := json.MarshalIndent(rule, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(command.OutOrStdout(), "%s\n", string(data))
		return nil
	})

	catalogCmd.AddCommand(listCmd, showCmd)
	return catalogCmd
}

func writeReport(writer io.Writer, output string, report lintpkg.Report) error {
	switch output {
	case "json":
		return lintpkg.WriteReportJSON(writer, report)
	case "text":
		return lintpkg.WriteReportText(writer, report)
	case "github":
		return lintpkg.WriteReportGitHub(writer, report)
	case "sarif":
		return lintpkg.WriteReportSARIF(writer, report)
	default:
		return coreerr.E("writeReport", "unsupported output format "+output, nil)
	}
}

func writeIndentedJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeCatalogSummary(writer io.Writer, findings []lintpkg.Finding) error {
	summary := lintpkg.Summarise(findings)
	if _, err := fmt.Fprintf(writer, "\n%d finding(s)", summary.Total); err != nil {
		return err
	}

	orderedSeverities := []string{"critical", "high", "medium", "low", "info", "error", "warning"}
	seen := make(map[string]bool, len(summary.BySeverity))
	var parts []string

	for _, severity := range orderedSeverities {
		count := summary.BySeverity[severity]
		if count == 0 {
			continue
		}
		seen[severity] = true
		parts = append(parts, fmt.Sprintf("%d %s", count, severity))
	}

	var extraSeverities []string
	for severity := range summary.BySeverity {
		if seen[severity] {
			continue
		}
		extraSeverities = append(extraSeverities, severity)
	}
	sort.Strings(extraSeverities)
	for _, severity := range extraSeverities {
		count := summary.BySeverity[severity]
		if count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s", count, severity))
	}

	if len(parts) > 0 {
		if _, err := fmt.Fprintf(writer, " (%s)", strings.Join(parts, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	return nil
}
