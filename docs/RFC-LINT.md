# RFC-LINT: core/lint Agent-Native CLI and Adapter Contract

- Status: Implemented
- Date: 2026-03-30
- Applies to: `forge.lthn.ai/core/lint`
- Standard: [`docs/RFC-CORE-008-AGENT-EXPERIENCE.md`](./RFC-CORE-008-AGENT-EXPERIENCE.md)

## Abstract

`core/lint` is a standalone Go CLI and library that detects project languages, runs matching lint adapters, merges their findings into one report, and writes machine-readable output for local development, CI, and agent QA.

The binary does not bundle external linters. It orchestrates tools already present in `PATH`, treats missing tools as `skipped`, and always returns a structured report.

This RFC describes the implementation that exists in this repository. It replaces the earlier draft that described a future Core service with Tasks, IPC actions, MCP wrapping, build stages, artifact stages, entitlement gates, and scheduled runs. Those designs are not the current contract.

## AX Decisions

This RFC follows the Agent Experience standard directly:

1. Predictable names: `RunInput`, `Report`, `ToolRun`, `ToolInfo`, `Service`, `Adapter`.
2. Comments as usage examples: command examples use real flags and real paths.
3. Path is documentation: the implementation map is the contract.
4. One input shape for orchestration: `pkg/lint/service.go` owns `RunInput`.
5. One output shape for orchestration: `pkg/lint/service.go` owns `Report`.
6. Stable sequencing over hidden magic: adapters run in deterministic order and reports are sorted before output.

## Path Map

An agent should be able to navigate the module from the path alone:

| Path | Meaning |
|------|---------|
| `cmd/core-lint/main.go` | CLI surface for `run`, `detect`, `tools`, `init`, language shortcuts, `hook`, and the legacy `lint` namespace |
| `pkg/lint/service.go` | Orchestrator for config loading, language selection, adapter selection, hook mode, and report assembly |
| `pkg/lint/adapter.go` | Adapter interface, built-in registry, external command execution, and output parsers |
| `pkg/lint/config.go` | Repo-local config contract and defaults for `core-lint init` |
| `pkg/lint/detect_project.go` | Project language detection from markers and file names |
| `pkg/lint/report.go` | `Summary` aggregation and JSON/text/GitHub/SARIF writers |
| `lint.go` | Embedded catalog loader for `lint check` and `lint catalog` |
| `catalog/*.yaml` | Embedded pattern catalog files used by the legacy catalog commands |
| `tests/cli/lint/...` | CLI artifact tests; the path is the command |

## Scope

In scope:

- Project language detection
- Config-driven lint tool selection
- Embedded catalog scanning
- External linter orchestration
- Structured report generation
- Git pre-commit hook installation and removal
- CLI artifact tests in `tests/cli/lint/...`

Out of scope:

- Core service registration
- IPC or MCP exposure
- Build-stage compilation checks
- Artifact-stage scans against compiled binaries or images
- Scheduler integration
- Sidecar SBOM file writing
- Automatic tool installation
- Entitlement enforcement

## Command Surface

The repository ships two CLI surfaces:

- The root AX surface: `core-lint run`, `core-lint detect`, `core-lint tools`, and friends
- The legacy catalog surface: `core-lint lint check` and `core-lint lint catalog ...`

The RFC commands are mounted twice: once at the root and once under `core-lint lint ...`. Both surfaces are real. The root surface is shorter. The namespaced surface keeps the path semantic.

| Capability | Root path | Namespaced alias | Example |
|------------|-----------|------------------|---------|
| Full orchestration | `core-lint run [path]` | `core-lint lint run [path]` | `core-lint run --output json .` |
| Go only | `core-lint go [path]` | `core-lint lint go [path]` | `core-lint go .` |
| PHP only | `core-lint php [path]` | `core-lint lint php [path]` | `core-lint php .` |
| JS and TS | `core-lint js [path]` | `core-lint lint js [path]` | `core-lint js .` |
| Python only | `core-lint python [path]` | `core-lint lint python [path]` | `core-lint python .` |
| Security tools only | `core-lint security [path]` | `core-lint lint security [path]` | `core-lint security --ci .` |
| Compliance tools only | `core-lint compliance [path]` | `core-lint lint compliance [path]` | `core-lint compliance --output json .` |
| Language detection | `core-lint detect [path]` | `core-lint lint detect [path]` | `core-lint detect --output json .` |
| Tool inventory | `core-lint tools` | `core-lint lint tools` | `core-lint tools --output json --lang go` |
| Default config | `core-lint init [path]` | `core-lint lint init [path]` | `core-lint init /tmp/project` |
| Pre-commit hook install | `core-lint hook install [path]` | `core-lint lint hook install [path]` | `core-lint hook install .` |
| Pre-commit hook remove | `core-lint hook remove [path]` | `core-lint lint hook remove [path]` | `core-lint hook remove .` |
| Embedded catalog scan | none | `core-lint lint check [path...]` | `core-lint lint check --format json tests/cli/lint/check/fixtures` |
| Embedded catalog list | none | `core-lint lint catalog list` | `core-lint lint catalog list --lang go` |
| Embedded catalog show | none | `core-lint lint catalog show RULE_ID` | `core-lint lint catalog show go-sec-001` |

## RunInput Contract

All orchestration commands resolve into one DTO:

```go
type RunInput struct {
    Path     string   `json:"path"`
    Output   string   `json:"output,omitempty"`
    Config   string   `json:"config,omitempty"`
    FailOn   string   `json:"fail_on,omitempty"`
    Category string   `json:"category,omitempty"`
    Lang     string   `json:"lang,omitempty"`
    Hook     bool     `json:"hook,omitempty"`
    CI       bool     `json:"ci,omitempty"`
    Files    []string `json:"files,omitempty"`
    SBOM     bool     `json:"sbom,omitempty"`
}
```

### Input Resolution Rules

`Service.Run()` resolves input in this order:

1. Empty `Path` becomes `.`
2. `CI=true` sets `Output=github` only when `Output` was not provided explicitly
3. Config is loaded from `--config` or `.core/lint.yaml`
4. Empty `FailOn` falls back to the loaded config
5. `Hook=true` with no explicit `Files` reads staged files from `git diff --cached --name-only`
6. `Lang` overrides auto-detection
7. `Files` override directory detection for language inference

### Category and Language Precedence

Tool group selection is intentionally simple and deterministic:

1. `Category=security` means only `lint.security`
2. `Category=compliance` means only `lint.compliance`
3. `Lang=go|php|js|ts|python|...` means only that language group
4. Plain `run` uses all detected language groups plus `infra`
5. Plain `run --ci` adds the `security` group
6. Plain `run --sbom` adds the `compliance` group

`Lang` is stronger than `CI` and `SBOM`. If `Lang` is set, the language group wins and the extra groups are not appended.

## Config Contract

Repo-local config lives at `.core/lint.yaml`.

`core-lint init /path/to/project` writes the default file from `pkg/lint/config.go`.

```yaml
lint:
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
  infra:
    - shellcheck
    - hadolint
    - yamllint
    - jsonlint
    - markdownlint
  security:
    - gitleaks
    - trivy
    - gosec
    - bandit
    - semgrep
  compliance:
    - syft
    - grype
    - scancode

output: json
fail_on: error
paths:
  - .
exclude:
  - vendor/
  - node_modules/
  - .core/
```

### Config Rules

- If `.core/lint.yaml` does not exist, `DefaultConfig()` is used in memory
- Relative `--config` paths resolve relative to `Path`
- Unknown tool names in config are inert; the adapter registry is authoritative
- The current default config includes `prettier`, but the adapter registry does not yet provide a `prettier` adapter

## Detection Contract

`pkg/lint/detect_project.go` is the only project-language detector used by orchestration commands.

### Marker Files

| Marker | Language |
|--------|----------|
| `go.mod` | `go` |
| `composer.json` | `php` |
| `package.json` | `js` |
| `tsconfig.json` | `ts` |
| `requirements.txt` | `python` |
| `pyproject.toml` | `python` |
| `Cargo.toml` | `rust` |
| `Dockerfile*` | `dockerfile` |

### File Extensions

| Extension | Language |
|-----------|----------|
| `.go` | `go` |
| `.php` | `php` |
| `.js`, `.jsx` | `js` |
| `.ts`, `.tsx` | `ts` |
| `.py` | `python` |
| `.rs` | `rust` |
| `.sh` | `shell` |
| `.yaml`, `.yml` | `yaml` |
| `.json` | `json` |
| `.md` | `markdown` |

### Detection Rules

- Directory traversal skips `vendor`, `node_modules`, `.git`, `testdata`, `.core`, and any hidden directory
- Results are de-duplicated and returned in sorted order
- `core-lint detect --output json tests/cli/lint/check/fixtures` currently returns `["go"]`

## Execution Model

`Service.Run()` is the orchestrator. The current implementation is sequential, not parallel.

### Step 1: Load Config

`LoadProjectConfig()` returns the repo-local config or the in-memory default.

### Step 2: Resolve File Scope

- If `Files` was provided, only those files are considered for language detection and adapter arguments
- If `Hook=true` and `Files` is empty, staged files are read from Git
- Otherwise the whole project path is scanned

### Step 3: Resolve Languages

- `Lang` wins first
- `Files` are used next
- `Detect(Path)` is the fallback

### Step 4: Select Adapters

`pkg/lint/service.go` builds a set of enabled tool names from config, then filters the registry from `pkg/lint/adapter.go`.

Special case:

- If `go` is present in the final language set and `Category != "compliance"`, a built-in `catalog` adapter is prepended automatically

### Step 5: Run Adapters

Every selected adapter runs with the same contract:

```go
type Adapter interface {
    Name() string
    Available() bool
    Languages() []string
    Command() string
    Entitlement() string
    RequiresEntitlement() bool
    MatchesLanguage(languages []string) bool
    Category() string
    Fast() bool
    Run(ctx context.Context, input RunInput, files []string) AdapterResult
}
```

Execution rules:

- Missing binaries become `ToolRun{Status: "skipped"}`
- External commands run with a 5 minute timeout
- Hook mode marks non-fast adapters as `skipped`
- Parsed findings are normalised, sorted, and merged into one report
- Adapter order becomes deterministic after `sortToolRuns()` and `sortFindings()`

### Step 6: Compute Pass or Fail

`passesThreshold()` applies the configured threshold:

| `fail_on` | Passes when |
|-----------|-------------|
| `error` or empty | `summary.errors == 0` |
| `warning` | `summary.errors == 0 && summary.warnings == 0` |
| `info` | `summary.total == 0` |

## Catalog Surfaces

The repository has two catalog paths. They are related, but they are not the same implementation.

### Legacy Embedded Catalog

These commands load the embedded YAML catalog via `lint.go`:

- `core-lint lint check`
- `core-lint lint catalog list`
- `core-lint lint catalog show`

The source of truth is `catalog/*.yaml`.

### Orchestration Catalog Adapter

`core-lint run`, `core-lint go`, and the other orchestration commands prepend a smaller built-in `catalog` adapter from `pkg/lint/adapter.go`.

That adapter reads the hard-coded `defaultCatalogRulesYAML` constant, not `catalog/*.yaml`.

Today the fallback adapter contains these Go rules:

- `go-cor-003`
- `go-cor-004`
- `go-sec-001`
- `go-sec-002`
- `go-sec-004`

The overlap is intentional, but the surfaces are different:

- `lint check` returns raw catalog findings with catalog severities such as `medium` or `high`
- `run` normalises those findings into report severities `warning`, `error`, or `info`

An agent must not assume that `core-lint lint check` and `core-lint run` execute the same rule set.

## Adapter Inventory

The registry in `pkg/lint/adapter.go` is the implementation contract.

### Built-in

| Adapter | Languages | Category | Fast | Notes |
|---------|-----------|----------|------|-------|
| `catalog` | `go` | `correctness` | yes | Built-in regex fallback rules |

### Go

| Adapter | Category | Fast |
|---------|----------|------|
| `golangci-lint` | `correctness` | yes |
| `gosec` | `security` | no |
| `govulncheck` | `security` | no |
| `staticcheck` | `correctness` | yes |
| `revive` | `style` | yes |
| `errcheck` | `correctness` | yes |

### PHP

| Adapter | Category | Fast |
|---------|----------|------|
| `phpstan` | `correctness` | yes |
| `psalm` | `correctness` | yes |
| `phpcs` | `style` | yes |
| `phpmd` | `correctness` | yes |
| `pint` | `style` | yes |

### JS and TS

| Adapter | Category | Fast |
|---------|----------|------|
| `biome` | `style` | yes |
| `oxlint` | `style` | yes |
| `eslint` | `style` | yes |
| `typescript` | `correctness` | yes |

### Python

| Adapter | Category | Fast |
|---------|----------|------|
| `ruff` | `style` | yes |
| `mypy` | `correctness` | yes |
| `bandit` | `security` | no |
| `pylint` | `style` | yes |

### Infra and Cross-Project

| Adapter | Category | Fast |
|---------|----------|------|
| `shellcheck` | `correctness` | yes |
| `hadolint` | `security` | yes |
| `yamllint` | `style` | yes |
| `jsonlint` | `style` | yes |
| `markdownlint` | `style` | yes |
| `gitleaks` | `security` | no |
| `trivy` | `security` | no |
| `semgrep` | `security` | no |
| `syft` | `compliance` | no |
| `grype` | `security` | no |
| `scancode` | `compliance` | no |

### Adapter Parsing Rules

- JSON tools are parsed recursively and schema-tolerantly by searching for common keys such as `file`, `line`, `column`, `code`, `message`, and `severity`
- Text tools are parsed from `file:line[:column]: message`
- Unparseable command failures become one synthetic finding with `code: command-failed`
- Duplicate findings are collapsed on `tool|file|line|column|code|message`
- `ToolRun.Version` exists in the report schema but is not populated yet

### Entitlement Metadata

Adapters still expose `Entitlement()` and `RequiresEntitlement()`, but `Service.Run()` does not enforce them today. The metadata is present; the gate is not.

## Output Contract

Orchestration commands return one report document:

```go
type Report struct {
    Project   string    `json:"project"`
    Timestamp time.Time `json:"timestamp"`
    Duration  string    `json:"duration"`
    Languages []string  `json:"languages"`
    Tools     []ToolRun `json:"tools"`
    Findings  []Finding `json:"findings"`
    Summary   Summary   `json:"summary"`
}

type ToolRun struct {
    Name     string `json:"name"`
    Version  string `json:"version,omitempty"`
    Status   string `json:"status"`
    Duration string `json:"duration"`
    Findings int    `json:"findings"`
}

type Summary struct {
    Total      int            `json:"total"`
    Errors     int            `json:"errors"`
    Warnings   int            `json:"warnings"`
    Info       int            `json:"info"`
    Passed     bool           `json:"passed"`
    BySeverity map[string]int `json:"by_severity,omitempty"`
}
```

`Finding` is shared with the legacy catalog scanner:

```go
type Finding struct {
    Tool     string `json:"tool,omitempty"`
    File     string `json:"file"`
    Line     int    `json:"line"`
    Column   int    `json:"column,omitempty"`
    Severity string `json:"severity"`
    Code     string `json:"code,omitempty"`
    Message  string `json:"message,omitempty"`
    Category string `json:"category,omitempty"`
    Fix      string `json:"fix,omitempty"`
    RuleID   string `json:"rule_id,omitempty"`
    Title    string `json:"title,omitempty"`
    Match    string `json:"match,omitempty"`
    Repo     string `json:"repo,omitempty"`
}
```

### Finding Normalisation

During orchestration:

- `Code` falls back to `RuleID`
- `Message` falls back to `Title`
- empty `Tool` becomes `catalog`
- file paths are made relative to `Path` when possible
- severities are collapsed to report levels:

| Raw severity | Report severity |
|--------------|-----------------|
| `critical`, `high`, `error`, `errors` | `error` |
| `medium`, `low`, `warning`, `warn` | `warning` |
| `info`, `note` | `info` |

### Output Modes

| Mode | How to request it | Writer |
|------|-------------------|--------|
| JSON | `--output json` | `WriteReportJSON` |
| Text | `--output text` | `WriteReportText` |
| GitHub annotations | `--output github` or `--ci` | `WriteReportGitHub` |
| SARIF | `--output sarif` | `WriteReportSARIF` |

## Hook Mode

`core-lint run --hook` is the installed pre-commit path.

Implementation details:

- staged files come from `git diff --cached --name-only`
- language detection runs only on those staged files
- adapters with `Fast() == false` are marked `skipped`
- output format still follows normal resolution rules; hook mode does not force text output
- `core-lint hook install` writes a managed block into `.git/hooks/pre-commit`
- `core-lint hook remove` removes only the managed block

Installed hook block:

```sh
# core-lint hook start
# Installed by core-lint
exec core-lint run --hook
# core-lint hook end
```

If the hook file already exists, install appends a guarded block instead of overwriting the file.

## Test Contract

The CLI artifact tests are the runnable contract for this RFC:

| Path | Command under test |
|------|--------------------|
| `tests/cli/lint/check/Taskfile.yaml` | `core-lint lint check` |
| `tests/cli/lint/catalog/list/Taskfile.yaml` | `core-lint lint catalog list` |
| `tests/cli/lint/catalog/show/Taskfile.yaml` | `core-lint lint catalog show` |
| `tests/cli/lint/detect/Taskfile.yaml` | `core-lint detect` |
| `tests/cli/lint/tools/Taskfile.yaml` | `core-lint tools` |
| `tests/cli/lint/init/Taskfile.yaml` | `core-lint init` |
| `tests/cli/lint/run/Taskfile.yaml` | `core-lint run` |
| `tests/cli/lint/Taskfile.yaml` | aggregate CLI suite |

The planted bug fixture is `tests/cli/lint/check/fixtures/input.go`.

Current expectations from the test suite:

- `lint check --format=json` finds `go-cor-003` in `input.go`
- `run --output json --fail-on warning` returns one finding and exits non-zero
- `detect --output json` returns `["go"]` for the shipped fixture
- `tools --output json --lang go` includes `golangci-lint` and `govulncheck`
- `init` writes `.core/lint.yaml`

Unit-level confirmation also exists in:

- `cmd/core-lint/main_test.go`
- `pkg/lint/service_test.go`
- `pkg/lint/detect_project_test.go`

## Explicit Non-Goals

These items are intentionally not part of the current contract:

- no Core runtime integration
- no `core.Task` pipeline
- no `lint.static`, `lint.build`, or `lint.artifact` action graph
- no scheduled cron registration
- no sidecar `sbom.cdx.json` or `sbom.spdx.json` output
- no parallel adapter execution
- no adapter entitlement enforcement
- no guarantee that every config tool name has a matching adapter

Any future RFC that adds those capabilities must describe the code that implements them, not just the aspiration.

## References

- `docs/RFC-CORE-008-AGENT-EXPERIENCE.md`
- `docs/index.md`
- `docs/development.md`
- `cmd/core-lint/main.go`
- `pkg/lint/service.go`
- `pkg/lint/adapter.go`
- `tests/cli/lint/Taskfile.yaml`

## Changelog

- 2026-03-30: Rewrote the RFC to match the implemented standalone CLI, adapter registry, fallback catalog adapter, hook mode, and CLI test paths
