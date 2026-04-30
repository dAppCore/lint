// Package qa provides quality assurance workflow commands.
//
// Unlike `core dev` which is about doing work (commit, push, pull),
// `core qa` is about verifying work (CI status, reviews, issues).
//
// Commands:
//   - watch: Monitor GitHub Actions after a push, report actionable data
//   - review: PR review status with actionable next steps
//   - health: Aggregate CI health across all repos
//   - issues: Intelligent issue triage
package qa

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/lint/locales"
)

var qaCore = newQACore()

func init() {
	cli.RegisterCommands(func(c *core.Core) {
		result := AddQACommands(c)
		if !result.OK {
			core.Warn("qa command registration failed", "err", result.Error())
		}
	})
}

// Style aliases from shared package
var (
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	warningStyle = cli.WarningStyle
	dimStyle     = cli.DimStyle
)

// AddQACommands registers the 'qa' command and all subcommands.
func AddQACommands(c *core.Core) core.Result {
	if r := c.Command("qa", core.Command{Description: qaText("cmd.qa.long")}); !r.OK {
		return r
	}

	// Go-focused subcommands
	for _, register := range []func(*core.Core) core.Result{
		addWatchCommand,
		addReviewCommand,
		addHealthCommand,
		addIssuesCommand,
		addDocblockCommand,
		addPHPCommands,
	} {
		if r := register(c); !r.OK {
			return r
		}
	}
	return core.Ok(nil)
}

func registerQACommand(c *core.Core, path string, description string, action func() core.Result) core.Result {
	return c.Command(path, core.Command{
		Description: description,
		Action: func(core.Options) core.Result {
			return action()
		},
	})
}

func qaFailure(err error) core.Result {
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

type qaCommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func runQACommand(ctx context.Context, name string, args ...string) qaCommandOutput {
	_ = ctx
	stdout := core.NewBuffer()
	stderr := core.NewBuffer()
	path := name
	if found := findQAExecutable(name); found.OK {
		path = found.Value.(string)
	}
	cmd := &core.Cmd{
		Path:   path,
		Args:   append([]string{name}, args...),
		Stdout: stdout,
		Stderr: stderr,
	}
	err := cmd.Run()
	return qaCommandOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: qaCommandExitCode(err),
		Err:      err,
	}
}

func findQAExecutable(name string) core.Result {
	if name == "" {
		return core.Fail(core.E("findQAExecutable", "empty executable name", nil))
	}
	for _, dir := range core.Split(core.Getenv("PATH"), string(core.PathListSeparator)) {
		if dir == "" {
			dir = "."
		}
		candidate := core.PathJoin(dir, name)
		stat := core.Stat(candidate)
		if stat.OK {
			info := stat.Value.(core.FsFileInfo)
			if !info.IsDir() && info.Mode()&0111 != 0 {
				return core.Ok(candidate)
			}
		}
	}
	return core.Fail(core.E("findQAExecutable", core.Sprintf("%s was not found in PATH", name), nil))
}

func qaCommandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func newQACore() *core.Core {
	c := core.New()
	svc, err := i18n.NewWithFS(locales.FS, ".")
	if err == nil {
		c.I18n().SetTranslator(svc)
	}
	return c
}

func qaText(key string, args ...any) string {
	r := qaCore.I18n().Translate(key, args...)
	if !r.OK {
		return key
	}
	text, ok := r.Value.(string)
	if !ok {
		return key
	}
	return text
}

func qaLabel(key string) string {
	switch key {
	case "repo":
		return "Repo"
	default:
		return qaText(core.Concat("common.label.", key))
	}
}
