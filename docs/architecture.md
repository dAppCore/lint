---
title: Architecture
description: Internal design of core/lint -- types, data flow, and extension points
---

# Architecture

This document explains how `core/lint` works internally. It covers the core library (`pkg/lint`), the PHP quality pipeline (`pkg/php`), and the QA command layer (`cmd/qa`).

## Overview

The system is organised into three layers:

```
cmd/core-lint     CLI entry point (lint check, lint catalog)
cmd/qa            QA workflow commands (watch, review, health, issues, PHP tools)
   |
pkg/lint          Core library: rules, catalog, matcher, scanner, reporting
pkg/php           PHP tool wrappers: format, analyse, audit, security, test
pkg/detect        Project type detection
   |
catalog/*.yaml    Embedded rule definitions
```

The root `lint.go` file ties the catalog layer to the library:

```go
//go:embed catalog/*.yaml
var catalogFS embed.FS

func LoadEmbeddedCatalog() (*lintpkg.Catalog, error) {
    return lintpkg.LoadFS(catalogFS, "catalog")
}
```

This means all YAML rules are baked into the binary at compile time. There are no runtime file lookups.

## Core Types (pkg/lint)

### Rule

A `Rule` represents a single lint check loaded from YAML. Key fields:

```go
type Rule struct {
    ID             string   `yaml:"id"`
    Title          string   `yaml:"title"`
    Severity       string   `yaml:"severity"`        // info, low, medium, high, critical
    Languages      []string `yaml:"languages"`        // e.g. ["go"], ["go", "php"]
    Tags           []string `yaml:"tags"`             // e.g. ["security", "injection"]
    Pattern        string   `yaml:"pattern"`          // Regex pattern to match
    ExcludePattern string   `yaml:"exclude_pattern"`  // Regex to suppress false positives
    Fix            string   `yaml:"fix"`              // Human-readable remediation
    Detection      string   `yaml:"detection"`        // "regex" (extensible to other types)
    AutoFixable    bool     `yaml:"auto_fixable"`
    ExampleBad     string   `yaml:"example_bad"`
    ExampleGood    string   `yaml:"example_good"`
    FoundIn        []string `yaml:"found_in"`         // Repos where pattern was observed
    FirstSeen      string   `yaml:"first_seen"`
}
```

Each rule validates itself via `Validate()`, which checks required fields and compiles regex patterns. Severity is constrained to five levels: `info`, `low`, `medium`, `high`, `critical`.

### Catalog

A `Catalog` is a flat collection of rules with query methods:

- `ForLanguage(lang)` -- returns rules targeting a specific language
- `AtSeverity(threshold)` -- returns rules at or above a severity level
- `ByID(id)` -- looks up a single rule

Loading is done via `LoadDir(dir)` for filesystem paths or `LoadFS(fsys, dir)` for embedded filesystems. Both read all `.yaml` files in the directory and parse them into `[]Rule`.

### Matcher

The `Matcher` is the regex execution engine. It pre-compiles all regex-detection rules into `compiledRule` structs:

```go
type compiledRule struct {
    rule    Rule
    pattern *regexp.Regexp
    exclude *regexp.Regexp
}
```

`NewMatcher(rules)` compiles patterns once. `Match(filename, content)` then scans line by line:

1. For each compiled rule, check if the filename itself matches the exclude pattern (e.g., skip `_test.go` files).
2. For each line, test against the rule's pattern.
3. If the line matches, check the exclude pattern to suppress false positives.
4. Emit a `Finding` with file, line number, matched text, and remediation advice.

Non-regex detection types are silently skipped, allowing the catalog schema to support future detection mechanisms (AST, semantic) without breaking the matcher.

### Scanner

The `Scanner` orchestrates directory walking and language-aware matching:

1. Walk the directory tree, skipping excluded directories (`vendor`, `node_modules`, `.git`, `testdata`, `.core`).
2. For each file, detect its language from the file extension using `DetectLanguage()`.
3. Filter the rule set to only rules targeting that language.
4. Build a language-scoped `Matcher` and run it against the file content.

Supported language extensions:

| Extension | Language |
|-----------|----------|
| `.go` | go |
| `.php` | php |
| `.ts`, `.tsx` | ts |
| `.js`, `.jsx` | js |
| `.cpp`, `.cc`, `.c`, `.h` | cpp |
| `.py` | py |

### Finding

A `Finding` is the output of a match:

```go
type Finding struct {
    RuleID   string `json:"rule_id"`
    Title    string `json:"title"`
    Severity string `json:"severity"`
    File     string `json:"file"`
    Line     int    `json:"line"`
    Match    string `json:"match"`
    Fix      string `json:"fix"`
    Repo     string `json:"repo,omitempty"`
}
```

### Report

The `report.go` file provides three output formats:

- `WriteText(w, findings)` -- human-readable: `file:line [severity] title (rule-id)`
- `WriteJSON(w, findings)` -- pretty-printed JSON array
- `WriteJSONL(w, findings)` -- newline-delimited JSON (one object per line)

`Summarise(findings)` aggregates counts by severity.

## Data Flow

A typical scan follows this path:

```
YAML files ──> LoadFS() ──> Catalog{Rules}
                                |
                     ForLanguage() / AtSeverity()
                                |
                           []Rule (filtered)
                                |
                          NewScanner(rules)
                                |
                  ScanDir(root) / ScanFile(path)
                                |
                ┌───────────────┼───────────────┐
                │  Walk tree    │  Detect lang   │
                │  Skip dirs    │  Filter rules  │
                │               │  NewMatcher()  │
                │               │  Match()       │
                └───────────────┴───────────────┘
                                |
                          []Finding
                                |
              WriteText() / WriteJSON() / WriteJSONL()
```

## Cyclomatic Complexity Analysis (pkg/lint/complexity.go)

The module includes a native Go AST-based cyclomatic complexity analyser. It uses `go/parser` and `go/ast` -- no external tools required.

```go
results, err := lint.AnalyseComplexity(lint.ComplexityConfig{
    Threshold: 15,
    Path:      "./pkg/...",
})
```

Complexity is calculated by starting at 1 and incrementing for each branching construct:
- `if`, `for`, `range`, `case` (non-default), `comm` (non-default)
- `&&`, `||` binary expressions
- `type switch`, `select`

There is also `AnalyseComplexitySource(src, filename, threshold)` for testing without file I/O.

## Coverage Tracking (pkg/lint/coverage.go)

The coverage subsystem supports:

- **Parsing** Go coverage output (`ParseCoverProfile` for `-coverprofile` format, `ParseCoverOutput` for `-cover` output)
- **Snapshotting** via `CoverageSnapshot` (timestamp, per-package percentages, metadata)
- **Persistence** via `CoverageStore` (JSON file-backed append-only store)
- **Regression detection** via `CompareCoverage(previous, current)` which returns a `CoverageComparison` with regressions, improvements, new packages, and removed packages

## Vulnerability Checking (pkg/lint/vulncheck.go)

`VulnCheck` wraps `govulncheck -json` and parses its newline-delimited JSON output into structured `VulnFinding` objects. The parser handles three message types from govulncheck's wire format:

- `config` -- extracts the module path
- `osv` -- stores vulnerability metadata (ID, aliases, summary, affected ranges)
- `finding` -- maps OSV IDs to call traces and affected packages

## Toolkit (pkg/lint/tools.go)

The `Toolkit` struct wraps common developer commands into structured Go APIs. It executes subprocesses and parses their output:

| Method | Wraps | Returns |
|--------|-------|---------|
| `FindTODOs(dir)` | `git grep` | `[]TODO` |
| `Lint(pkg)` | `go vet` | `[]ToolFinding` |
| `Coverage(pkg)` | `go test -cover` | `[]CoverageReport` |
| `RaceDetect(pkg)` | `go test -race` | `[]RaceCondition` |
| `AuditDeps()` | `govulncheck` (text) | `[]Vulnerability` |
| `ScanSecrets(dir)` | `gitleaks` | `[]SecretLeak` |
| `GocycloComplexity(threshold)` | `gocyclo` | `[]ComplexFunc` |
| `DepGraph(pkg)` | `go mod graph` | `*Graph` |
| `GitLog(n)` | `git log` | `[]Commit` |
| `DiffStat()` | `git diff --stat` | `DiffSummary` |
| `UncommittedFiles()` | `git status` | `[]string` |
| `Build(targets...)` | `go build` | `[]BuildResult` |
| `TestCount(pkg)` | `go test -list` | `int` |
| `CheckPerms(dir)` | `filepath.Walk` | `[]PermIssue` |
| `ModTidy()` | `go mod tidy` | `error` |

All methods use the `Run(name, args...)` helper which captures stdout, stderr, and exit code.

## PHP Quality Pipeline (pkg/php)

The `pkg/php` package provides structured wrappers around PHP ecosystem tools. Each tool has:

1. **Detection** -- checks for config files and vendor binaries (e.g., `DetectAnalyser`, `DetectPsalm`, `DetectRector`)
2. **Options struct** -- configures the tool run
3. **Execution function** -- builds the command, runs it, and returns structured results

### Supported Tools

| Function | Tool | Purpose |
|----------|------|---------|
| `Format()` | Laravel Pint | Code style formatting |
| `Analyse()` | PHPStan / Larastan | Static analysis |
| `RunPsalm()` | Psalm | Type-level static analysis |
| `RunAudit()` | Composer audit + npm audit | Dependency vulnerability scanning |
| `RunSecurityChecks()` | Built-in checks | .env exposure, debug mode, filesystem security |
| `RunRector()` | Rector | Automated code refactoring |
| `RunInfection()` | Infection | Mutation testing |
| `RunTests()` | Pest / PHPUnit | Test execution |

### QA Pipeline

The pipeline system (`pipeline.go` + `runner.go`) organises checks into three stages:

- **Quick** -- audit, fmt, stan (fast, run on every push)
- **Standard** -- psalm (if available), test
- **Full** -- rector, infection (slow, run in full QA)

The `QARunner` builds `process.RunSpec` objects with dependency ordering (e.g., `stan` runs after `fmt`, `test` runs after `stan`). This allows future parallelisation while respecting ordering constraints.

### Project Detection (pkg/detect)

The `detect` package identifies project types by checking for marker files:

- `go.mod` present => Go project
- `composer.json` present => PHP project

`DetectAll(dir)` returns all detected types, enabling polyglot project support.

## QA Command Layer (cmd/qa)

The `cmd/qa` package provides workflow-level commands that integrate with GitHub via the `gh` CLI:

- **watch** -- polls GitHub Actions for a specific commit, shows real-time status, drills into failure details (failed job, step, error line from logs)
- **review** -- fetches open PRs, analyses CI status, review decisions, and merge readiness, suggests next actions
- **health** -- scans all repos in a `repos.yaml` registry, reports aggregate CI health with pass rates
- **issues** -- fetches issues across repos, categorises them (needs response, ready, blocked, triage), prioritises by labels and activity
- **docblock** -- parses Go source with `go/ast`, counts exported symbols with and without doc comments, enforces a coverage threshold

Commands register themselves via `cli.RegisterCommands` in an `init()` function, making them available when the package is imported.

## Extension Points

### Adding New Rules

Create a new YAML file in `catalog/` following the schema:

```yaml
- id: go-xxx-001
  title: "Description of the issue"
  severity: medium             # info, low, medium, high, critical
  languages: [go]
  tags: [security]
  pattern: 'regex-pattern'
  exclude_pattern: 'false-positive-filter'
  fix: "How to fix the issue"
  detection: regex
  auto_fixable: false
  example_bad: 'problematic code'
  example_good: 'corrected code'
```

The file will be embedded automatically on the next build.

### Adding New Detection Types

The `Detection` field on `Rule` currently supports `"regex"`. The `Matcher` skips non-regex rules, so adding a new detection type (e.g., `"ast"` for Go AST patterns) requires:

1. Adding the new type to the `Validate()` method
2. Creating a new matcher implementation
3. Integrating it into `Scanner.ScanDir()`

### Loading External Catalogs

Use `LoadDir(path)` to load rules from a directory on disk rather than the embedded catalog:

```go
cat, err := lintpkg.LoadDir("/path/to/custom/rules")
```

This allows organisations to maintain private rule sets alongside the built-in catalog.
