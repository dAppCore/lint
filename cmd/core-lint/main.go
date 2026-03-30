package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"forge.lthn.ai/core/cli/pkg/cli"
	coreerr "forge.lthn.ai/core/go-log"
	lint "forge.lthn.ai/core/lint"
	lintpkg "forge.lthn.ai/core/lint/pkg/lint"
)

func main() {
	cli.WithAppName("core-lint")
	cli.Main(cli.WithCommands("lint", addLintCommands))
}

func addLintCommands(root *cli.Command) {
	lintCmd := cli.NewGroup("lint", "Pattern-based code linter", "")

	// ── check ──────────────────────────────────────────────────────────────
	var (
		checkFormat   string
		checkLang     string
		checkSeverity string
	)

	checkCmd := cli.NewCommand("check", "Scan files for pattern matches", "", func(cmd *cli.Command, args []string) error {
		cat, err := lint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.check", "loading catalog", err)
		}

		rules := cat.Rules

		// Filter by language if specified.
		if checkLang != "" {
			rules = cat.ForLanguage(checkLang)
			if len(rules) == 0 {
				fmt.Fprintf(os.Stderr, "no rules for language %q\n", checkLang)
				return nil
			}
		}

		// Filter by severity threshold if specified.
		if checkSeverity != "" {
			filtered := (&lintpkg.Catalog{Rules: rules}).AtSeverity(checkSeverity)
			if len(filtered) == 0 {
				fmt.Fprintf(os.Stderr, "no rules at severity %q or above\n", checkSeverity)
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

		var allFindings []lintpkg.Finding
		for _, p := range paths {
			info, err := os.Stat(p)
			if err != nil {
				return coreerr.E("cmd.check", "stat "+p, err)
			}

			var findings []lintpkg.Finding
			if info.IsDir() {
				findings, err = scanner.ScanDir(p)
			} else {
				findings, err = scanner.ScanFile(p)
			}
			if err != nil {
				return err
			}
			allFindings = append(allFindings, findings...)
		}

		switch checkFormat {
		case "json":
			return lintpkg.WriteJSON(os.Stdout, allFindings)
		case "jsonl":
			return lintpkg.WriteJSONL(os.Stdout, allFindings)
		default:
			lintpkg.WriteText(os.Stdout, allFindings)
		}

		if checkFormat == "text" && len(allFindings) > 0 {
			summary := lintpkg.Summarise(allFindings)
			fmt.Fprintf(os.Stdout, "\n%d finding(s)", summary.Total)

			orderedSeverities := []string{"critical", "high", "medium", "low", "info"}
			seen := make(map[string]bool, len(summary.BySeverity))
			var parts []string

			for _, sev := range orderedSeverities {
				count := summary.BySeverity[sev]
				if count == 0 {
					continue
				}
				seen[sev] = true
				parts = append(parts, fmt.Sprintf("%d %s", count, sev))
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
				fmt.Fprintf(os.Stdout, " (%s)", strings.Join(parts, ", "))
			}
			fmt.Fprintln(os.Stdout)
		}

		return nil
	})

	cli.StringFlag(checkCmd, &checkFormat, "format", "f", "text", "Output format: text, json, jsonl")
	cli.StringFlag(checkCmd, &checkLang, "lang", "l", "", "Filter rules by language (e.g. go, php, ts)")
	cli.StringFlag(checkCmd, &checkSeverity, "severity", "s", "", "Minimum severity threshold (info, low, medium, high, critical)")

	// ── catalog ────────────────────────────────────────────────────────────
	catalogCmd := cli.NewGroup("catalog", "Browse the pattern catalog", "")

	// catalog list
	var listLang string

	listCmd := cli.NewCommand("list", "List all rules in the catalog", "", func(cmd *cli.Command, args []string) error {
		cat, err := lint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.catalog.list", "loading catalog", err)
		}

		rules := cat.Rules
		if listLang != "" {
			rules = cat.ForLanguage(listLang)
		}

		if len(rules) == 0 {
			fmt.Println("No rules found.")
			return nil
		}

		rules = append([]lintpkg.Rule(nil), rules...)
		sort.Slice(rules, func(i, j int) bool {
			if rules[i].Severity == rules[j].Severity {
				return strings.Compare(rules[i].ID, rules[j].ID) < 0
			}
			return strings.Compare(rules[i].Severity, rules[j].Severity) < 0
		})

		for _, r := range rules {
			fmt.Printf("%-14s [%-8s] %s\n", r.ID, r.Severity, r.Title)
		}
		fmt.Fprintf(os.Stderr, "\n%d rule(s)\n", len(rules))
		return nil
	})

	cli.StringFlag(listCmd, &listLang, "lang", "l", "", "Filter by language")

	// catalog show
	showCmd := cli.NewCommand("show", "Show details of a specific rule", "", func(cmd *cli.Command, args []string) error {
		if len(args) == 0 {
			return coreerr.E("cmd.catalog.show", "rule ID required", nil)
		}

		cat, err := lint.LoadEmbeddedCatalog()
		if err != nil {
			return coreerr.E("cmd.catalog.show", "loading catalog", err)
		}

		r := cat.ByID(args[0])
		if r == nil {
			return coreerr.E("cmd.catalog.show", "rule "+args[0]+" not found", nil)
		}

		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	})

	catalogCmd.AddCommand(listCmd, showCmd)
	lintCmd.AddCommand(checkCmd, catalogCmd)
	root.AddCommand(lintCmd)
}
