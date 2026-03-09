# CLAUDE.md

## Project Overview

`core/lint` is a standalone pattern catalog and regex-based code checker. It loads YAML rule definitions and matches them against source files. Zero framework dependencies.

## Build & Development

```bash
core go test
core go qa
core build          # produces ./bin/core-lint
```

## Architecture

- `catalog/` — YAML rule files (embedded at compile time)
- `pkg/lint/` — Library: Rule, Catalog, Matcher, Scanner, Report types
- `cmd/core-lint/` — CLI binary using `cli.Main()`

## Rule Schema

Each YAML file contains an array of rules with: id, title, severity, languages, tags, pattern (regex), exclude_pattern, fix, example_bad, example_good, detection type.

## Coding Standards

- UK English
- All functions have typed params/returns
- Tests use testify
- License: EUPL-1.2
