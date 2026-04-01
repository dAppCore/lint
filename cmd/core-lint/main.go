package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"forge.lthn.ai/core/cli/pkg/cli"
	coreerr "forge.lthn.ai/core/go-log"
	cataloglint "forge.lthn.ai/core/lint"
	lintpkg "forge.lthn.ai/core/lint/pkg/lint"
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
		newRunCommand("compliance", "Run compliance linters", lintpkg.RunInput{Category: "compliance", SBOM: true}),
		newHookCommand(),
	)
}

func newRunCommand(commandName string, summary string, defaults lintpkg.RunInput) *cli.Command {
	var (
		output   string
		config   string
		failOn   string
		category string
		lang     string
		files    []string
		hook     bool
		ci       bool
		sbom     bool
	)

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		input := defaults
		input.Output = output
		input.Config = config
		input.FailOn = failOn
		input.Category = category
		input.Lang = lang
		input.Files = files
		input.Hook = hook
		input.CI = ci
		input.SBOM = sbom

		if len(args) > 0 {
			input.Path = args[0]
		}
		if input.Path == "" {
			input.Path = "."
		}

		output, err := resolvedOutput(input)
		if err != nil {
			return err
		}
		input.Output = output

		service := lintpkg.NewService()
		report, err := service.Run(context.Background(), input)
		if err != nil {
			return err
		}

		if err := writeReport(command.OutOrStdout(), input.Output, report); err != nil {
			return err
		}
		if !report.Summary.Passed {
			return coreerr.E("cmd."+commandName, "lint failed", nil)
		}
		return nil
	})

	cli.StringFlag(command, &output, "output", "o", defaults.Output, "Output format: json, text, github, sarif")
	cli.StringFlag(command, &config, "config", "c", defaults.Config, "Config path (default: .core/lint.yaml)")
	cli.StringFlag(command, &failOn, "fail-on", "", defaults.FailOn, "Fail threshold: error, warning, info")
	cli.StringFlag(command, &category, "category", "", defaults.Category, "Restrict to one category")
	cli.StringFlag(command, &lang, "lang", "l", defaults.Lang, "Restrict to one language")
	cli.StringSliceFlag(command, &files, "files", "", defaults.Files, "Restrict scanning to specific files")
	cli.BoolFlag(command, &hook, "hook", "", defaults.Hook, "Run in pre-commit mode against staged files")
	cli.BoolFlag(command, &ci, "ci", "", defaults.CI, "GitHub Actions mode (github annotations)")
	cli.BoolFlag(command, &sbom, "sbom", "", defaults.SBOM, "Enable compliance/SBOM tools")

	return command
}

func newDetectCommand(commandName string, summary string) *cli.Command {
	var output string

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		languages := lintpkg.Detect(path)
		switch output {
		case "", "text":
			for _, language := range languages {
				fmt.Fprintln(command.OutOrStdout(), language)
			}
			return nil
		case "json":
			return writeJSON(command.OutOrStdout(), languages)
		default:
			return coreerr.E("cmd.detect", "unsupported output format "+output, nil)
		}
	})

	cli.StringFlag(command, &output, "output", "o", "text", "Output format: text, json")
	return command
}

func newToolsCommand(commandName string, summary string) *cli.Command {
	var output string
	var language string

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		service := lintpkg.NewService()

		var languages []string
		if language != "" {
			languages = []string{language}
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
			return writeJSON(command.OutOrStdout(), tools)
		default:
			return coreerr.E("cmd.tools", "unsupported output format "+output, nil)
		}
	})

	cli.StringFlag(command, &output, "output", "o", "text", "Output format: text, json")
	cli.StringFlag(command, &language, "lang", "l", "", "Filter by language")
	return command
}

func newInitCommand(commandName string, summary string) *cli.Command {
	var force bool

	command := cli.NewCommand(commandName, summary, "", func(command *cli.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		service := lintpkg.NewService()
		writtenPath, err := service.WriteDefaultConfig(path, force)
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
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		service := lintpkg.NewService()
		if err := service.InstallHook(path); err != nil {
			return err
		}
		fmt.Fprintln(command.OutOrStdout(), "installed")
		return nil
	})

	removeCmd := cli.NewCommand("remove", "Remove the pre-commit hook", "", func(command *cli.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		service := lintpkg.NewService()
		if err := service.RemoveHook(path); err != nil {
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
		default:
			lintpkg.WriteText(command.OutOrStdout(), findings)
			if format == "text" && len(findings) > 0 {
				writeLegacySummary(command.OutOrStdout(), findings)
			}
			return nil
		}
	})

	cli.StringFlag(command, &format, "format", "f", "text", "Output format: text, json, jsonl")
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

func resolvedOutput(input lintpkg.RunInput) (string, error) {
	if input.Output != "" {
		return input.Output, nil
	}
	if input.CI {
		return "github", nil
	}

	config, _, err := lintpkg.LoadProjectConfig(input.Path, input.Config)
	if err != nil {
		return "", err
	}
	if config.Output != "" {
		return config.Output, nil
	}
	return "text", nil
}

func writeReport(writer io.Writer, output string, report lintpkg.Report) error {
	switch output {
	case "json":
		return lintpkg.WriteReportJSON(writer, report)
	case "text":
		lintpkg.WriteReportText(writer, report)
		return nil
	case "github":
		lintpkg.WriteReportGitHub(writer, report)
		return nil
	case "sarif":
		return lintpkg.WriteReportSARIF(writer, report)
	default:
		return coreerr.E("writeReport", "unsupported output format "+output, nil)
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeLegacySummary(writer io.Writer, findings []lintpkg.Finding) {
	summary := lintpkg.Summarise(findings)
	fmt.Fprintf(writer, "\n%d finding(s)", summary.Total)

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
		fmt.Fprintf(writer, " (%s)", strings.Join(parts, ", "))
	}
	fmt.Fprintln(writer)
}
