# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`core/lint` is a standalone pattern catalog, regex-based code checker, and multi-language QA toolkit. It loads YAML rule definitions and matches them against source files, plus wraps external Go and PHP tooling into structured APIs. Zero framework dependencies — uses `forge.lthn.ai/core/cli` for CLI scaffolding only.

## Build & Development

```bash
core go test          # run all tests
core go test ./pkg/lint/...   # run tests for a specific package
core go qa            # full QA pipeline (vet, lint, test)
core build            # produces ./bin/core-lint
```

Run a single test:
```bash
go test -run TestMatcherExcludePattern ./pkg/lint/
```

## Architecture

Three distinct subsystems share this module:

### 1. Pattern Catalog & Scanner (`pkg/lint/`)

The core lint engine. YAML rules in `catalog/` are embedded at compile time via `//go:embed` in `lint.go` and loaded through `LoadEmbeddedCatalog()`.

**Data flow:** YAML → `ParseRules` → `Catalog` → filter by language/severity → `NewMatcher` (compiles regexes) → `Scanner.ScanDir`/`ScanFile` → `[]Finding` → output as text/JSON/JSONL via `report.go`.

Key types:
- `Rule` — parsed from YAML, validated with `Validate()`. Only `detection: "regex"` rules are matched; other detection types are stored but skipped by `Matcher`.
- `Matcher` — holds pre-compiled `regexp.Regexp` for each rule's `pattern` and optional `exclude_pattern`. Matches line-by-line.
- `Scanner` — walks directory trees, auto-detects language from file extension (`extensionMap`), skips `vendor/node_modules/.git/testdata/.core`.
- `Finding` — a match result with rule ID, file, line, severity, and fix suggestion.

### 2. Go Dev Toolkit (`pkg/lint/tools.go`, `complexity.go`, `coverage.go`, `vulncheck.go`)

Structured Go APIs wrapping external tools (`go vet`, `govulncheck`, `gocyclo`, `gitleaks`, `git`). The `Toolkit` type executes subprocesses and parses their output into typed structs (`ToolFinding`, `Vulnerability`, `CoverageReport`, `RaceCondition`, etc.).

`complexity.go` provides native AST-based cyclomatic complexity analysis (no external tools needed) via `AnalyseComplexity`.

`coverage.go` provides `CoverageStore` for persisting and comparing coverage snapshots over time, detecting regressions.

`vulncheck.go` parses `govulncheck -json` NDJSON output into `VulnFinding` structs.

### 3. PHP QA Toolchain (`pkg/php/`, `pkg/detect/`, `cmd/qa/`)

Wraps PHP ecosystem tools (Pint, PHPStan/Larastan, Psalm, Rector, Infection, PHPUnit/Pest, composer audit). `pkg/detect/` identifies project type by filesystem markers (go.mod → Go, composer.json → PHP).

`pkg/php/pipeline.go` defines a staged QA pipeline: quick (audit, fmt, stan) → standard (+psalm, test) → full (+rector, infection).

`pkg/php/runner.go` builds `process.RunSpec` entries with dependency ordering (`After` field) for the `core/go-process` runner.

### CLI Entry Points

- `cmd/core-lint/main.go` — `core-lint lint check` and `core-lint lint catalog` commands
- `cmd/qa/` — `core qa` subcommands registered via `init()` → `cli.RegisterCommands`. Go-focused (watch, review, health, issues, docblock) and PHP-focused (fmt, stan, psalm, audit, security, rector, infection, test).

## Rule Schema

Each YAML file in `catalog/` contains an array of rules:

```yaml
- id: go-sec-001            # unique identifier
  title: "..."              # human-readable title
  severity: high            # info | low | medium | high | critical
  languages: [go]           # file extensions mapped via extensionMap
  tags: [security]          # free-form tags
  pattern: 'regex'          # Go regexp syntax
  exclude_pattern: 'regex'  # optional, skips matching lines/files
  fix: "..."                # suggested fix text
  detection: regex          # only "regex" is actively matched
  auto_fixable: false
  example_bad: '...'
  example_good: '...'
```

Rules are validated on load — `Validate()` checks required fields and compiles regex patterns.

## Coding Standards

- UK English (e.g. `Analyse`, `Summarise`, `Colour`)
- All functions have typed params/returns
- Tests use `testify` (assert/require)
- Licence: EUPL-1.2
