// cmd_review.go implements the 'qa review' command for PR review status.
//
// Usage:
//   core qa review              # Show all PRs needing attention
//   core qa review --mine       # Show status of your open PRs
//   core qa review --requested  # Show PRs you need to review

package qa

import (
	"context"
	"io"
	"os"
	"sort"
	"time"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/log"
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
	if result := (core.App{}).Find("gh", "GitHub CLI"); !result.OK {
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
				Error: core.Trim(err.Error()),
			})
			if !reviewJSON {
				cli.Warnf("failed to fetch your PRs for %s: %s", repoFullName, core.Trim(err.Error()))
			}
		} else {
			sort.Slice(prs, func(i, j int) bool {
				if prs[i].Number == prs[j].Number {
					return prs[i].Title < prs[j].Title
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
				Error: core.Trim(err.Error()),
			})
			if !reviewJSON {
				cli.Warnf("failed to fetch review requested PRs for %s: %s", repoFullName, core.Trim(err.Error()))
			}
		} else {
			sort.Slice(prs, func(i, j int) bool {
				if prs[i].Number == prs[j].Number {
					return prs[i].Title < prs[j].Title
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
		result := core.JSONMarshal(output)
		if !result.OK {
			return resultError(result, "qa.review.json")
		}
		cli.Print("%s\n", string(result.Value.([]byte)))
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

	output, stderr, err := runReviewCommand(ctx, "gh", args...)
	if err != nil {
		message := core.Trim(stderr)
		if message == "" {
			message = core.Trim(err.Error())
		}
		return nil, log.E("qa.fetchPRs", message, nil)
	}

	var prs []PullRequest
	result := core.JSONUnmarshal(output, &prs)
	if !result.OK {
		return nil, resultError(result, "qa.fetchPRs.json")
	}

	return prs, nil
}

type reviewCommandReadResult struct {
	data []byte
	err  error
}

type reviewCommandWaitResult struct {
	state *os.ProcessState
	err   error
}

func runReviewCommand(ctx context.Context, command string, args ...string) ([]byte, string, error) {
	appResult := (core.App{}).Find(command, command)
	if !appResult.OK {
		return nil, "", resultError(appResult, "qa.review.find")
	}
	app := appResult.Value.(*core.App)

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, "", err
	}
	defer stdoutReader.Close()

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		stdoutWriter.Close()
		return nil, "", err
	}
	defer stderrReader.Close()

	process, err := os.StartProcess(app.Path, append([]string{command}, args...), &os.ProcAttr{
		Files: []*os.File{os.Stdin, stdoutWriter, stderrWriter},
		Env:   os.Environ(),
	})
	stdoutWriter.Close()
	stderrWriter.Close()
	if err != nil {
		return nil, "", err
	}

	stdoutDone := make(chan reviewCommandReadResult, 1)
	stderrDone := make(chan reviewCommandReadResult, 1)
	waitDone := make(chan reviewCommandWaitResult, 1)

	go readReviewCommandOutput(stdoutReader, stdoutDone)
	go readReviewCommandOutput(stderrReader, stderrDone)
	go func() {
		state, waitErr := process.Wait()
		waitDone <- reviewCommandWaitResult{state: state, err: waitErr}
	}()

	var waitResult reviewCommandWaitResult
	select {
	case waitResult = <-waitDone:
	case <-ctx.Done():
		_ = process.Kill()
		waitResult = <-waitDone
	}

	stdoutResult := <-stdoutDone
	stderrResult := <-stderrDone
	stderr := string(stderrResult.data)

	if stdoutResult.err != nil {
		return stdoutResult.data, stderr, stdoutResult.err
	}
	if stderrResult.err != nil {
		return stdoutResult.data, stderr, stderrResult.err
	}
	if ctx.Err() != nil {
		return stdoutResult.data, stderr, ctx.Err()
	}
	if waitResult.err != nil {
		return stdoutResult.data, stderr, waitResult.err
	}
	if waitResult.state != nil && !waitResult.state.Success() {
		return stdoutResult.data, stderr, core.E("qa.review.run", core.Sprintf("%s exited with status %d", command, waitResult.state.ExitCode()), nil)
	}

	return stdoutResult.data, stderr, nil
}

func readReviewCommandOutput(reader *os.File, done chan<- reviewCommandReadResult) {
	data, err := io.ReadAll(reader)
	done <- reviewCommandReadResult{data: data, err: err}
}

func resultError(result core.Result, operation string) error {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E(operation, core.Sprintf("core result failed: %v", result.Value), nil)
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
	stats := core.Sprintf("+%d/-%d, %d files",
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
			return "✗", errorStyle, core.Sprintf("CI failed: %s", failedChecks[0])
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
