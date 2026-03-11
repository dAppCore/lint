---
title: core/lint
description: Pattern catalog, regex-based code checker, and quality assurance toolkit for Go and PHP projects
---

# core/lint

`forge.lthn.ai/core/lint` is a standalone pattern catalog and code quality toolkit. It ships a YAML-based rule catalog for detecting security issues, correctness bugs, and modernisation opportunities in Go source code. It also provides a full PHP quality assurance pipeline and a suite of developer tooling wrappers.

The library is designed to be embedded into other tools. The YAML rule files are compiled into the binary at build time via `go:embed`, so there are no runtime file dependencies.

## Module Path

```
forge.lthn.ai/core/lint
```

Requires Go 1.26+.

## Quick Start

### As a Library

```go
import (
    lint "forge.lthn.ai/core/lint"
    lintpkg "forge.lthn.ai/core/lint/pkg/lint"
)

// Load the embedded rule catalog.
cat, err := lint.LoadEmbeddedCatalog()
if err != nil {
    log.Fatal(err)
}

// Filter rules for Go, severity medium and above.
rules := cat.ForLanguage("go")
filtered := (&lintpkg.Catalog{Rules: rules}).AtSeverity("medium")

// Create a scanner and scan a directory.
scanner, err := lintpkg.NewScanner(filtered)
if err != nil {
    log.Fatal(err)
}

findings, err := scanner.ScanDir("./src")
if err != nil {
    log.Fatal(err)
}

// Output results.
lintpkg.WriteText(os.Stdout, findings)
```

### As a CLI

```bash
# Build the binary
core build          # produces ./bin/core-lint

# Scan the current directory with all rules
core-lint lint check

# Scan with filters
core-lint lint check --lang go --severity high ./pkg/...

# Output as JSON
core-lint lint check --format json .

# Browse the catalog
core-lint lint catalog list
core-lint lint catalog list --lang go
core-lint lint catalog show go-sec-001
```

### QA Commands

The `qa` command group provides workflow-level quality assurance:

```bash
# Go-focused
core qa watch              # Monitor GitHub Actions after a push
core qa review             # PR review status with actionable next steps
core qa health             # Aggregate CI health across all repos
core qa issues             # Intelligent issue triage
core qa docblock           # Check Go docblock coverage

# PHP-focused
core qa fmt                # Format PHP code with Laravel Pint
core qa stan               # Run PHPStan/Larastan static analysis
core qa psalm              # Run Psalm static analysis
core qa audit              # Audit composer and npm dependencies
core qa security           # Security checks (.env, filesystem, deps)
core qa rector             # Automated code refactoring
core qa infection          # Mutation testing
core qa test               # Run Pest or PHPUnit tests
```

## Package Layout

| Package | Path | Description |
|---------|------|-------------|
| `lint` (root) | `lint.go` | Embeds YAML catalogs and exposes `LoadEmbeddedCatalog()` |
| `pkg/lint` | `pkg/lint/` | Core library: Rule, Catalog, Matcher, Scanner, Report, Complexity, Coverage, VulnCheck, Toolkit |
| `pkg/detect` | `pkg/detect/` | Project type detection (Go, PHP) by filesystem markers |
| `pkg/php` | `pkg/php/` | PHP quality tools: format, analyse, audit, security, refactor, mutation, test, pipeline, runner |
| `cmd/core-lint` | `cmd/core-lint/` | CLI binary (`core-lint lint check`, `core-lint lint catalog`) |
| `cmd/qa` | `cmd/qa/` | QA workflow commands (watch, review, health, issues, docblock, PHP tools) |
| `catalog/` | `catalog/` | YAML rule definitions (embedded at compile time) |

## Rule Catalogs

Three built-in YAML catalogs ship with the module:

| File | Rules | Focus |
|------|-------|-------|
| `go-security.yaml` | 6 | SQL injection, path traversal, XSS, timing attacks, log injection, secret leaks |
| `go-correctness.yaml` | 7 | Unsynchronised goroutines, silent error swallowing, panics in library code, file deletion |
| `go-modernise.yaml` | 5 | Replace legacy patterns with modern stdlib (`slices.Clone`, `slices.Sort`, `maps.Keys`, `errgroup`) |

Total: **18 rules** across 3 severity tiers (info, medium, high, critical). All rules target Go. The catalog is extensible -- add more YAML files to `catalog/` and they will be embedded automatically.

## Dependencies

Direct dependencies:

| Module | Purpose |
|--------|---------|
| `forge.lthn.ai/core/cli` | CLI framework (`cli.Main()`, command registration, TUI styles) |
| `forge.lthn.ai/core/go-i18n` | Internationalisation for CLI strings |
| `forge.lthn.ai/core/go-io` | Filesystem abstraction for registry loading |
| `forge.lthn.ai/core/go-log` | Structured logging and error wrapping |
| `forge.lthn.ai/core/go-scm` | Repository registry (`repos.yaml`) for multi-repo commands |
| `github.com/stretchr/testify` | Test assertions |
| `gopkg.in/yaml.v3` | YAML parsing for rule catalogs |

The `pkg/lint` sub-package has minimal dependencies (only `gopkg.in/yaml.v3` and standard library). The heavier CLI and SCM dependencies live in `cmd/`.

## Licence

EUPL-1.2
