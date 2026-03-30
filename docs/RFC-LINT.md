# core/lint RFC — Linter Orchestration & QA Gate

> Pure linter orchestration — no AI. Runs tools, outputs structured JSON.
> Usable in dispatch QA, GitHub CI, and local dev. Zero API keys required.
> An agent should be able to implement any component from this document alone.

**Module:** `dappco.re/go/lint`
**Repository:** `dappco.re/go/lint`
**Binary:** `core-lint`
**Config:** `.core/lint.yaml` (per-repo) or `agents.yaml` (fleet-wide defaults)

---

## 1. Overview

core/lint detects languages in a project, runs every matching linter, and aggregates results into a single structured JSON report. No AI, no network calls, no API keys — pure static analysis.

Three consumers:

| Consumer | How it runs | Purpose |
|----------|------------|---------|
| core/agent dispatch | `core lint run` in QA step | Gate agent output before PR |
| GitHub Actions CI | `core lint run --ci` | PR check gate on public repos |
| Developer local | `core lint run` | Pre-commit validation |

Same binary, same config, same output format everywhere.

---

## 2. Configuration

### 2.1 Per-Repo Config (`.core/lint.yaml`)

```yaml
lint:
  # Language-specific linters
  go:
    - golangci-lint
    - gosec
    - govulncheck
    - staticcheck
    - revive
    - errcheck
  php:
    - phpstan
    - psalm
    - phpcs
    - phpmd
    - pint
  js:
    - biome
    - oxlint
    - eslint
    - prettier
  ts:
    - biome
    - oxlint
    - typescript
  python:
    - ruff
    - mypy
    - bandit
    - pylint

  # Infrastructure linters (language-independent)
  infra:
    - shellcheck
    - hadolint
    - yamllint
    - jsonlint
    - markdownlint

  # Security scanners
  security:
    - gitleaks
    - trivy
    - gosec
    - bandit
    - semgrep

  # Compliance
  compliance:
    - syft
    - grype
    - scancode

# Output format
output: json          # json, text, github (annotations)

# Fail threshold
fail_on: error        # error, warning, info

# Paths to scan (default: .)
paths:
  - .

# Paths to exclude
exclude:
  - vendor/
  - node_modules/
  - .core/
```

### 2.2 Language Detection

If no `.core/lint.yaml` exists, detect languages from files present and run all available linters for those languages:

```go
// Detect project languages from file extensions and markers
//
//   langs := lint.Detect(".")  // ["go", "php", "yaml", "dockerfile"]
func Detect(path string) []string { }
```

| Marker | Language |
|--------|----------|
| `go.mod` | go |
| `composer.json` | php |
| `package.json` | js/ts |
| `tsconfig.json` | ts |
| `requirements.txt`, `pyproject.toml` | python |
| `Cargo.toml` | rust |
| `Dockerfile*` | dockerfile |
| `*.sh` | shell |
| `*.yaml`, `*.yml` | yaml |

### 2.3 Tool Discovery

If a tool is not installed, skip it gracefully. Never fail because a linter is missing — report it as skipped in the output. Each adapter implements `Available() bool` on the Adapter interface — typically checks if the binary is in PATH via `c.Process()`.

---

## 3. Execution Pipeline

### 3.1 Three Stages

```
Stage 1: Static — lint source files
  → run language linters + infra linters on source
  → structured findings per file

Stage 2: Build — compile and capture errors
  → go build, composer install, npm run build, tsc
  → build errors with file:line:column

Stage 3: Artifact — scan compiled output
  → security scanners on binaries, images, bundles
  → SBOM generation, vulnerability matching
```

Each stage produces findings in the same format. Stages are independent — a build failure in Stage 2 does not prevent Stage 3 from running on whatever artifacts exist.

### 3.2 Execution Model

The three-stage pipeline is a Core Task — declarative orchestration:

```go
func (s *Service) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    // Pipeline as a Task — stages are Steps
    c.Task("lint/pipeline", core.Task{
        Steps: []core.Step{
            {Action: "lint.static"},
            {Action: "lint.build"},
            {Action: "lint.artifact"},
        },
    })

    return core.Result{OK: true}
}
```

### 3.3 Core Accessor Usage

Every Core accessor is used — lint is a full Core citizen:

```go
// handleRun is the action handler for lint.run
// Actions accept core.Options per the ActionHandler contract. The handler unmarshals to a typed DTO.
//
//   result := c.Action("lint.run").Run(ctx, c, core.Options{"path": ".", "output": "json"})
func (s *Service) handleRun(ctx context.Context, opts core.Options) core.Result {
    input := RunInput{
        Path:     opts.String("path"),
        Output:   opts.String("output"),
        FailOn:   opts.String("fail_on"),
        Category: opts.String("category"),
        Lang:     opts.String("lang"),
        Hook:     opts.Bool("hook"),
        SBOM:     opts.Bool("sbom"),
    }
    c := s.Core()
    fs := c.Fs()                         // filesystem — read configs, scan files
    proc := c.Process()                  // run external linters as managed processes
    cfg := c.Config()                    // load .core/lint.yaml
    log := c.Log()                       // structured logging per linter run

    if input.Path == "" {
        return core.Result{OK: false, Error: core.E("lint.Run", "path is required", nil)}
    }

    // Load config from .core/lint.yaml — determines which linters to run and paths to scan
    var lintConfig LintConfig
    cfg.Get("lint", &lintConfig)

    // Detect languages — config overrides auto-detection if languages are specified
    langs := s.detect(fs, input.Path)
    if input.Lang != "" {
        langs = []string{input.Lang}
    }
    log.Info("lint.run", "languages", langs, "path", input.Path)

    // Broadcast start via IPC
    c.ACTION(LintStarted{
        Path:      input.Path,
        Languages: langs,
        Tools:     len(s.adaptersFor(langs)),
    })

    // Run adapters — each adapter handles its own process execution via c.Process()
    var findings []Finding
    for _, adapter := range s.adaptersFor(langs) {
        if !adapter.Available() {
            log.Warn("lint.skip", "tool", adapter.Name(), "reason", "not installed")
            continue
        }

        result := adapter.Run(ctx, input)
        if result.OK {
            if parsed, ok := result.Value.([]Finding); ok {
                findings = append(findings, parsed...)
            }
        }
    }

    // Broadcast completion via IPC
    report := s.buildReport(input.Path, langs, findings)
    c.ACTION(LintCompleted{
        Path:     input.Path,
        Findings: report.Summary.Total,
        Errors:   report.Summary.Errors,
        Passed:   report.Summary.Passed,
        Duration: report.Duration,
    })

    return core.Result{Value: report, OK: report.Summary.Passed}
}
```

### 3.4 Embedded Defaults

Default rule configs and ignore patterns are embedded via `c.Data()`:

Default configs are loaded via `c.Data()` which reads from the service's embedded assets. The embed directive is on the Data subsystem, not in lint source directly.

```go
// defaultConfigFor returns the default rule config for a linter tool.
// Returns empty string if no default is bundled.
//
//   cfg := s.defaultConfigFor("golangci")  // returns golangci.yml content
func (s *Service) defaultConfigFor(tool string) string {
    r := s.Core().Data().ReadString(core.Sprintf("defaults/%s", tool))
    if r.OK {
        if s, ok := r.Value.(string); ok {
            return s
        }
    }
    return ""
}
```

### 3.5 Entitlements

Premium linters (security scanners, SBOM generators) can be gated behind entitlements:

```go
func (s *Service) adaptersFor(langs []string) []Adapter {
    c := s.Core()
    var adapters []Adapter

    for _, a := range s.registry {
        if a.RequiresEntitlement() && !c.Entitled(a.Entitlement()).Allowed {
            continue
        }
        if a.MatchesLanguage(langs) {
            adapters = append(adapters, a)
        }
    }
    return adapters
}
```

| Tier | Linters | Entitlement |
|------|---------|-------------|
| Free | golangci-lint, staticcheck, revive, errcheck, govulncheck, phpstan, psalm, phpcs, phpmd, pint, biome, oxlint, eslint, ruff, mypy, pylint, shellcheck, hadolint, yamllint, markdownlint, jsonlint | none |
| Pro | gosec, semgrep, bandit, trivy, gitleaks | `lint.security` |
| Enterprise | syft, grype, scancode | `lint.compliance` |

### 3.6 IPC Messages

```go
// Broadcast during lint operations
type LintStarted struct {
    Path      string
    Languages []string
    Tools     int
}

type LintCompleted struct {
    Path     string
    Findings int
    Errors   int
    Passed   bool
    Duration string
}

type FindingsReported struct {
    Tool     string
    Findings int
    Severity string // highest severity found
}
```

Linters run in parallel where possible. Each linter runs via `c.Process()` with a timeout (default 5 minutes per linter). Results are merged into a single report.

---

## 4. Output Format

### 4.1 Report

```go
type Report struct {
    Project    string     `json:"project"`
    Timestamp  time.Time  `json:"timestamp"`
    Duration   string     `json:"duration"`
    Languages  []string   `json:"languages"`
    Tools      []ToolRun  `json:"tools"`
    Findings   []Finding  `json:"findings"`
    Summary    Summary    `json:"summary"`
}

type ToolRun struct {
    Name     string `json:"name"`
    Version  string `json:"version"`
    Status   string `json:"status"`    // passed, failed, skipped, timeout
    Duration string `json:"duration"`
    Findings int    `json:"findings"`
}

type Summary struct {
    Total    int `json:"total"`
    Errors   int `json:"errors"`
    Warnings int `json:"warnings"`
    Info     int `json:"info"`
    Passed   bool `json:"passed"`
}
```

### 4.2 Finding

```go
type Finding struct {
    Tool     string `json:"tool"`      // which linter found this
    File     string `json:"file"`      // relative path
    Line     int    `json:"line"`      // line number (0 if unknown)
    Column   int    `json:"column"`    // column number (0 if unknown)
    Severity string `json:"severity"`  // error, warning, info
    Code     string `json:"code"`      // linter-specific rule code
    Message  string `json:"message"`   // human-readable description
    Category string `json:"category"`  // security, style, correctness, performance
    Fix      string `json:"fix"`       // suggested fix (if linter provides one)
}
```

### 4.3 Output Modes

| Mode | Flag | Use case |
|------|------|----------|
| JSON | `--output json` | Machine consumption, dispatch pipeline, training data |
| Text | `--output text` | Developer terminal |
| GitHub | `--output github` | GitHub Actions annotations (`::error file=...`) |
| SARIF | `--output sarif` | GitHub Code Scanning, IDE integration |

---

## 5. Linter Adapters

Each linter is an adapter implementing a common interface:

```go
// Adapter wraps a linter tool and normalises its output.
// Adapters receive the Core reference at construction — all I/O goes through Core primitives.
//
//   adapter := lint.NewGolangciLint(c)
//   result := adapter.Run(ctx, lint.RunInput{Path: "."})
type Adapter interface {
    Name() string
    Available() bool
    Languages() []string
    Command() string
    Args() []string
    Entitlement() string
    RequiresEntitlement() bool
    MatchesLanguage(langs []string) bool
    Fast() bool
    Run(ctx context.Context, input RunInput) core.Result
    RunFiles(ctx context.Context, files []string) []Finding  // returns nil for whole-project adapters (Fast()=false)
    Parse(output string) []Finding
}
```

### 5.1 Adapter Registry

Adapters are registered in `registerAdapters()` during service startup. Adding a new linter is one file — implement `Adapter`, add the constructor call to `registerAdapters()`, done. No global registry, no init() magic.

### 5.2 Adapter Responsibilities

Each adapter:
1. Checks if the tool binary exists (`Available()`)
2. Runs the tool via `c.Process()` — never `os/exec` directly
3. Reads output via `c.Fs()` — never `os.ReadFile` or `io.ReadAll`
4. Parses the tool-specific JSON into normalised `Finding` structs
5. Maps tool-specific severity levels to `error/warning/info`
6. Maps tool-specific rule codes to categories
7. Uses `core.E()` for errors — never `fmt.Errorf` or `errors.New`
8. Uses `core.Split`, `core.Trim`, `core.JoinPath` — never raw `strings.*` or `path/filepath.*`

### 5.3 Banned Imports

The following stdlib imports are banned in core/lint source code. Core provides wrappers for all of them:

| Banned | Use instead |
|--------|------------|
| `os` | `c.Fs()` |
| `os/exec` | `c.Process()` |
| `fmt` | `core.Sprintf`, `core.Print` |
| `log` | `c.Log()` |
| `errors` | `core.E()` |
| `strings` | `core.Split`, `core.Trim`, `core.Contains`, `core.HasPrefix` |
| `path/filepath` | `core.JoinPath`, `core.PathDir`, `core.PathBase` |
| `encoding/json` | `core.JSON` |
| `io` | `c.Fs()` for file I/O, `c.Process()` for command output |

All replacement primitives (`core.Split`, `core.Trim`, `core.Contains`, `core.HasPrefix`, `core.JoinPath`, `core.PathDir`, `core.PathBase`, `core.Sprintf`, `core.Print`, `core.JSON`) are defined in `code/core/go/RFC.md` § "String Helpers" and "Path Helpers".

### 5.4 Built-in Adapters

| Adapter | Tool | JSON Flag | Categories |
|---------|------|-----------|------------|
| `golangci-lint` | golangci-lint | `--out-format json` | style, correctness, performance |
| `gosec` | gosec | `-fmt json` | security |
| `govulncheck` | govulncheck | `-json` | security |
| `staticcheck` | staticcheck | `-f json` | correctness, performance |
| `revive` | revive | `-formatter json` | style |
| `errcheck` | errcheck | `-` (parse stderr) | correctness |
| `phpstan` | phpstan | `--format json` | correctness |
| `psalm` | psalm | `--output-format json` | correctness |
| `phpcs` | phpcs | `--report=json` | style |
| `phpmd` | phpmd | `json` | style, correctness |
| `biome` | biome | `--reporter json` | style, correctness |
| `oxlint` | oxlint | `--format json` | style, correctness |
| `eslint` | eslint | `--format json` | style, correctness |
| `ruff` | ruff | `--output-format json` | style, correctness |
| `mypy` | mypy | `--output json` | correctness |
| `bandit` | bandit | `-f json` | security |
| `pylint` | pylint | `--output-format json` | style, correctness |
| `shellcheck` | shellcheck | `-f json` | correctness |
| `hadolint` | hadolint | `-f json` | correctness, security |
| `yamllint` | yamllint | `-f parsable` (line-based, parsed by adapter) | style |
| `gitleaks` | gitleaks | `--report-format json` | security |
| `trivy` | trivy | `--format json` | security |
| `semgrep` | semgrep | `--json` | security, correctness |
| `syft` | syft | `-o json` | compliance |
| `grype` | grype | `-o json` | security |
| `scancode` | scancode-toolkit | `--json` | compliance |
| `markdownlint` | markdownlint-cli | `--json` | style |
| `jsonlint` | jsonlint | (exit code + stderr, parsed by adapter) | style |
| `pint` | pint | `--format json` | style |

---

## 6. CLI

```bash
# Run all linters (auto-detect languages)
core lint run

# Run with specific config
core lint run --config .core/lint.yaml

# Run only security linters
core lint run --category security

# Run only for Go
core lint run --lang go

# CI mode (GitHub annotations output, exit 1 on failure)
core lint run --ci

# Pre-commit hook (only changed files, fast, exit 1 on errors)
core lint run --hook

# JSON output to file
core lint run --output json > report.json

# List available linters
core lint tools

# List detected languages
core lint detect

# Generate default config
core lint init

# Install as git pre-commit hook
core lint hook install

# Remove git pre-commit hook
core lint hook remove
```

### 6.1 Pre-Commit Hook Mode

`--hook` mode is optimised for speed in the commit workflow:

1. Only scans files staged for commit (`git diff --cached --name-only`)
2. Skips slow linters (trivy, grype, SBOM — those belong in CI)
3. Runs only Stage 1 (static) — no build or artifact scanning
4. Exits non-zero on errors, zero on warnings-only
5. Text output by default (developer terminal), respects `--output` override

```go
// Hook mode — lint only staged files
//
//   core lint run --hook
func (s *Service) hookMode(ctx context.Context) core.Result {
    c := s.Core()
    proc := c.Process()

    // Get staged files
    result := proc.RunIn(ctx, ".", "git", "diff", "--cached", "--name-only")
    if !result.OK {
        return result
    }
    output, _ := result.Value.(string)
    staged := core.Split(core.Trim(output), "\n")

    // Detect languages from staged files only
    langs := s.detectFromFiles(staged)

    // Run fast linters only (skip security/compliance tier)
    adapters := s.adaptersFor(langs)
    adapters = filterFast(adapters) // exclude slow scanners

    // Lint only staged files
    var findings []Finding
    for _, a := range adapters {
        findings = append(findings, a.RunFiles(ctx, staged)...)
    }

    report := s.buildReport(".", langs, findings)
    return core.Result{Value: report, OK: report.Summary.Errors == 0}
}
```

### 6.2 Scheduled Runs

Lint runs can be registered as scheduled Tasks. The scheduler invokes the same actions — no special scheduling code in lint:

Scheduled lint runs are configured in `.core/lint.yaml`, not in code:

```yaml
# .core/lint.yaml
schedules:
  nightly-security:
    cron: "0 0 * * *"
    categories: [security, compliance]
    output: json

  hourly-quick:
    cron: "0 * * * *"
    categories: [static]
    paths: [.]
    fail_on: error
```

Lint registers Tasks for each schedule entry during startup:

```go
// Register scheduled tasks from config
for name, sched := range lintConfig.Schedules {
    c.Task(core.Sprintf("lint/schedule/%s", name), core.Task{
        Steps: []core.Step{
            {Action: "lint.run"},
        },
    })
}
```

The scheduler subsystem reads cron expressions from config and fires the matching Tasks. Lint doesn't implement scheduling — it registers Tasks that CAN be scheduled. Until the scheduler lands, these Tasks are callable manually via `core lint/run --category security`.

### 6.3 Hook Installation

```bash
# Install — creates .git/hooks/pre-commit
core lint hook install
```

Creates a pre-commit hook that runs `core lint run --hook`:

```bash
#!/bin/sh
# Installed by core-lint
exec core-lint run --hook
```

If a hook already exists, appends to it rather than overwriting. `core lint hook remove` reverses the installation.

---

## 7. Integration Points

### 7.1 core/agent QA Gate

The dispatch pipeline calls `core lint run --output json` as part of the QA step. Findings are parsed and used to determine pass/fail:

```
AgentCompleted
  → core lint run --output json > /tmp/lint-report.json
  → parse report.Summary.Passed
  → if passed: continue to PR
  → if failed: mark workspace as failed, include findings in status
```

### 7.2 GitHub Actions

```yaml
# .github/workflows/lint.yml
name: Lint
on: [pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install core-lint
        run: go install dappco.re/go/lint/cmd/core-lint@latest
      - name: Run linters
        run: core-lint run --ci
```

No AI, no API keys, no secrets. Pure static analysis on the public CI runner.

### 7.3 Training Data Pipeline

Every finding that gets fixed by a Codex dispatch produces a training pair:

```
Input:  finding JSON (tool, file, line, message, code)
Output: git diff that fixed it
```

These pairs are structured for downstream training pipelines. The output format is consistent regardless of consumer.

---

## 8. SBOM Integration

When compliance linters run, SBOM artifacts are generated alongside the lint report:

```bash
# Generate SBOM during lint
core lint compliance

# Output: report.json + sbom.cdx.json (CycloneDX) + sbom.spdx.json (SPDX)
```

SBOM generation uses:
- `syft` for multi-language SBOM
- `cyclonedx-gomod` for Go-specific
- `cdxgen` for JS/TS projects

Vulnerability scanning uses the SBOM:
```
syft → sbom.cdx.json → grype → vulnerability findings
```

---

## 9. Build & Binary

### 9.1 Binary

core-lint builds as a standalone binary. All linter adapters are compiled in — the binary orchestrates external tools, it does not bundle them.

```bash
# Build
go build -o bin/core-lint ./cmd/core-lint/

# Install
go install dappco.re/go/lint/cmd/core-lint@latest
```

The binary expects linter tools to be in PATH. In the core-dev Docker image, all tools are pre-installed. On a developer machine, missing tools are skipped gracefully.

### 9.2 CLI Test Suite (Taskfile)

Tests use Taskfile.yaml as test harnesses. Directory structure maps to CLI commands — the path IS the test:

```
tests/cli/
├── core/
│   └── lint/
│       ├── Taskfile.yaml          ← test `core-lint` (root command)
│       ├── go/
│       │   ├── Taskfile.yaml      ← test `core-lint go`
│       │   └── fixtures/          ← sample Go files with known issues
│       ├── php/
│       │   ├── Taskfile.yaml      ← test `core-lint php`
│       │   └── fixtures/
│       ├── js/
│       │   ├── Taskfile.yaml      ← test `core-lint js`
│       │   └── fixtures/
│       ├── python/
│       │   ├── Taskfile.yaml      ← test `core-lint python`
│       │   └── fixtures/
│       ├── security/
│       │   ├── Taskfile.yaml      ← test `core-lint security`
│       │   └── fixtures/          ← files with known secrets, vulns
│       ├── compliance/
│       │   ├── Taskfile.yaml      ← test `core-lint compliance`
│       │   └── fixtures/
│       ├── detect/
│       │   ├── Taskfile.yaml      ← test `core-lint detect`
│       │   └── fixtures/          ← mixed-language projects
│       ├── tools/
│       │   └── Taskfile.yaml      ← test `core-lint tools`
│       ├── init/
│       │   └── Taskfile.yaml      ← test `core-lint init`
│       └── run/
│           ├── Taskfile.yaml      ← test `core-lint run` (full pipeline)
│           └── fixtures/
```

### 9.3 Test Pattern

Each Taskfile runs core-lint against fixtures with known issues, captures JSON output, and validates the report:

```yaml
# tests/cli/core/lint/go/Taskfile.yaml
version: '3'

tasks:
  test:
    desc: Test core-lint go command
    cmds:
      - core-lint go --output json fixtures/ > /tmp/lint-go-report.json
      - |
        # Verify expected findings exist
        jq -e '.findings | length > 0' /tmp/lint-go-report.json
        jq -e '.findings[] | select(.tool == "golangci-lint")' /tmp/lint-go-report.json
        jq -e '.summary.errors > 0' /tmp/lint-go-report.json

  test-clean:
    desc: Test core-lint go on clean code (should pass)
    cmds:
      - core-lint go --output json fixtures/clean/ > /tmp/lint-go-clean.json
      - jq -e '.summary.passed == true' /tmp/lint-go-clean.json

  test-missing-tool:
    desc: Test graceful skip when linter not installed
    cmds:
      - PATH=/usr/bin core-lint go --output json fixtures/ > /tmp/lint-go-skip.json
      - jq -e '.tools[] | select(.status == "skipped")' /tmp/lint-go-skip.json
```

### 9.4 Fixtures

Each language directory has fixtures with known issues for deterministic testing:

```
fixtures/
├── bad_imports.go          ← imports "fmt" (banned)
├── missing_error_check.go  ← unchecked error return
├── insecure_random.go      ← math/rand instead of crypto/rand
└── clean/
    └── good.go             ← passes all linters
```

Security fixtures contain planted secrets and known-vulnerable dependencies:

```
fixtures/
├── leaked_key.go           ← contains AWS_SECRET_ACCESS_KEY pattern
├── go.mod                  ← depends on package with known CVE
└── Dockerfile              ← runs as root, no healthcheck
```

### 9.5 CI Integration Test

The top-level Taskfile runs all sub-tests:

```yaml
# tests/cli/core/lint/Taskfile.yaml
version: '3'

tasks:
  test:
    desc: Run all core-lint CLI tests
    cmds:
      - task -d detect test
      - task -d tools test
      - task -d go test
      - task -d php test
      - task -d js test
      - task -d python test
      - task -d security test
      - task -d compliance test
      - task -d run test

  test-report:
    desc: Run full pipeline and validate report structure
    cmds:
      - core-lint run --output json fixtures/mixed/ > /tmp/lint-full-report.json
      - |
        # Validate report structure
        jq -e '.project' /tmp/lint-full-report.json
        jq -e '.timestamp' /tmp/lint-full-report.json
        jq -e '.languages | length > 0' /tmp/lint-full-report.json
        jq -e '.tools | length > 0' /tmp/lint-full-report.json
        jq -e '.findings | length > 0' /tmp/lint-full-report.json
        jq -e '.summary.total > 0' /tmp/lint-full-report.json
```

---

## 10. Core Service Registration

### 10.1 Service

core/lint registers as a Core service exposing linter orchestration via IPC actions:

```go
// Service is the lint orchestrator. It holds the adapter registry and runs linters via Core primitives.
type Service struct {
    *core.ServiceRuntime[Options]
    registry []Adapter   // registered linter adapters
}

// Register the lint service with Core
//
//   c := core.New(
//       core.WithService(lint.Register),
//   )
func Register(c *core.Core) core.Result {
    svc := &Service{
        ServiceRuntime: core.NewServiceRuntime(c, Options{}),
    }
    svc.registerAdapters(c)
    return core.Result{Value: svc, OK: true}
}

// registerAdapters populates the adapter registry with all built-in linters.
// Each adapter receives the Core reference for process execution and filesystem access.
func (s *Service) registerAdapters(c *core.Core) {
    s.registry = []Adapter{
        NewGolangciLint(c), NewGosec(c), NewGovulncheck(c), NewStaticcheck(c),
        NewRevive(c), NewErrcheck(c),
        NewPHPStan(c), NewPsalm(c), NewPHPCS(c), NewPHPMD(c), NewPint(c),
        NewBiome(c), NewOxlint(c), NewESLint(c),
        NewRuff(c), NewMypy(c), NewBandit(c), NewPylint(c),
        NewShellcheck(c), NewHadolint(c), NewYamllint(c),
        NewGitleaks(c), NewTrivy(c), NewSemgrep(c),
        NewSyft(c), NewGrype(c), NewScancode(c),
        NewMarkdownlint(c), NewJsonlint(c),
    }
}

// Helper functions used by the orchestrator:

// adaptersFor returns adapters matching the detected languages, filtered by entitlements.
func (s *Service) adaptersFor(langs []string) []Adapter { }

// detect returns languages found in the project at the given path.
func (s *Service) detect(fs core.Fs, path string) []string { }

// detectFromFiles returns languages based on a list of file paths (used in hook mode).
func (s *Service) detectFromFiles(files []string) []string { }

// buildReport assembles a Report from path, languages, and collected findings.
func (s *Service) buildReport(path string, langs []string, findings []Finding) Report { }

// filterFast removes slow adapters for hook mode.
// Uses Adapter.Fast() — adapters self-declare whether they are suitable for pre-commit.
// Fast = Stage 1 only linters that operate on individual files (not whole-project scanners).
// govulncheck, trivy, syft, grype, scancode, semgrep return Fast()=false.
func filterFast(adapters []Adapter) []Adapter { }


func (s *Service) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    // Pipeline stage actions (used by lint.pipeline Task)
    c.Action("lint.static", s.handleStatic)
    c.Action("lint.build", s.handleBuild)
    c.Action("lint.artifact", s.handleArtifact)

    // Orchestration actions
    c.Action("lint.run", s.handleRun)
    c.Action("lint.detect", s.handleDetect)
    c.Action("lint.tools", s.handleTools)

    // Per-language actions
    c.Action("lint.go", s.handleGo)
    c.Action("lint.php", s.handlePHP)
    c.Action("lint.js", s.handleJS)
    c.Action("lint.python", s.handlePython)
    c.Action("lint.security", s.handleSecurity)
    c.Action("lint.compliance", s.handleCompliance)

    // CLI commands — each calls the matching action with DTO constructed from flags
    c.Command("lint", core.Command{Description: "Run linters on project code"})
    c.Command("lint/run", core.Command{Description: "Run all configured linters", Action: s.cmdRun})
    c.Command("lint/detect", core.Command{Description: "Detect project languages", Action: s.cmdDetect})
    c.Command("lint/tools", core.Command{Description: "List available linters", Action: s.cmdTools})
    c.Command("lint/init", core.Command{Description: "Generate default .core/lint.yaml", Action: s.cmdInit})
    c.Command("lint/go", core.Command{Description: "Run Go linters", Action: s.cmdGo})
    c.Command("lint/php", core.Command{Description: "Run PHP linters", Action: s.cmdPHP})
    c.Command("lint/js", core.Command{Description: "Run JS/TS linters", Action: s.cmdJS})
    c.Command("lint/python", core.Command{Description: "Run Python linters", Action: s.cmdPython})
    c.Command("lint/security", core.Command{Description: "Run security scanners", Action: s.cmdSecurity})
    c.Command("lint/compliance", core.Command{Description: "Run compliance scanners", Action: s.cmdCompliance})
    c.Command("lint/hook/install", core.Command{Description: "Install git pre-commit hook", Action: s.cmdHookInstall})
    c.Command("lint/hook/remove", core.Command{Description: "Remove git pre-commit hook", Action: s.cmdHookRemove})

    // Pipeline task — three stages, orchestrated declaratively
    c.Task("lint/pipeline", core.Task{
        Steps: []core.Step{
            {Action: "lint.static"},
            {Action: "lint.build"},
            {Action: "lint.artifact"},
        },
    })

    return core.Result{OK: true}
}
```

### 10.2 Input DTOs

Actions accept typed DTOs, not named props:

```go
// RunInput is the DTO for lint.run, lint.go, lint.php, etc.
//
//   lint.RunInput{Path: ".", Output: "json", FailOn: "error"}
type RunInput struct {
    Path     string   `json:"path"`               // project path to scan
    Output   string   `json:"output,omitempty"`    // json, text, github, sarif
    Config   string   `json:"config,omitempty"`    // path to .core/lint.yaml
    FailOn   string   `json:"fail_on,omitempty"`   // error, warning, info
    Category string   `json:"category,omitempty"`  // security, compliance, static
    Lang     string   `json:"lang,omitempty"`      // go, php, js, python
    Hook     bool     `json:"hook,omitempty"`       // pre-commit mode
    Files    []string `json:"files,omitempty"`      // specific files to lint
    SBOM     bool     `json:"sbom,omitempty"`       // generate SBOM alongside report
}

// ToolInfo describes an available linter
//
//   info := lint.ToolInfo{Name: "golangci-lint", Available: true, Languages: []string{"go"}}
type ToolInfo struct {
    Name        string   `json:"name"`
    Available   bool     `json:"available"`
    Languages   []string `json:"languages"`
    Category    string   `json:"category"`    // style, correctness, security, compliance
    Entitlement string   `json:"entitlement"` // empty if free tier
}

// DetectInput is the DTO for lint.detect
//
//   lint.DetectInput{Path: "."}
type DetectInput struct {
    Path string `json:"path"`
}
```

### 10.3 IPC Actions

Actions are the public interface. CLI, MCP, and API are surfaces that construct the DTO and call the action:

```go
// Any Core service can request linting via IPC
//
//   result := c.Action("lint.run").Run(ctx, c, core.Options{"path": repoDir, "output": "json"})
//   report, _ := result.Value.(lint.Report)
```

| Action | Input DTO | Returns |
|--------|-----------|---------|
| `lint.run` | RunInput | Report (full pipeline) |
| `lint.detect` | DetectInput | []string (languages) |
| `lint.tools` | (none) | []ToolInfo (available linters) |
| `lint.go` | RunInput | Report (Go linters only) |
| `lint.php` | RunInput | Report (PHP linters only) |
| `lint.js` | RunInput | Report (JS/TS linters only) |
| `lint.python` | RunInput | Report (Python linters only) |
| `lint.security` | RunInput | Report (security scanners only) |
| `lint.compliance` | RunInput | Report (SBOM + compliance only) |

CLI commands construct the DTO from flags:

```go
func (s *Service) cmdRun(ctx context.Context, opts core.Options) core.Result {
    // CLI commands call the action handler directly — same signature
    return s.handleRun(ctx, opts)
}
```

MCP tools construct the DTO from tool parameters. Same action, same DTO, different surface.

### 10.4 MCP Tool Exposure (core-agent plugin)

When loaded into core-agent, lint actions become MCP tools. Claude and Codex can lint code from within a session:

```
claude/lint/
├── SKILL.md            ← "Run linters on the current workspace"
└── commands/
    ├── run.md          ← /lint:run
    ├── go.md           ← /lint:go
    ├── security.md     ← /lint:security
    └── compliance.md   ← /lint:compliance
```

MCP tool registration is handled by core-agent (see `code/core/agent/RFC.md`), not by core/lint. core/lint exposes named Actions — the agent MCP subsystem wraps those Actions as MCP tools. core/lint does not know about MCP.
```

This means:
- **I** (Claude) can run `lint_run` on any workspace via MCP to check code quality
- **Codex** agents inside Docker get `core-lint` binary for QA gates
- **Developers** get the same `core lint` CLI locally
- **GitHub Actions** get `core-lint run --ci` for PR checks

Same adapters, same output format, four surfaces.

### 10.5 core/agent QA Integration

The agent dispatch pipeline loads lint as a service and calls it during QA:

The agent QA handler calls `lint.run` via action and uses the returned `lint.Report` to determine pass/fail:

```go
result := c.Action("lint.run").Run(ctx, c, core.Options{
    "path":    repoDir,
    "output":  "json",
    "fail_on": "error",
})
report, _ := result.Value.(lint.Report)
```

See `code/core/agent/RFC.md` § "Completion Pipeline" for the QA handler. core/lint returns `core.Result{Value: lint.Report{...}}` — the consumer decides what to do with it.

---

## 11. Reference Material

| Resource | Location |
|----------|----------|
| Core framework | `code/core/go/RFC.md` |
| Agent pipeline | `code/core/agent/RFC.md` § "Completion Pipeline" |
| Build system | `code/core/go/build/RFC.md` |

---

## Changelog

- 2026-03-30: Initial RFC — linter orchestration, adapter pattern, three-stage pipeline, SBOM, CI integration, training data pipeline, Taskfile CLI test suite, fixtures, Core service registration, IPC actions, MCP tool exposure, agent QA integration
