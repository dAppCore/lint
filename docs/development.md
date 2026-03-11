---
title: Development Guide
description: How to build, test, and contribute to core/lint
---

# Development Guide

## Prerequisites

- Go 1.26 or later
- `core` CLI (for build and QA commands)
- `gh` CLI (only needed for the `qa watch`, `qa review`, `qa health`, and `qa issues` commands)

## Building

The project uses the `core` build system. Configuration lives in `.core/build.yaml`.

```bash
# Build the binary (outputs to ./bin/core-lint)
core build

# Build targets: linux/amd64, linux/arm64, darwin/arm64, windows/amd64
# CGO is disabled; the binary is fully static.
```

To build manually with `go build`:

```bash
go build -trimpath -ldflags="-s -w" -o bin/core-lint ./cmd/core-lint
```

## Running Tests

```bash
# Run all tests
core go test

# Run a single test by name
core go test --run TestRule_Validate_Good

# Generate coverage report
core go cov
core go cov --open    # Opens HTML report in browser
```

The test suite covers all packages:

| Package | Test count | Focus |
|---------|-----------|-------|
| `pkg/lint` | ~89 | Rule validation, catalog loading, matcher, scanner, report, complexity, coverage, vulncheck, toolkit |
| `pkg/detect` | 6 | Project type detection |
| `pkg/php` | ~125 | All PHP tool wrappers (format, analyse, audit, security, refactor, mutation, test, pipeline, runner) |

Tests follow the `_Good`, `_Bad`, `_Ugly` suffix convention:
- `_Good` -- happy path
- `_Bad` -- expected error conditions
- `_Ugly` -- edge cases and panics

### Test Examples

Testing rules against source content:

```go
func TestMatcher_Match_Good(t *testing.T) {
    rules := []Rule{
        {
            ID:        "test-001",
            Title:     "TODO found",
            Severity:  "low",
            Pattern:   `TODO`,
            Detection: "regex",
        },
    }
    m, err := NewMatcher(rules)
    require.NoError(t, err)

    findings := m.Match("example.go", []byte("// TODO: fix this"))
    assert.Len(t, findings, 1)
    assert.Equal(t, "test-001", findings[0].RuleID)
    assert.Equal(t, 1, findings[0].Line)
}
```

Testing complexity analysis without file I/O:

```go
func TestAnalyseComplexitySource_Good(t *testing.T) {
    src := `package example
func simple() { if true {} }
func complex() {
    if a {} else if b {} else if c {}
    for i := range items {
        switch {
        case x: if y {}
        case z:
        }
    }
}`
    results, err := AnalyseComplexitySource(src, "test.go", 3)
    require.NoError(t, err)
    assert.NotEmpty(t, results)
}
```

## Quality Assurance

```bash
# Full QA pipeline: format, vet, lint, test
core go qa

# Extended QA: includes race detection, vulnerability scan, security checks
core go qa full

# Individual checks
core go fmt        # Format code
core go vet        # Run go vet
core go lint       # Run linter
```

## Project Structure

```
lint/
├── .core/
│   └── build.yaml          # Build configuration
├── bin/                     # Build output (gitignored)
├── catalog/
│   ├── go-correctness.yaml  # Correctness rules (7 rules)
│   ├── go-modernise.yaml    # Modernisation rules (5 rules)
│   └── go-security.yaml     # Security rules (6 rules)
├── cmd/
│   ├── core-lint/
│   │   └── main.go          # CLI binary entry point
│   └── qa/
│       ├── cmd_qa.go         # QA command group registration
│       ├── cmd_watch.go      # GitHub Actions monitoring
│       ├── cmd_review.go     # PR review status
│       ├── cmd_health.go     # Aggregate CI health
│       ├── cmd_issues.go     # Issue triage
│       ├── cmd_docblock.go   # Docblock coverage
│       └── cmd_php.go        # PHP QA subcommands
├── pkg/
│   ├── detect/
│   │   ├── detect.go         # Project type detection
│   │   └── detect_test.go
│   ├── lint/
│   │   ├── catalog.go        # Catalog loading and querying
│   │   ├── complexity.go     # Cyclomatic complexity (native AST)
│   │   ├── coverage.go       # Coverage tracking and comparison
│   │   ├── matcher.go        # Regex matching engine
│   │   ├── report.go         # Output formatters (text, JSON, JSONL)
│   │   ├── rule.go           # Rule type and validation
│   │   ├── scanner.go        # Directory walking and file scanning
│   │   ├── tools.go          # Toolkit (subprocess wrappers)
│   │   ├── vulncheck.go      # govulncheck JSON parser
│   │   ├── testdata/
│   │   │   └── catalog/
│   │   │       └── test-rules.yaml
│   │   └── *_test.go
│   └── php/
│       ├── analyse.go         # PHPStan/Larastan/Psalm wrappers
│       ├── audit.go           # Composer audit + npm audit
│       ├── format.go          # Laravel Pint wrapper
│       ├── mutation.go        # Infection wrapper
│       ├── pipeline.go        # QA stage definitions
│       ├── refactor.go        # Rector wrapper
│       ├── runner.go          # Process spec builder
│       ├── security.go        # Security checks (.env, filesystem)
│       ├── test.go            # Pest/PHPUnit wrapper
│       └── *_test.go
├── lint.go                    # Root package: embedded catalog loader
├── go.mod
├── go.sum
├── CLAUDE.md
└── README.md
```

## Writing New Rules

### Rule Schema

Each YAML file in `catalog/` contains an array of rule objects. Required fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (convention: `{lang}-{category}-{number}`, e.g., `go-sec-001`) |
| `title` | string | Short human-readable description |
| `severity` | string | One of: `info`, `low`, `medium`, `high`, `critical` |
| `languages` | []string | Target languages (e.g., `[go]`, `[go, php]`) |
| `pattern` | string | Detection pattern (regex for `detection: regex`) |
| `fix` | string | Remediation guidance |
| `detection` | string | Detection type (currently only `regex`) |

Optional fields:

| Field | Type | Description |
|-------|------|-------------|
| `tags` | []string | Categorisation tags (e.g., `[security, injection]`) |
| `exclude_pattern` | string | Regex to suppress false positives |
| `found_in` | []string | Repos where the pattern was originally observed |
| `example_bad` | string | Code example that triggers the rule |
| `example_good` | string | Corrected code example |
| `first_seen` | string | Date the pattern was first catalogued |
| `auto_fixable` | bool | Whether automated fixing is feasible |

### Naming Convention

Rule IDs follow the pattern `{lang}-{category}-{number}`:

- `go-sec-*` -- Security rules
- `go-cor-*` -- Correctness rules
- `go-mod-*` -- Modernisation rules

### Testing a New Rule

Create a test that verifies the pattern matches expected code and does not match exclusions:

```go
func TestNewRule_Matches(t *testing.T) {
    rules := []Rule{
        {
            ID:             "go-xxx-001",
            Title:          "My new rule",
            Severity:       "medium",
            Languages:      []string{"go"},
            Pattern:        `my-pattern`,
            ExcludePattern: `safe-variant`,
            Detection:      "regex",
        },
    }

    m, err := NewMatcher(rules)
    require.NoError(t, err)

    // Should match
    findings := m.Match("example.go", []byte("code with my-pattern here"))
    assert.Len(t, findings, 1)

    // Should not match (exclusion)
    findings = m.Match("example.go", []byte("code with safe-variant here"))
    assert.Empty(t, findings)
}
```

## Adding PHP Tool Support

To add support for a new PHP tool:

1. Create a new file in `pkg/php/` (e.g., `newtool.go`).
2. Add a detection function that checks for config files or vendor binaries.
3. Add an options struct and an execution function.
4. Add a command in `cmd/qa/cmd_php.go` that wires the tool to the CLI.
5. Add the tool to the pipeline stages in `pipeline.go` if appropriate.
6. Write tests in a corresponding `*_test.go` file.

Follow the existing pattern -- each tool module exports:
- `Detect*()` -- returns whether the tool is available
- `Run*()` or the tool function -- executes the tool with options
- A `*Options` struct -- configures behaviour

## Coding Standards

- **UK English** throughout: `colour`, `organisation`, `centre`, `modernise`, `analyse`, `serialise`
- **Strict typing**: All function parameters and return values must have explicit types
- **Testing**: Use `testify` assertions (`assert`, `require`)
- **Error wrapping**: Use `fmt.Errorf("context: %w", err)` for error chains
- **Formatting**: Standard Go formatting via `gofmt` / `core go fmt`

## Licence

This project is licenced under the EUPL-1.2.
