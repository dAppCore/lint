// cmd_watch.go implements the 'qa watch' command for monitoring GitHub Actions.
//
// Usage:
//   core qa watch              # Watch current repo's latest push
//   core qa watch --repo X     # Watch specific repo
//   core qa watch --commit SHA # Watch specific commit
//   core qa watch --timeout 5m # Custom timeout (default: 10m)

package qa

import (
	"cmp"
	"context"
	"slices"
	"time"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

const (
	cmdWatchQaWatchcea0f7 = "qa.watch"
	cmdWatchRepo644bfb    = "--repo"
	cmdWatchSS02a8c6      = "%s %s\n"
)

// Watch command flags
var (
	watchRepo    string
	watchCommit  string
	watchTimeout time.Duration
)

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID           int64     `json:"databaseId"`
	Name         string    `json:"name"`
	DisplayTitle string    `json:"displayTitle"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	HeadSha      string    `json:"headSha"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// WorkflowJob represents a job within a workflow run
type WorkflowJob struct {
	ID         int64     `json:"databaseId"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	URL        string    `json:"url"`
	Steps      []JobStep `json:"steps"`
}

// JobStep represents a step within a job
type JobStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Number     int    `json:"number"`
}

// addWatchCommand adds the 'watch' subcommand to the qa command.
func addWatchCommand(c *core.Core) core.Result {
	watchTimeout = 10 * time.Minute
	return registerQACommand(c, "qa/watch", qaText("cmd.qa.watch.long"), runWatch)
}

func runWatch() core.Result {
	if !findQAExecutable("gh").OK {
		return core.Fail(core.E(cmdWatchQaWatchcea0f7, qaText("error.gh_not_found"), nil))
	}

	repoResult := resolveRepo(watchRepo)
	if !repoResult.OK {
		return repoResult
	}
	repoFullName := repoResult.Value.(string)
	commitResult := resolveCommit(watchCommit)
	if !commitResult.OK {
		return commitResult
	}
	commitSha := commitResult.Value.(string)

	printWatchHeader(repoFullName, commitSha)
	ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
	defer cancel()
	return watchWorkflowRuns(ctx, repoFullName, commitSha)
}

func printWatchHeader(repoFullName string, commitSha string) {
	cli.Print(cmdWatchSS02a8c6, dimStyle.Render(qaLabel("repo")), repoFullName)
	shaPrefix := commitSha
	if len(commitSha) > 8 {
		shaPrefix = commitSha[:8]
	}
	cli.Print(cmdWatchSS02a8c6, dimStyle.Render(qaText("cmd.qa.watch.commit")), shaPrefix)
	cli.Blank()
}

func watchWorkflowRuns(ctx context.Context, repoFullName string, commitSha string) core.Result {
	pollInterval := 3 * time.Second
	var lastStatus string
	waitingStatus := dimStyle.Render(qaText("cmd.qa.watch.waiting_for_workflows"))

	for {
		if ctx.Err() != nil {
			cli.Blank()
			return core.Fail(core.E(cmdWatchQaWatchcea0f7, qaText("cmd.qa.watch.timeout", map[string]any{"Duration": watchTimeout}), nil))
		}

		runResult := fetchWorkflowRunsForCommit(ctx, repoFullName, commitSha)
		if !runResult.OK {
			return core.Fail(core.Wrap(runResult.Value.(error), cmdWatchQaWatchcea0f7, "failed to fetch workflow runs"))
		}
		runs := runResult.Value.([]WorkflowRun)
		if len(runs) == 0 {
			if waitingStatus != lastStatus {
				cli.Print("%s\n", waitingStatus)
				lastStatus = waitingStatus
			}
			time.Sleep(pollInterval)
			continue
		}

		counts := countWorkflowRuns(runs)
		lastStatus = printWatchStatus(formatWorkflowStatus(len(runs), counts), lastStatus)
		if counts.allComplete {
			cli.Blank()
			return printResults(ctx, repoFullName, runs)
		}

		time.Sleep(pollInterval)
	}
}

type workflowRunCounts struct {
	pending     int
	success     int
	failed      int
	allComplete bool
}

func countWorkflowRuns(runs []WorkflowRun) workflowRunCounts {
	counts := workflowRunCounts{allComplete: true}
	for _, run := range runs {
		if run.Status != "completed" {
			counts.allComplete = false
			counts.pending++
			continue
		}
		if run.Conclusion == "success" {
			counts.success++
			continue
		}
		counts.failed++
	}
	return counts
}

func formatWorkflowStatus(total int, counts workflowRunCounts) string {
	parts := make([]string, 0, 3)
	if counts.pending > 0 {
		parts = append(parts, warningStyle.Render(core.Sprintf("%d running", counts.pending)))
	}
	if counts.success > 0 {
		parts = append(parts, successStyle.Render(core.Sprintf("%d passed", counts.success)))
	}
	if counts.failed > 0 {
		parts = append(parts, errorStyle.Render(core.Sprintf("%d failed", counts.failed)))
	}
	return core.Sprintf("%d workflow(s): %s", total, core.Join(", ", parts...))
}

func printWatchStatus(status string, lastStatus string) string {
	if status != lastStatus {
		cli.Print("%s\n", status)
		return status
	}
	return lastStatus
}

// resolveRepo determines the repo to watch
func resolveRepo(specified string) core.Result {
	if specified != "" {
		// If it contains /, assume it's already full name
		if core.Contains(specified, "/") {
			return core.Ok(specified)
		}
		// Try to get org from current directory
		org := detectOrgFromGit()
		if org != "" {
			return core.Ok(core.Concat(org, "/", specified))
		}
		return core.Fail(core.E(cmdWatchQaWatchcea0f7, qaText("cmd.qa.watch.error.repo_format"), nil))
	}

	// Detect from current directory
	return detectRepoFromGit()
}

// resolveCommit determines the commit to watch
func resolveCommit(specified string) core.Result {
	if specified != "" {
		return core.Ok(specified)
	}

	// Get HEAD commit
	output := runQACommand(core.Background(), "git", "rev-parse", "HEAD")
	if output.Err != nil {
		return core.Fail(core.Wrap(output.Err, cmdWatchQaWatchcea0f7, "failed to get HEAD commit"))
	}

	return core.Ok(core.Trim(output.Stdout))
}

// detectRepoFromGit detects the repo from git remote
func detectRepoFromGit() core.Result {
	output := runQACommand(core.Background(), "git", "remote", "get-url", "origin")
	if output.Err != nil {
		return core.Fail(core.E(cmdWatchQaWatchcea0f7, qaText("cmd.qa.watch.error.not_git_repo"), nil))
	}

	url := core.Trim(output.Stdout)
	return parseGitHubRepo(url)
}

// detectOrgFromGit tries to detect the org from git remote
func detectOrgFromGit() string {
	repoResult := detectRepoFromGit()
	if !repoResult.OK {
		return ""
	}
	parts := core.Split(repoResult.Value.(string), "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// parseGitHubRepo extracts org/repo from a git URL
func parseGitHubRepo(url string) core.Result {
	// Handle SSH URLs: git@github.com:org/repo.git
	if core.HasPrefix(url, "git@github.com:") {
		repoPath := core.TrimPrefix(url, "git@github.com:")
		repoPath = core.TrimSuffix(repoPath, ".git")
		return core.Ok(repoPath)
	}

	// Handle HTTPS URLs: https://github.com/org/repo.git
	if core.Contains(url, "github.com/") {
		parts := core.Split(url, "github.com/")
		if len(parts) >= 2 {
			repoPath := core.TrimSuffix(parts[1], ".git")
			return core.Ok(repoPath)
		}
	}

	return core.Fail(core.E("qa.parseGitHubRepo", core.Concat("could not parse GitHub repo from URL: ", url), nil))
}

// fetchWorkflowRunsForCommit fetches workflow runs for a specific commit
func fetchWorkflowRunsForCommit(ctx context.Context, repoFullName, commitSha string) core.Result {
	args := []string{
		"run", "list",
		cmdWatchRepo644bfb, repoFullName,
		"--commit", commitSha,
		"--json", "databaseId,name,displayTitle,status,conclusion,headSha,url,createdAt,updatedAt",
	}

	output := runQACommand(ctx, "gh", args...)
	if output.Err != nil {
		// Check if context was cancelled/deadline exceeded
		if ctx.Err() != nil {
			return core.Fail(ctx.Err())
		}
		message := core.Trim(output.Stderr)
		if message != "" {
			return core.Fail(cli.Err("%s", message))
		}
		return core.Fail(output.Err)
	}

	var runs []WorkflowRun
	if r := core.JSONUnmarshal([]byte(output.Stdout), &runs); !r.OK {
		return r
	}

	return core.Ok(runs)
}

// printResults prints the final results with actionable information
func printResults(ctx context.Context, repoFullName string, runs []WorkflowRun) core.Result {
	var failures []WorkflowRun
	var successes []WorkflowRun

	for _, run := range runs {
		if run.Conclusion == "success" {
			successes = append(successes, run)
		} else {
			// Treat all non-success as failures (failure, cancelled, timed_out, etc.)
			failures = append(failures, run)
		}
	}

	slices.SortFunc(successes, compareWorkflowRun)
	slices.SortFunc(failures, compareWorkflowRun)

	// Print successes briefly
	for _, run := range successes {
		cli.Print(cmdWatchSS02a8c6, successStyle.Render(qaText("common.label.success")), run.Name)
	}

	// Print failures with details
	for _, run := range failures {
		cli.Print(cmdWatchSS02a8c6, errorStyle.Render(qaText("common.label.error")), run.Name)

		// Fetch failed job details
		failedJob, failedStep, errorLine := fetchFailureDetails(ctx, repoFullName, run.ID)
		if failedJob != "" {
			cli.Print("  %s Job: %s", dimStyle.Render("->"), failedJob)
			if failedStep != "" {
				cli.Print(" (step: %s)", failedStep)
			}
			cli.Blank()
		}
		if errorLine != "" {
			cli.Print("  %s Error: %s\n", dimStyle.Render("->"), errorLine)
		}
		cli.Print("  %s %s\n", dimStyle.Render("->"), run.URL)
	}

	// Exit with error if any failures
	if len(failures) > 0 {
		cli.Blank()
		return core.Fail(cli.Err("%s", qaText("cmd.qa.watch.workflows_failed", map[string]any{"Count": len(failures)})))
	}

	cli.Blank()
	cli.Print("%s\n", successStyle.Render(qaText("cmd.qa.watch.all_passed")))
	return core.Ok(nil)
}

// fetchFailureDetails fetches details about why a workflow failed
func fetchFailureDetails(ctx context.Context, repoFullName string, runID int64) (jobName, stepName, errorLine string) {
	// Fetch jobs for this run
	args := []string{
		"run", "view", core.Sprintf("%d", runID),
		cmdWatchRepo644bfb, repoFullName,
		"--json", "jobs",
	}

	output := runQACommand(ctx, "gh", args...)
	if output.Err != nil {
		return "", "", ""
	}

	var result struct {
		Jobs []WorkflowJob `json:"jobs"`
	}

	if r := core.JSONUnmarshal([]byte(output.Stdout), &result); !r.OK {
		return "", "", ""
	}

	slices.SortFunc(result.Jobs, compareWorkflowJob)

	// Find the failed job and step
	for _, job := range result.Jobs {
		if job.Conclusion == "failure" {
			jobName = job.Name
			slices.SortFunc(job.Steps, compareJobStep)
			for _, step := range job.Steps {
				if step.Conclusion == "failure" {
					stepName = core.Sprintf("%d: %s", step.Number, step.Name)
					break
				}
			}
			break
		}
	}

	// Try to get the error line from logs (if available)
	errorLine = fetchErrorFromLogs(ctx, repoFullName, runID)

	return jobName, stepName, errorLine
}

// fetchErrorFromLogs attempts to extract the first error line from workflow logs
func fetchErrorFromLogs(ctx context.Context, repoFullName string, runID int64) string {
	// Use gh run view --log-failed to get failed step logs
	args := []string{
		"run", "view", core.Sprintf("%d", runID),
		cmdWatchRepo644bfb, repoFullName,
		"--log-failed",
	}

	output := runQACommand(ctx, "gh", args...)
	if output.Err != nil {
		return ""
	}

	// Parse output to find the first meaningful error line
	lines := core.Split(output.Stdout, "\n")
	for _, line := range lines {
		line = core.Trim(line)
		if line == "" {
			continue
		}

		// Skip common metadata/progress lines
		lower := core.Lower(line)
		if core.HasPrefix(lower, "##[") { // GitHub Actions command markers
			continue
		}
		if core.HasPrefix(line, "Run ") || core.HasPrefix(line, "Running ") {
			continue
		}

		// Look for error indicators
		if core.Contains(lower, "error") ||
			core.Contains(lower, "failed") ||
			core.Contains(lower, "fatal") ||
			core.Contains(lower, "panic") ||
			core.Contains(line, ": ") { // Likely a file:line or key: value format
			// Truncate long lines
			if len(line) > 120 {
				line = line[:117] + "..."
			}
			return line
		}
	}

	return ""
}

func compareWorkflowRun(a, b WorkflowRun) int {
	return cmp.Or(
		cmp.Compare(a.Name, b.Name),
		cmp.Compare(a.DisplayTitle, b.DisplayTitle),
		a.CreatedAt.Compare(b.CreatedAt),
		a.UpdatedAt.Compare(b.UpdatedAt),
		cmp.Compare(a.ID, b.ID),
		cmp.Compare(a.URL, b.URL),
	)
}

func compareWorkflowJob(a, b WorkflowJob) int {
	return cmp.Or(
		cmp.Compare(a.Name, b.Name),
		cmp.Compare(a.Conclusion, b.Conclusion),
		cmp.Compare(a.Status, b.Status),
		cmp.Compare(a.ID, b.ID),
		cmp.Compare(a.URL, b.URL),
	)
}

func compareJobStep(a, b JobStep) int {
	return cmp.Or(
		cmp.Compare(a.Number, b.Number),
		cmp.Compare(a.Name, b.Name),
		cmp.Compare(a.Conclusion, b.Conclusion),
		cmp.Compare(a.Status, b.Status),
	)
}
