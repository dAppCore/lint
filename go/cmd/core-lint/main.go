package main

import (
	"context"
	"sort"

	core "dappco.re/go"
	cataloglint "dappco.re/go/lint"
	lintpkg "dappco.re/go/lint/pkg/lint"
)

const (
	mainCmdCatalogShow2419de          = "cmd.catalog.show"
	mainCmdCheck11d6fd                = "cmd.check"
	mainLoadingCataloge27725          = "loading catalog"
	mainUnsupportedOutputFormat785877 = "unsupported output format "
)

type checkOptions struct {
	format   string
	language string
	severity string
}

type parsedArgs struct {
	flags       map[string][]string
	positionals []string
}

type checkRulesResult struct {
	rules []lintpkg.Rule
	ok    bool
}

type commandWriters struct {
	stdout core.Writer
	stderr core.Writer
}

func main() {
	writers := commandWriters{stdout: core.Stdout(), stderr: core.Stderr()}
	if result := runRoot(core.Args()[1:], writers); !result.OK {
		core.WriteString(writers.stderr, result.Error()+"\n")
		core.Exit(1)
	}
}

func runRoot(args []string, writers commandWriters) core.Result {
	if len(args) == 0 {
		return printRootHelp(writers.stdout)
	}
	command := args[0]
	rest := args[1:]
	switch command {
	case "run":
		return runRunCommand(command, rest, lintpkg.RunInput{}, writers)
	case "go":
		return runRunCommand(command, rest, lintpkg.RunInput{Lang: "go"}, writers)
	case "php":
		return runRunCommand(command, rest, lintpkg.RunInput{Lang: "php"}, writers)
	case "js":
		return runRunCommand(command, rest, lintpkg.RunInput{Lang: "js"}, writers)
	case "python":
		return runRunCommand(command, rest, lintpkg.RunInput{Lang: "python"}, writers)
	case "security":
		return runRunCommand(command, rest, lintpkg.RunInput{Category: "security"}, writers)
	case "compliance":
		return runRunCommand(command, rest, lintpkg.RunInput{Category: "compliance"}, writers)
	case "detect":
		return runDetectCommand(rest, writers)
	case "tools":
		return runToolsCommand(rest, writers)
	case "init":
		return runInitCommand(rest, writers)
	case "hook":
		return runHookCommand(rest, writers)
	case "lint":
		return runLintNamespace(rest, writers)
	case "catalog":
		return runCatalogNamespace(rest, writers)
	default:
		return core.Fail(core.E("cmd", "unknown command "+command, nil))
	}
}

func printRootHelp(stdout core.Writer) core.Result {
	return core.WriteString(stdout, "Commands: run, detect, tools, init, hook, lint, catalog\n")
}

func runLintNamespace(args []string, writers commandWriters) core.Result {
	if len(args) == 0 {
		return printRootHelp(writers.stdout)
	}
	switch args[0] {
	case "check":
		return runCheckCommand(writers.stdout, writers.stderr, args[1:], parseCheckOptions(args[1:]))
	case "catalog":
		return runCatalogNamespace(args[1:], writers)
	case "run":
		return runRunCommand("run", args[1:], lintpkg.RunInput{}, writers)
	default:
		return core.Fail(core.E("cmd.lint", "unknown lint command "+args[0], nil))
	}
}

func runRunCommand(commandName string, args []string, defaults lintpkg.RunInput, writers commandWriters) core.Result {
	input := runInputFromArgs(args, defaults)
	resolvedOutputFormat := lintpkg.ResolveRunOutputFormat(input)
	if !resolvedOutputFormat.OK {
		return resolvedOutputFormat
	}
	input.Output = resolvedOutputFormat.Value.(string)

	reportResult := lintpkg.NewService().Run(context.Background(), input)
	if !reportResult.OK {
		return reportResult
	}
	report := reportResult.Value.(lintpkg.Report)

	if written := writeReport(writers.stdout, input.Output, report); !written.OK {
		return written
	}
	if !report.Summary.Passed {
		return core.Fail(core.E(
			"cmd."+commandName,
			core.Sprintf(
				"lint failed (fail-on=%s): %d error(s), %d warning(s), %d info finding(s)",
				input.FailOn,
				report.Summary.Errors,
				report.Summary.Warnings,
				report.Summary.Info,
			),
			nil,
		))
	}
	return core.Ok(nil)
}

func runInputFromArgs(args []string, defaults lintpkg.RunInput) lintpkg.RunInput {
	parsed := parseArgs(args)
	input := defaults
	input.Output = firstFlag(parsed, "output", "o", defaults.Output)
	input.Config = firstFlag(parsed, "config", "c", defaults.Config)
	input.Schedule = firstFlag(parsed, "schedule", "", defaults.Schedule)
	input.FailOn = firstFlag(parsed, "fail-on", "", defaults.FailOn)
	input.Category = firstFlag(parsed, "category", "", defaults.Category)
	input.Lang = firstFlag(parsed, "lang", "l", defaults.Lang)
	input.Files = parsed.flags["files"]
	input.Hook = boolFlag(parsed, "hook", defaults.Hook)
	input.CI = boolFlag(parsed, "ci", defaults.CI)
	input.SBOM = boolFlag(parsed, "sbom", defaults.SBOM)
	if len(parsed.positionals) > 0 {
		input.Path = parsed.positionals[0]
	}
	if input.Path == "" {
		input.Path = "."
	}
	return input
}

func runDetectCommand(args []string, writers commandWriters) core.Result {
	parsed := parseArgs(args)
	projectPath := "."
	if len(parsed.positionals) > 0 {
		projectPath = parsed.positionals[0]
	}
	languages := lintpkg.Detect(projectPath)
	output := firstFlag(parsed, "output", "o", "text")
	switch output {
	case "", "text":
		for _, language := range languages {
			if written := core.WriteString(writers.stdout, language+"\n"); !written.OK {
				return written
			}
		}
		return core.Ok(nil)
	case "json":
		return writeIndentedJSON(writers.stdout, languages)
	default:
		return core.Fail(core.E("cmd.detect", mainUnsupportedOutputFormat785877+output, nil))
	}
}

func runToolsCommand(args []string, writers commandWriters) core.Result {
	parsed := parseArgs(args)
	output := firstFlag(parsed, "output", "o", "text")
	languageFilter := firstFlag(parsed, "lang", "l", "")
	var languages []string
	if languageFilter != "" {
		languages = []string{languageFilter}
	}
	tools := lintpkg.NewService().Tools(languages)
	switch output {
	case "", "text":
		for _, tool := range tools {
			status := "missing"
			if tool.Available {
				status = "available"
			}
			line := core.Sprintf("%-14s [%-11s] %s langs=%s", tool.Name, tool.Category, status, core.Join(",", tool.Languages...))
			if tool.Entitlement != "" {
				line += " entitlement=" + tool.Entitlement
			}
			if written := core.WriteString(writers.stdout, line+"\n"); !written.OK {
				return written
			}
		}
		return core.Ok(nil)
	case "json":
		return writeIndentedJSON(writers.stdout, tools)
	default:
		return core.Fail(core.E("cmd.tools", mainUnsupportedOutputFormat785877+output, nil))
	}
}

func runInitCommand(args []string, writers commandWriters) core.Result {
	parsed := parseArgs(args)
	projectPath := "."
	if len(parsed.positionals) > 0 {
		projectPath = parsed.positionals[0]
	}
	result := lintpkg.NewService().WriteDefaultConfig(projectPath, boolFlag(parsed, "force", false))
	if !result.OK {
		return result
	}
	return core.WriteString(writers.stdout, result.Value.(string)+"\n")
}

func runHookCommand(args []string, writers commandWriters) core.Result {
	if len(args) == 0 {
		return core.Fail(core.E("cmd.hook", "subcommand required", nil))
	}
	parsed := parseArgs(args[1:])
	projectPath := "."
	if len(parsed.positionals) > 0 {
		projectPath = parsed.positionals[0]
	}
	service := lintpkg.NewService()
	switch args[0] {
	case "install":
		if result := service.InstallHook(projectPath); !result.OK {
			return result
		}
		return core.WriteString(writers.stdout, "installed\n")
	case "remove":
		if result := service.RemoveHook(projectPath); !result.OK {
			return result
		}
		return core.WriteString(writers.stdout, "removed\n")
	default:
		return core.Fail(core.E("cmd.hook", "unknown hook command "+args[0], nil))
	}
}

func parseCheckOptions(args []string) checkOptions {
	parsed := parseArgs(args)
	return checkOptions{
		format:   firstFlag(parsed, "format", "f", "text"),
		language: firstFlag(parsed, "lang", "l", ""),
		severity: firstFlag(parsed, "severity", "s", ""),
	}
}

func runCheckCommand(stdout core.Writer, stderr core.Writer, args []string, opts checkOptions) core.Result {
	catalogResult := cataloglint.LoadEmbeddedCatalog()
	if !catalogResult.OK {
		err, _ := catalogResult.Value.(error)
		return core.Fail(core.E(mainCmdCheck11d6fd, mainLoadingCataloge27725, err))
	}
	catalog := catalogResult.Value.(*lintpkg.Catalog)

	rulesResult := checkRules(stderr, catalog, opts)
	if !rulesResult.OK {
		return rulesResult
	}
	rules := rulesResult.Value.(checkRulesResult)
	if !rules.ok {
		return core.Ok(nil)
	}

	scannerResult := lintpkg.NewScanner(rules.rules)
	if !scannerResult.OK {
		err, _ := scannerResult.Value.(error)
		return core.Fail(core.E(mainCmdCheck11d6fd, "creating scanner", err))
	}
	findingsResult := scanCheckPaths(scannerResult.Value.(*lintpkg.Scanner), parseArgs(args).positionals)
	if !findingsResult.OK {
		return findingsResult
	}
	return writeCheckFindings(stdout, opts.format, findingsResult.Value.([]lintpkg.Finding))
}

func checkRules(stderr core.Writer, catalog *lintpkg.Catalog, opts checkOptions) core.Result {
	rules := catalog.Rules
	if opts.language != "" {
		rules = catalog.ForLanguage(opts.language)
		if len(rules) == 0 {
			if written := core.WriteString(stderr, core.Sprintf("no rules for language %q\n", opts.language)); !written.OK {
				return written
			}
			return core.Ok(checkRulesResult{ok: false})
		}
	}
	if opts.severity == "" {
		return core.Ok(checkRulesResult{rules: rules, ok: true})
	}
	filtered := (&lintpkg.Catalog{Rules: rules}).AtSeverity(opts.severity)
	if len(filtered) == 0 {
		if written := core.WriteString(stderr, core.Sprintf("no rules at severity %q or above\n", opts.severity)); !written.OK {
			return written
		}
		return core.Ok(checkRulesResult{ok: false})
	}
	return core.Ok(checkRulesResult{rules: filtered, ok: true})
}

func scanCheckPaths(scanner *lintpkg.Scanner, args []string) core.Result {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var findings []lintpkg.Finding
	for _, path := range paths {
		pathFindings := scanCheckPath(scanner, path)
		if !pathFindings.OK {
			return pathFindings
		}
		findings = append(findings, pathFindings.Value.([]lintpkg.Finding)...)
	}
	return core.Ok(findings)
}

func scanCheckPath(scanner *lintpkg.Scanner, path string) core.Result {
	infoResult := core.Stat(path)
	if !infoResult.OK {
		err, _ := infoResult.Value.(error)
		return core.Fail(core.E(mainCmdCheck11d6fd, "stat "+path, err))
	}
	if infoResult.Value.(core.FsFileInfo).IsDir() {
		return scanner.ScanDir(path)
	}
	return scanner.ScanFile(path)
}

func writeCheckFindings(writer core.Writer, format string, findings []lintpkg.Finding) core.Result {
	switch format {
	case "json":
		return lintpkg.WriteJSON(writer, findings)
	case "jsonl":
		return lintpkg.WriteJSONL(writer, findings)
	case "sarif":
		return lintpkg.WriteReportSARIF(writer, lintpkg.Report{
			Findings: findings,
			Summary:  lintpkg.Summarise(findings),
		})
	default:
		return writeTextCheckFindings(writer, format, findings)
	}
}

func writeTextCheckFindings(writer core.Writer, format string, findings []lintpkg.Finding) core.Result {
	if written := lintpkg.WriteText(writer, findings); !written.OK {
		return written
	}
	if format != "text" || len(findings) == 0 {
		return core.Ok(nil)
	}
	return writeCatalogSummary(writer, findings)
}

func runCatalogNamespace(args []string, writers commandWriters) core.Result {
	if len(args) == 0 {
		return core.Fail(core.E("cmd.catalog", "subcommand required", nil))
	}
	switch args[0] {
	case "list":
		return runCatalogList(writers.stdout, writers.stderr, parseArgs(args[1:]))
	case "show":
		return runCatalogShow(writers.stdout, args[1:])
	default:
		return core.Fail(core.E("cmd.catalog", "unknown catalog command "+args[0], nil))
	}
}

func runCatalogList(stdout core.Writer, stderr core.Writer, parsed parsedArgs) core.Result {
	catalogResult := cataloglint.LoadEmbeddedCatalog()
	if !catalogResult.OK {
		err, _ := catalogResult.Value.(error)
		return core.Fail(core.E("cmd.catalog.list", mainLoadingCataloge27725, err))
	}
	catalog := catalogResult.Value.(*lintpkg.Catalog)
	rules := catalog.Rules
	if language := firstFlag(parsed, "lang", "l", ""); language != "" {
		rules = catalog.ForLanguage(language)
	}
	if len(rules) == 0 {
		return core.WriteString(stdout, "No rules found.\n")
	}
	for _, rule := range sortedCatalogRules(rules) {
		if written := core.WriteString(stdout, core.Sprintf("%-14s [%-8s] %s\n", rule.ID, rule.Severity, rule.Title)); !written.OK {
			return written
		}
	}
	return core.WriteString(stderr, core.Sprintf("\n%d rule(s)\n", len(rules)))
}

func sortedCatalogRules(rules []lintpkg.Rule) []lintpkg.Rule {
	sorted := append([]lintpkg.Rule(nil), rules...)
	sort.Slice(sorted, func(left int, right int) bool {
		if sorted[left].Severity == sorted[right].Severity {
			return core.Compare(sorted[left].ID, sorted[right].ID) < 0
		}
		return core.Compare(sorted[left].Severity, sorted[right].Severity) < 0
	})
	return sorted
}

func runCatalogShow(stdout core.Writer, args []string) core.Result {
	if len(args) == 0 {
		return core.Fail(core.E(mainCmdCatalogShow2419de, "rule ID required", nil))
	}
	catalogResult := cataloglint.LoadEmbeddedCatalog()
	if !catalogResult.OK {
		err, _ := catalogResult.Value.(error)
		return core.Fail(core.E(mainCmdCatalogShow2419de, mainLoadingCataloge27725, err))
	}
	rule := catalogResult.Value.(*lintpkg.Catalog).ByID(args[0])
	if rule == nil {
		return core.Fail(core.E(mainCmdCatalogShow2419de, "rule "+args[0]+" not found", nil))
	}
	return writeIndentedJSON(stdout, rule)
}

func writeReport(writer core.Writer, output string, report lintpkg.Report) core.Result {
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
		return core.Fail(core.E("writeReport", mainUnsupportedOutputFormat785877+output, nil))
	}
}

func writeIndentedJSON(writer core.Writer, value any) core.Result {
	data := core.JSONMarshalIndent(value, "", "  ")
	if !data.OK {
		return data
	}
	return core.WriteString(writer, string(data.Value.([]byte))+"\n")
}

func writeCatalogSummary(writer core.Writer, findings []lintpkg.Finding) core.Result {
	summary := lintpkg.Summarise(findings)
	if written := core.WriteString(writer, core.Sprintf("\n%d finding(s)", summary.Total)); !written.OK {
		return written
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
		parts = append(parts, core.Sprintf("%d %s", count, severity))
	}
	var extraSeverities []string
	for severity := range summary.BySeverity {
		if !seen[severity] {
			extraSeverities = append(extraSeverities, severity)
		}
	}
	sort.Strings(extraSeverities)
	for _, severity := range extraSeverities {
		count := summary.BySeverity[severity]
		if count != 0 {
			parts = append(parts, core.Sprintf("%d %s", count, severity))
		}
	}
	if len(parts) > 0 {
		if written := core.WriteString(writer, " ("+core.Join(", ", parts...)+")"); !written.OK {
			return written
		}
	}
	return core.WriteString(writer, "\n")
}

func parseArgs(args []string) parsedArgs {
	parsed := parsedArgs{flags: make(map[string][]string)}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		key, value, ok := parseFlag(arg)
		if !ok {
			parsed.positionals = append(parsed.positionals, arg)
			continue
		}
		if value == "" && index+1 < len(args) && !core.IsFlag(args[index+1]) {
			value = args[index+1]
			index++
		}
		if value == "" {
			value = "true"
		}
		parsed.flags[key] = append(parsed.flags[key], value)
	}
	return parsed
}

func parseFlag(arg string) (string, string, bool) {
	key, value, ok := core.ParseFlag(arg)
	if !ok {
		return "", "", false
	}
	switch key {
	case "o":
		key = "output"
	case "c":
		key = "config"
	case "l":
		key = "lang"
	case "f":
		key = "format"
	case "s":
		key = "severity"
	}
	return key, value, true
}

func firstFlag(parsed parsedArgs, key string, short string, fallback string) string {
	if values := parsed.flags[key]; len(values) > 0 {
		return values[len(values)-1]
	}
	if short != "" {
		if values := parsed.flags[short]; len(values) > 0 {
			return values[len(values)-1]
		}
	}
	return fallback
}

func boolFlag(parsed parsedArgs, key string, fallback bool) bool {
	values := parsed.flags[key]
	if len(values) == 0 {
		return fallback
	}
	value := values[len(values)-1]
	return value == "true" || value == "1" || value == "yes"
}
