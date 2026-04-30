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
	if result := (core.App{}).Find("gh", "GitHub CLI"); !result.OK {
		return core.E("qa.review", i18n.T("error.gh_not_found"), nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repoFullName, err := resolveReviewRepo()
	if err != nil {
		return err
	}

	scopes := selectedReviewScopes()
	fetches := fetchReviewPRs(ctx, repoFullName, scopes)
	return outputReview(repoFullName, scopes, fetches)
}

type reviewScopes struct {
	showMine      bool
	showRequested bool
}

type reviewFetches struct {
	mine              []PullRequest
	requested         []PullRequest
	errors            []ReviewFetchError
	mineFetched       bool
	requestedFetched  bool
	successfulFetches int
}

func resolveReviewRepo() (string, error) {
	if reviewRepo != "" {
		return reviewRepo, nil
	}
	repoFullName, err := detectRepoFromGit()
	if err != nil {
		return "", core.E("qa.review", i18n.T("cmd.qa.review.error.no_repo"), nil)
	}
	return repoFullName, nil
}

func selectedReviewScopes() reviewScopes {
	showMine := reviewMine || (!reviewMine && !reviewRequested)
	showRequested := reviewRequested || (!reviewMine && !reviewRequested)
	return reviewScopes{showMine: showMine, showRequested: showRequested}
}

func fetchReviewPRs(ctx context.Context, repoFullName string, scopes reviewScopes) reviewFetches {
	fetches := reviewFetches{errors: make([]ReviewFetchError, 0)}
	if scopes.showMine {
		fetches.mine, fetches.mineFetched = fetchReviewScope(ctx, repoFullName, "mine", "author:@me", &fetches)
	}
	if scopes.showRequested {
		fetches.requested, fetches.requestedFetched = fetchReviewScope(ctx, repoFullName, "requested", "review-requested:@me", &fetches)
	}
	return fetches
}

func fetchReviewScope(
	ctx context.Context,
	repoFullName string,
	scope string,
	query string,
	fetches *reviewFetches,
) ([]PullRequest, bool) {
	prs, err := fetchPRs(ctx, repoFullName, query)
	if err != nil {
		recordReviewFetchError(repoFullName, scope, err, fetches)
		return nil, false
	}
	sortPullRequests(prs)
	fetches.successfulFetches++
	return prs, true
}

func recordReviewFetchError(repoFullName string, scope string, err error, fetches *reviewFetches) {
	message := core.Trim(err.Error())
	fetches.errors = append(fetches.errors, ReviewFetchError{
		Repo:  repoFullName,
		Scope: scope,
		Error: message,
	})
	if reviewJSON {
		return
	}
	if scope == "mine" {
		cli.Warnf("failed to fetch your PRs for %s: %s", repoFullName, message)
		return
	}
	cli.Warnf("failed to fetch review requested PRs for %s: %s", repoFullName, message)
}

func sortPullRequests(prs []PullRequest) {
	sort.Slice(prs, func(i, j int) bool {
		if prs[i].Number == prs[j].Number {
			return prs[i].Title < prs[j].Title
		}
		return prs[i].Number < prs[j].Number
	})
}

func outputReview(repoFullName string, scopes reviewScopes, fetches reviewFetches) error {
	output := reviewOutput{
		Mine:             fetches.mine,
		Requested:        fetches.requested,
		TotalMine:        len(fetches.mine),
		TotalRequested:   len(fetches.requested),
		ShowingMine:      scopes.showMine,
		ShowingRequested: scopes.showRequested,
		FetchErrors:      fetches.errors,
	}

	if reviewJSON {
		return outputReviewJSON(repoFullName, output, fetches)
	}
	return outputReviewText(repoFullName, scopes, fetches)
}

func outputReviewJSON(repoFullName string, output reviewOutput, fetches reviewFetches) error {
	result := core.JSONMarshal(output)
	if !result.OK {
		return resultError(result, "qa.review.json")
	}
	cli.Print("%s\n", string(result.Value.([]byte)))
	return reviewFetchFailure(repoFullName, fetches)
}

func outputReviewText(repoFullName string, scopes reviewScopes, fetches reviewFetches) error {
	if err := reviewFetchFailure(repoFullName, fetches); err != nil {
		return err
	}

	if scopes.showMine && fetches.mineFetched {
		if err := printMyPRs(fetches.mine); err != nil {
			return err
		}
	}

	if scopes.showRequested && fetches.requestedFetched {
		if scopes.showMine && fetches.mineFetched {
			cli.Blank()
		}
		return printRequestedPRs(fetches.requested)
	}

	return nil
}

func reviewFetchFailure(repoFullName string, fetches reviewFetches) error {
	if fetches.successfulFetches == 0 && len(fetches.errors) > 0 {
		return cli.Err("failed to fetch pull requests for %s", repoFullName)
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
		return nil, core.E("qa.fetchPRs", message, nil)
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
		killErr := process.Kill()
		waitResult = <-waitDone
		if killErr != nil && waitResult.err == nil {
			waitResult.err = killErr
		}
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
	if pr.IsDraft {
		return "◯", dimStyle, "Draft - convert to ready when done"
	}

	ci := prCIStatus(pr)
	approved := pr.ReviewDecision == "APPROVED"
	changesRequested := pr.ReviewDecision == "CHANGES_REQUESTED"
	hasConflicts := pr.Mergeable == "CONFLICTING"

	if hasConflicts {
		return "✗", errorStyle, "Needs rebase - has merge conflicts"
	}
	if ci.failed {
		return failedPRStatus(ci.failedChecks)
	}
	if changesRequested {
		return "✗", warningStyle, "Changes requested - address review feedback"
	}
	if ci.pending {
		return "◯", warningStyle, "CI running..."
	}
	if !approved && pr.ReviewDecision != "" {
		return "◯", warningStyle, "Awaiting review"
	}
	if approved && ci.passed {
		return "✓", successStyle, "Ready to merge"
	}
	return "◯", dimStyle, ""
}

type pullRequestCIStatus struct {
	passed       bool
	failed       bool
	pending      bool
	failedChecks []string
}

func prCIStatus(pr PullRequest) pullRequestCIStatus {
	ci := pullRequestCIStatus{passed: true}
	if pr.StatusChecks == nil {
		return ci
	}
	for _, check := range pr.StatusChecks.Contexts {
		applyCheckStatus(&ci, check)
	}
	return ci
}

func applyCheckStatus(ci *pullRequestCIStatus, check StatusContext) {
	switch check.Conclusion {
	case "FAILURE", "failure":
		ci.failed = true
		ci.passed = false
		ci.failedChecks = append(ci.failedChecks, check.Name)
	case "PENDING", "pending", "":
		if check.State == "PENDING" || check.State == "" {
			ci.pending = true
			ci.passed = false
		}
	}
}

func failedPRStatus(failedChecks []string) (string, *cli.AnsiStyle, string) {
	if len(failedChecks) == 0 {
		return "✗", errorStyle, "CI failed"
	}
	sort.Strings(failedChecks)
	return "✗", errorStyle, core.Sprintf("CI failed: %s", failedChecks[0])
}

// truncate shortens a string to max length (rune-safe for UTF-8)
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
