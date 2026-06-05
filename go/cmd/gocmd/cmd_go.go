// Package gocmd provides `core go` — the Go module quality-assurance gate.
//
// `core go qa` runs the Go toolchain directly against the module in the working
// directory: fmt, vet, lint (golangci-lint) and test, with `core go qa full`
// adding race, vuln and security. It composes lint.Toolkit. This is distinct
// from `core qa`, which verifies CI / PR / repo state (and runs the
// multi-language linter adapters); `core go` is the focused Go gate.
//
//	core go qa        # fmt, vet, lint, test
//	core go qa full   # + race, vuln, security
//	core go test      # go test ./...
package gocmd

import (
	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/lint/pkg/lint"
)

// Shared styles from the CLI library, matching `core qa`'s output.
var (
	okStyle   = cli.SuccessStyle
	failStyle = cli.ErrorStyle
	warnStyle = cli.WarningStyle
	dimStyle  = cli.DimStyle
)

// AddGoCommands registers `core go` and its subcommands onto c.
//
//	cli.Main(cli.WithCommands("go", gocmd.AddGoCommands))
func AddGoCommands(c *core.Core) core.Result {
	if r := c.Command("go", core.Command{
		Description: "Go module QA — fmt, vet, lint, test",
	}); !r.OK {
		return r
	}
	for _, register := range []func(*core.Core) core.Result{
		addGoQACommands,
		addGoTestCommand,
	} {
		if r := register(c); !r.OK {
			return r
		}
	}
	return core.Ok(nil)
}

func addGoQACommands(c *core.Core) core.Result {
	if r := c.Command("go/qa", core.Command{
		Description: "Run the Go QA gate: fmt, vet, lint, test",
		Action:      func(core.Options) core.Result { return runGoQA(false) },
	}); !r.OK {
		return r
	}
	return c.Command("go/qa/full", core.Command{
		Description: "Run the Go QA gate plus race, vuln and security",
		Action:      func(core.Options) core.Result { return runGoQA(true) },
	})
}

func addGoTestCommand(c *core.Core) core.Result {
	return c.Command("go/test", core.Command{
		Description: "Run go test ./... for the module in the working directory",
		Action:      func(core.Options) core.Result { return runGoTest() },
	})
}

// goStep is one named check in the QA gate.
type goStep struct {
	name string
	run  func(*lint.Toolkit) stepOutcome
}

// stepOutcome is the result of running a goStep: ok, fail (with detail to
// show), or skip (an optional tool was absent, with the reason).
type stepOutcome struct {
	status string // "ok" | "fail" | "skip"
	detail string
}

func okOutcome() stepOutcome                { return stepOutcome{status: "ok"} }
func failOutcome(detail string) stepOutcome { return stepOutcome{status: "fail", detail: detail} }
func skipOutcome(reason string) stepOutcome { return stepOutcome{status: "skip", detail: reason} }

// runGoQA runs the standard gate (fmt, vet, lint, test); full adds race, vuln
// and security. Every step runs so all problems surface in one pass; the gate
// fails if any step failed. Skipped optional tools never fail the gate.
func runGoQA(full bool) core.Result {
	toolkit := lint.NewToolkit(".")

	steps := []goStep{
		{"fmt", stepFmt},
		{"vet", stepVet},
		{"lint", stepLint},
		{"test", stepTest},
	}
	if full {
		steps = append(steps,
			goStep{"race", stepRace},
			goStep{"vuln", stepVuln},
			goStep{"security", stepSecurity},
		)
	}

	failed := 0
	for _, step := range steps {
		outcome := step.run(toolkit)
		printStep(step.name, outcome)
		if outcome.status == "fail" {
			failed++
		}
	}

	core.Print(core.Stdout(), "\n")
	if failed > 0 {
		core.Print(core.Stdout(), "%s %s\n",
			failStyle.Render("FAIL"),
			core.Sprintf("go qa — %d step(s) failed", failed))
		return core.Fail(core.E("gocmd.runGoQA", core.Sprintf("go qa: %d step(s) failed", failed), nil))
	}
	core.Print(core.Stdout(), "%s %s\n", okStyle.Render("PASS"), "go qa — all checks passed")
	return core.Ok(nil)
}

// runGoTest runs the module's tests on their own.
func runGoTest() core.Result {
	outcome := stepTest(lint.NewToolkit("."))
	printStep("test", outcome)
	if outcome.status == "fail" {
		return core.Fail(core.E("gocmd.runGoTest", "go test failed", nil))
	}
	return core.Ok(nil)
}

func printStep(name string, outcome stepOutcome) {
	switch outcome.status {
	case "ok":
		core.Print(core.Stdout(), "  %s %s\n", okStyle.Render("✓"), name)
	case "skip":
		core.Print(core.Stdout(), "  %s %s %s\n",
			warnStyle.Render("•"), name, dimStyle.Render("(skipped: "+outcome.detail+")"))
	default:
		core.Print(core.Stdout(), "  %s %s\n", failStyle.Render("✗"), name)
		if outcome.detail != "" {
			core.Print(core.Stdout(), "%s\n", indentLines(outcome.detail, "      "))
		}
	}
}

// indentLines prefixes every line of s with prefix, for readable failure output.
func indentLines(s, prefix string) string {
	lines := core.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return core.Join("\n", lines...)
}

// --- steps -----------------------------------------------------------------

// run executes a tool through the toolkit and returns its captured output.
func run(toolkit *lint.Toolkit, name string, args ...string) lint.CommandOutput {
	return toolkit.Run(name, args...).Value.(lint.CommandOutput)
}

// detailOf returns the tool's findings (stdout), falling back to stderr.
func detailOf(out lint.CommandOutput) string {
	if d := core.Trim(out.Stdout); d != "" {
		return d
	}
	return core.Trim(out.Stderr)
}

func stepFmt(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "gofmt", "-l", ".")
	if out.ExitCode == -1 {
		return skipOutcome("gofmt not found")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	if files := core.Trim(out.Stdout); files != "" {
		return failOutcome("unformatted files:\n" + files)
	}
	return okOutcome()
}

func stepVet(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "go", "vet", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("go not found")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}

func stepLint(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "golangci-lint", "run", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("golangci-lint not installed")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}

func stepTest(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "go", "test", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("go not found")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}

func stepRace(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "go", "test", "-race", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("go not found")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}

func stepVuln(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "govulncheck", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("govulncheck not installed")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}

func stepSecurity(toolkit *lint.Toolkit) stepOutcome {
	out := run(toolkit, "gosec", "-quiet", "./...")
	if out.ExitCode == -1 {
		return skipOutcome("gosec not installed")
	}
	if out.ExitCode != 0 {
		return failOutcome(detailOf(out))
	}
	return okOutcome()
}
