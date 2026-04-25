// cmd_review.go implements the 'qa review' command for PR review status.
//
// Usage:
//   core qa review              # Show all PRs needing attention
//   core qa review --mine       # Show status of your open PRs
//   core qa review --requested  # Show PRs you need to review

package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/log"
)

// Review command flags
var (
	reviewMine      bool
	reviewRequested bool
	reviewRepo      string
	reviewJSON      bool
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number         int                `json:"number"`
	Title          string             `json:"title"`
	Author         Author             `json:"author"`
	State          string             `json:"state"`
	IsDraft        bool               `json:"isDraft"`
	Mergeable      string             `json:"mergeable"`
	ReviewDecision string             `json:"reviewDecision"`
	URL            string             `json:"url"`
	HeadRefName    string             `json:"headRefName"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
	Additions      int                `json:"additions"`
	Deletions      int                `json:"deletions"`
	ChangedFiles   int                `json:"changedFiles"`
	StatusChecks   *StatusCheckRollup `json:"statusCheckRollup"`
	ReviewRequests ReviewRequests     `json:"reviewRequests"`
	Reviews        []Review           `json:"reviews"`
}

// Author represents a GitHub user
type Author struct {
	Login string `json:"login"`
}

// StatusCheckRollup contains CI check status
type StatusCheckRollup struct {
	Contexts []StatusContext `json:"contexts"`
}

// StatusContext represents a single check
type StatusContext struct {
	State      string `json:"state"`
	Conclusion string `json:"conclusion"`
	Name       string `json:"name"`
}

// ReviewRequests contains pending review requests
type ReviewRequests struct {
	Nodes []ReviewRequest `json:"nodes"`
}

// ReviewRequest represents a review request
type ReviewRequest struct {
	RequestedReviewer Author `json:"requestedReviewer"`
}

// Review represents a PR review
type Review struct {
	Author Author `json:"author"`
	State  string `json:"state"`
}

// ReviewFetchError captures a partial fetch failure while preserving any
// successfully fetched PRs in the same review run.
type ReviewFetchError struct {
	Repo  string `json:"repo"`
	Scope string `json:"scope"`
	Error string `json:"error"`
}

type reviewOutput struct {
	Mine             []PullRequest      `json:"mine"`
	Requested        []PullRequest      `json:"requested"`
	TotalMine        int                `json:"total_mine"`
	TotalRequested   int                `json:"total_requested"`
	ShowingMine      bool               `json:"showing_mine"`
	ShowingRequested bool               `json:"showing_requested"`
	FetchErrors      []ReviewFetchError `json:"fetch_errors"`
}

// addReviewCommand adds the 'review' subcommand to the qa command.
func addReviewCommand(parent *cli.Command) {
	reviewCmd := &cli.Command{
		Use:   "review",
		Short: i18n.T("cmd.qa.review.short"),
		Long:  i18n.T("cmd.qa.review.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			return runReview()
		},
	}

	reviewCmd.Flags().BoolVarP(&reviewMine, "mine", "m", false, i18n.T("cmd.qa.review.flag.mine"))
	reviewCmd.Flags().BoolVarP(&reviewRequested, "requested", "r", false, i18n.T("cmd.qa.review.flag.requested"))
	reviewCmd.Flags().StringVar(&reviewRepo, "repo", "", i18n.T("cmd.qa.review.flag.repo"))
	reviewCmd.Flags().BoolVar(&reviewJSON, "json", false, i18n.T("common.flag.json"))

	parent.AddCommand(reviewCmd)
}

func runReview() error {
	// Check gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return log.E("qa.review", i18n.T("error.gh_not_found"), nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Determine repo
	repoFullName := reviewRepo
	if repoFullName == "" {
		var err error
		repoFullName, err = detectRepoFromGit()
		if err != nil {
			return log.E("qa.review", i18n.T("cmd.qa.review.error.no_repo"), nil)
		}
	}

	// Default: show both mine and requested if neither flag is set
	showMine := reviewMine || (!reviewMine && !reviewRequested)
	showRequested := reviewRequested || (!reviewMine && !reviewRequested)
	minePRs := []PullRequest{}
	requestedPRs := []PullRequest{}
	fetchErrors := make([]ReviewFetchError, 0)
	mineFetched := false
	requestedFetched := false
	successfulFetches := 0

	if showMine {
		prs, err := fetchPRs(ctx, repoFullName, "author:@me")
		if err != nil {
			fetchErrors = append(fetchErrors, ReviewFetchError{
				Repo:  repoFullName,
				Scope: "mine",
				Error: strings.TrimSpace(err.Error()),
			})
			if !reviewJSON {
				cli.Warnf("failed to fetch your PRs for %s: %s", repoFullName, strings.TrimSpace(err.Error()))
			}
		} else {
			sort.Slice(prs, func(i, j int) bool {
				if prs[i].Number == prs[j].Number {
					return strings.Compare(prs[i].Title, prs[j].Title) < 0
				}
				return prs[i].Number < prs[j].Number
			})
			minePRs = prs
			mineFetched = true
			successfulFetches++
		}
	}

	if showRequested {
		prs, err := fetchPRs(ctx, repoFullName, "review-requested:@me")
		if err != nil {
			fetchErrors = append(fetchErrors, ReviewFetchError{
				Repo:  repoFullName,
				Scope: "requested",
				Error: strings.TrimSpace(err.Error()),
			})
			if !reviewJSON {
				cli.Warnf("failed to fetch review requested PRs for %s: %s", repoFullName, strings.TrimSpace(err.Error()))
			}
		} else {
			sort.Slice(prs, func(i, j int) bool {
				if prs[i].Number == prs[j].Number {
					return strings.Compare(prs[i].Title, prs[j].Title) < 0
				}
				return prs[i].Number < prs[j].Number
			})
			requestedPRs = prs
			requestedFetched = true
			successfulFetches++
		}
	}

	output := reviewOutput{
		Mine:             minePRs,
		Requested:        requestedPRs,
		TotalMine:        len(minePRs),
		TotalRequested:   len(requestedPRs),
		ShowingMine:      showMine,
		ShowingRequested: showRequested,
		FetchErrors:      fetchErrors,
	}

	if reviewJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		cli.Print("%s\n", string(data))
		if successfulFetches == 0 && len(fetchErrors) > 0 {
			return cli.Err("failed to fetch pull requests for %s", repoFullName)
		}
		return nil
	}

	if successfulFetches == 0 && len(fetchErrors) > 0 {
		return cli.Err("failed to fetch pull requests for %s", repoFullName)
	}

	if showMine && mineFetched {
		if err := printMyPRs(minePRs); err != nil {
			return err
		}
	}

	if showRequested && requestedFetched {
		if showMine && mineFetched {
			cli.Blank()
		}
		if err := printRequestedPRs(requestedPRs); err != nil {
			return err
		}
	}

	return nil
}

// printMyPRs shows the user's open PRs with status
func printMyPRs(prs []PullRequest) error {
	if len(prs) == 0 {
		cli.Print("%s\n", dimStyle.Render(i18n.T("cmd.qa.review.no_prs")))
		return nil
	}

	cli.Print("%s (%d):\n", i18n.T("cmd.qa.review.your_prs"), len(prs))

	for _, pr := range prs {
		printPRStatus(pr)
	}

	return nil
}

// printRequestedPRs shows PRs where user's review is requested
func printRequestedPRs(prs []PullRequest) error {
	if len(prs) == 0 {
		cli.Print("%s\n", dimStyle.Render(i18n.T("cmd.qa.review.no_reviews")))
		return nil
	}

	cli.Print("%s (%d):\n", i18n.T("cmd.qa.review.review_requested"), len(prs))

	for _, pr := range prs {
		printPRForReview(pr)
	}

	return nil
}

// fetchPRs fetches PRs matching the search query
func fetchPRs(ctx context.Context, repo, search string) ([]PullRequest, error) {
	args := []string{
		"pr", "list",
		"--state", "open",
		"--search", search,
		"--json", "number,title,author,state,isDraft,mergeable,reviewDecision,url,headRefName,createdAt,updatedAt,additions,deletions,changedFiles,statusCheckRollup,reviewRequests,reviews",
	}

	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, log.E("qa.fetchPRs", strings.TrimSpace(string(exitErr.Stderr)), nil)
		}
		return nil, err
	}

	var prs []PullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, err
	}

	return prs, nil
}

// printPRStatus prints a PR with its merge status
func printPRStatus(pr PullRequest) {
	// Determine status icon and color
	status, style, action := analyzePRStatus(pr)

	cli.Print("  %s #%d %s\n",
		style.Render(status),
		pr.Number,
		truncate(pr.Title, 50))

	if action != "" {
		cli.Print("      %s %s\n", dimStyle.Render("->"), action)
	}
}

// printPRForReview prints a PR that needs review
func printPRForReview(pr PullRequest) {
	// Show PR info with stats
	stats := fmt.Sprintf("+%d/-%d, %d files",
		pr.Additions, pr.Deletions, pr.ChangedFiles)

	cli.Print("  %s #%d %s\n",
		warningStyle.Render("◯"),
		pr.Number,
		truncate(pr.Title, 50))
	cli.Print("      %s @%s, %s\n",
		dimStyle.Render("->"),
		pr.Author.Login,
		stats)
	cli.Print("      %s gh pr checkout %d\n",
		dimStyle.Render("->"),
		pr.Number)
}

// analyzePRStatus determines the status, style, and action for a PR
func analyzePRStatus(pr PullRequest) (status string, style *cli.AnsiStyle, action string) {
	// Check if draft
	if pr.IsDraft {
		return "◯", dimStyle, "Draft - convert to ready when done"
	}

	// Check CI status
	ciPassed := true
	ciFailed := false
	ciPending := false
	var failedChecks []string

	if pr.StatusChecks != nil {
		for _, check := range pr.StatusChecks.Contexts {
			switch check.Conclusion {
			case "FAILURE", "failure":
				ciFailed = true
				ciPassed = false
				failedChecks = append(failedChecks, check.Name)
			case "PENDING", "pending", "":
				if check.State == "PENDING" || check.State == "" {
					ciPending = true
					ciPassed = false
				}
			}
		}
	}

	// Check review status
	approved := pr.ReviewDecision == "APPROVED"
	changesRequested := pr.ReviewDecision == "CHANGES_REQUESTED"

	// Check mergeable status
	hasConflicts := pr.Mergeable == "CONFLICTING"

	// Determine overall status
	if hasConflicts {
		return "✗", errorStyle, "Needs rebase - has merge conflicts"
	}

	if ciFailed {
		if len(failedChecks) > 0 {
			sort.Strings(failedChecks)
			return "✗", errorStyle, fmt.Sprintf("CI failed: %s", failedChecks[0])
		}
		return "✗", errorStyle, "CI failed"
	}

	if changesRequested {
		return "✗", warningStyle, "Changes requested - address review feedback"
	}

	if ciPending {
		return "◯", warningStyle, "CI running..."
	}

	if !approved && pr.ReviewDecision != "" {
		return "◯", warningStyle, "Awaiting review"
	}

	if approved && ciPassed {
		return "✓", successStyle, "Ready to merge"
	}

	return "◯", dimStyle, ""
}

// truncate shortens a string to max length (rune-safe for UTF-8)
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
