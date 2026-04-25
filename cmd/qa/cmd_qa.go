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
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/lint/locales"
)

func init() {
	cli.RegisterCommands(AddQACommands, locales.FS)
}

// Style aliases from shared package
var (
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	warningStyle = cli.WarningStyle
	dimStyle     = cli.DimStyle
)

// AddQACommands registers the 'qa' command and all subcommands.
func AddQACommands(root *cli.Command) {
	qaCmd := &cli.Command{
		Use:   "qa",
		Short: i18n.T("cmd.qa.short"),
		Long:  i18n.T("cmd.qa.long"),
	}
	root.AddCommand(qaCmd)

	// Go-focused subcommands
	addWatchCommand(qaCmd)
	addReviewCommand(qaCmd)
	addHealthCommand(qaCmd)
	addIssuesCommand(qaCmd)
	addDocblockCommand(qaCmd)

	// PHP subcommands
	addPHPCommands(qaCmd)
}
