// cmd_issues.go implements the 'qa issues' command for intelligent issue triage.
//
// Usage:
//   core qa issues              # Show prioritised, actionable issues
//   core qa issues --mine       # Show issues assigned to you
//   core qa issues --triage     # Show issues needing triage (no labels/assignee)
//   core qa issues --blocked    # Show blocked issues

package qa

import (
	"cmp"
	"slices"
	"time"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/io"
	"dappco.re/go/scm/repos"
)

const (
	cmdIssuesQaIssues972b25 = "qa.issues"
)

// Issue command flags
var (
	issuesMine     bool
	issuesTriage   bool
	issuesBlocked  bool
	issuesRegistry string
	issuesLimit    int
	issuesJSON     bool
)

// Issue represents a GitHub issue with triage metadata
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Comments struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
	URL string `json:"url"`

	// Computed fields
	RepoName   string `json:"repo_name"`
	Priority   int    `json:"priority"` // Lower = higher priority
	Category   string `json:"category"` // "needs_response", "ready", "blocked", "triage"
	ActionHint string `json:"action_hint,omitempty"`
}

type IssueFetchError struct {
	Repo  string `json:"repo"`
	Error string `json:"error"`
}

type IssueCategoryOutput struct {
	Category string  `json:"category"`
	Count    int     `json:"count"`
	Issues   []Issue `json:"issues"`
}

type IssuesOutput struct {
	TotalIssues    int                   `json:"total_issues"`
	FilteredIssues int                   `json:"filtered_issues"`
	ShowingMine    bool                  `json:"showing_mine"`
	ShowingTriage  bool                  `json:"showing_triage"`
	ShowingBlocked bool                  `json:"showing_blocked"`
	Categories     []IssueCategoryOutput `json:"categories"`
	FetchErrors    []IssueFetchError     `json:"fetch_errors"`
}

// addIssuesCommand adds the 'issues' subcommand to qa.
func addIssuesCommand(c *core.Core) core.Result {
	issuesLimit = 50
	return registerQACommand(c, "qa/issues", qaText("cmd.qa.issues.long"), runQAIssues)
}

func runQAIssues() core.Result {
	if !findQAExecutable("gh").OK {
		return core.Fail(core.E(cmdIssuesQaIssues972b25, qaText("error.gh_not_found"), nil))
	}

	regResult := loadIssuesRegistry()
	if !regResult.OK {
		return regResult
	}
	reg := regResult.Value.(*repos.Registry)

	fetched := fetchAllQAIssues(reg)
	if len(fetched.issues) == 0 {
		return handleNoIssues(fetched)
	}

	categorised := categoriseIssues(fetched.issues)
	categorised = filterIssueCategories(categorised)
	if issuesJSON {
		return printCategorisedIssuesJSON(len(fetched.issues), categorised, fetched.errors)
	}

	printCategorisedIssues(categorised)
	return core.Ok(nil)
}

type issueFetchResult struct {
	issues            []Issue
	errors            []IssueFetchError
	successfulFetches int
}

func loadIssuesRegistry() core.Result {
	var reg *repos.Registry
	var err error
	if issuesRegistry != "" {
		reg, err = repos.LoadRegistry(io.Local, issuesRegistry)
	} else {
		registryPath, findErr := repos.FindRegistry(io.Local)
		if findErr != nil {
			return core.Fail(core.E(cmdIssuesQaIssues972b25, qaText("error.registry_not_found"), nil))
		}
		reg, err = repos.LoadRegistry(io.Local, registryPath)
	}
	if err != nil {
		return core.Fail(core.E(cmdIssuesQaIssues972b25, "failed to load registry", err))
	}
	return core.Ok(reg)
}

func fetchAllQAIssues(reg *repos.Registry) issueFetchResult {
	result := issueFetchResult{}
	fetchErrors := make([]IssueFetchError, 0)
	repoList := reg.List()
	slices.SortFunc(repoList, func(a, b *repos.Repo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i, repo := range repoList {
		printIssueFetchProgress(i, len(repoList), repo.Name)
		issuesResult := fetchQAIssues(reg.Org, repo.Name, issuesLimit)
		if !issuesResult.OK {
			err := issuesResult.Value.(error)
			fetchErrors = append(fetchErrors, IssueFetchError{
				Repo:  repo.Name,
				Error: core.Trim(err.Error()),
			})
			if !issuesJSON {
				cli.Print("%s\n", warningStyle.Render(qaText(
					"cmd.qa.issues.fetch_error",
					map[string]any{"Repo": repo.Name, "Error": core.Trim(err.Error())},
				)))
			}
			continue // Skip repos with errors
		}
		issues := issuesResult.Value.([]Issue)
		result.issues = append(result.issues, issues...)
		result.successfulFetches++
	}
	result.errors = fetchErrors
	return result
}

func printIssueFetchProgress(index int, total int, repoName string) {
	if issuesJSON {
		return
	}
	cli.Print("%s %d/%d %s\n",
		dimStyle.Render(qaText("cmd.qa.issues.fetching")),
		index+1, total, repoName)
}

func handleNoIssues(result issueFetchResult) core.Result {
	if issuesJSON {
		if r := printCategorisedIssuesJSON(0, emptyIssueCategories(), result.errors); !r.OK {
			return r
		}
		return issueFetchFailure(result)
	}
	if r := issueFetchFailure(result); !r.OK {
		return r
	}
	cli.Text(qaText("cmd.qa.issues.no_issues"))
	return core.Ok(nil)
}

func emptyIssueCategories() map[string][]Issue {
	return map[string][]Issue{
		"needs_response": {},
		"ready":          {},
		"blocked":        {},
		"triage":         {},
	}
}

func issueFetchFailure(result issueFetchResult) core.Result {
	if result.successfulFetches == 0 && len(result.errors) > 0 {
		return core.Fail(cli.Err("failed to fetch issues from any repository"))
	}
	return core.Ok(nil)
}

func filterIssueCategories(categorised map[string][]Issue) map[string][]Issue {
	if issuesMine {
		categorised = filterMine(categorised)
	}
	if issuesTriage {
		categorised = filterCategory(categorised, "triage")
	}
	if issuesBlocked {
		categorised = filterCategory(categorised, "blocked")
	}
	return categorised
}

func fetchQAIssues(org, repoName string, limit int) core.Result {
	repoFullName := cli.Sprintf("%s/%s", org, repoName)

	args := []string{
		"issue", "list",
		"--repo", repoFullName,
		"--state", "open",
		"--limit", cli.Sprintf("%d", limit),
		"--json", "number,title,state,body,createdAt,updatedAt,author,assignees,labels,comments,url",
	}

	output := runQACommand(core.Background(), "gh", args...)
	if output.Err != nil {
		if output.Stderr != "" {
			return core.Fail(core.E("qa.fetchQAIssues", core.Trim(output.Stderr), nil))
		}
		return core.Fail(output.Err)
	}

	var issues []Issue
	if r := core.JSONUnmarshal([]byte(output.Stdout), &issues); !r.OK {
		return r
	}

	// Tag with repo name
	for i := range issues {
		issues[i].RepoName = repoName
	}

	return core.Ok(issues)
}

func categoriseIssues(issues []Issue) map[string][]Issue {
	result := map[string][]Issue{
		"needs_response": {},
		"ready":          {},
		"blocked":        {},
		"triage":         {},
	}

	currentUser := getCurrentUser()

	for i := range issues {
		issue := &issues[i]
		categoriseIssue(issue, currentUser)
		result[issue.Category] = append(result[issue.Category], *issue)
	}

	// Sort each category by priority
	for cat := range result {
		slices.SortFunc(result[cat], func(a, b Issue) int {
			if priority := cmp.Compare(a.Priority, b.Priority); priority != 0 {
				return priority
			}
			if byDate := cmp.Compare(b.UpdatedAt.Unix(), a.UpdatedAt.Unix()); byDate != 0 {
				return byDate
			}
			if repo := cmp.Compare(a.RepoName, b.RepoName); repo != 0 {
				return repo
			}
			return cmp.Compare(a.Number, b.Number)
		})
	}

	return result
}

func categoriseIssue(issue *Issue, currentUser string) {
	labels := getLabels(issue)

	// Check if blocked
	for _, l := range labels {
		if core.HasPrefix(l, "blocked") || l == "waiting" {
			issue.Category = "blocked"
			issue.Priority = 30
			issue.ActionHint = qaText("cmd.qa.issues.hint.blocked")
			return
		}
	}

	// Check if needs triage (no labels, no assignee)
	if len(issue.Labels.Nodes) == 0 && len(issue.Assignees.Nodes) == 0 {
		issue.Category = "triage"
		issue.Priority = 20
		issue.ActionHint = qaText("cmd.qa.issues.hint.triage")
		return
	}

	// Check if needs response (recent comment from someone else)
	if issue.Comments.TotalCount > 0 && len(issue.Comments.Nodes) > 0 {
		lastComment := issue.Comments.Nodes[len(issue.Comments.Nodes)-1]
		// If last comment is not from current user and is recent
		if lastComment.Author.Login != currentUser {
			age := time.Since(lastComment.CreatedAt)
			if age < 48*time.Hour {
				issue.Category = "needs_response"
				issue.Priority = 10
				issue.ActionHint = cli.Sprintf("@%s %s", lastComment.Author.Login, qaText("cmd.qa.issues.hint.needs_response"))
				return
			}
		}
	}

	// Default: ready to work
	issue.Category = "ready"
	issue.Priority = calculatePriority(labels)
	issue.ActionHint = ""
}

// calculatePriority chooses the most urgent matching label so label order
// does not change how issues are ranked.
func calculatePriority(labels []string) int {
	priority := 50

	// Priority labels
	for _, l := range labels {
		switch {
		case core.Contains(l, "critical") || core.Contains(l, "urgent"):
			priority = min(priority, 1)
		case core.Contains(l, "high"):
			priority = min(priority, 10)
		case core.Contains(l, "medium"):
			priority = min(priority, 30)
		case core.Contains(l, "low"):
			priority = min(priority, 70)
		case l == "good-first-issue" || l == "good first issue":
			priority = min(priority, 15) // Boost good first issues
		case l == "help-wanted" || l == "help wanted":
			priority = min(priority, 20)
		case l == "agent:ready" || l == "agentic":
			priority = min(priority, 5) // AI-ready issues are high priority
		}
	}

	return priority
}

func getLabels(issue *Issue) []string {
	var labels []string
	for _, l := range issue.Labels.Nodes {
		labels = append(labels, core.Lower(l.Name))
	}
	return labels
}

func getCurrentUser() string {
	output := runQACommand(core.Background(), "gh", "api", "user", "--jq", ".login")
	if output.Err != nil {
		return ""
	}
	return core.Trim(output.Stdout)
}

func filterMine(categorised map[string][]Issue) map[string][]Issue {
	currentUser := getCurrentUser()
	result := make(map[string][]Issue)

	for cat, issues := range categorised {
		var filtered []Issue
		for _, issue := range issues {
			for _, a := range issue.Assignees.Nodes {
				if a.Login == currentUser {
					filtered = append(filtered, issue)
					break
				}
			}
		}
		if len(filtered) > 0 {
			result[cat] = filtered
		}
	}

	return result
}

func filterCategory(categorised map[string][]Issue, category string) map[string][]Issue {
	if issues, ok := categorised[category]; ok && len(issues) > 0 {
		return map[string][]Issue{category: issues}
	}
	return map[string][]Issue{}
}

func printCategorisedIssues(categorised map[string][]Issue) {
	// Print in order: needs_response, ready, blocked, triage
	categories := []struct {
		key   string
		title string
		style *cli.AnsiStyle
	}{
		{"needs_response", qaText("cmd.qa.issues.category.needs_response"), warningStyle},
		{"ready", qaText("cmd.qa.issues.category.ready"), successStyle},
		{"blocked", qaText("cmd.qa.issues.category.blocked"), errorStyle},
		{"triage", qaText("cmd.qa.issues.category.triage"), dimStyle},
	}

	first := true
	for _, cat := range categories {
		issues := categorised[cat.key]
		if len(issues) == 0 {
			continue
		}

		if !first {
			cli.Blank()
		}
		first = false

		cli.Print("%s (%d):\n", cat.style.Render(cat.title), len(issues))

		for _, issue := range issues {
			printTriagedIssue(issue)
		}
	}

	if first {
		cli.Text(qaText("cmd.qa.issues.no_issues"))
	}
}

func printCategorisedIssuesJSON(totalIssues int, categorised map[string][]Issue, fetchErrors []IssueFetchError) core.Result {
	categories := []string{"needs_response", "ready", "blocked", "triage"}
	filteredIssues := 0
	categoryOutput := make([]IssueCategoryOutput, 0, len(categories))

	for _, category := range categories {
		issues := categorised[category]
		filteredIssues += len(issues)
		categoryOutput = append(categoryOutput, IssueCategoryOutput{
			Category: category,
			Count:    len(issues),
			Issues:   issues,
		})
	}

	output := IssuesOutput{
		TotalIssues:    totalIssues,
		FilteredIssues: filteredIssues,
		ShowingMine:    issuesMine,
		ShowingTriage:  issuesTriage,
		ShowingBlocked: issuesBlocked,
		Categories:     categoryOutput,
		FetchErrors:    fetchErrors,
	}

	data := core.JSONMarshalIndent(output, "", "  ")
	if !data.OK {
		return data
	}
	cli.Print("%s\n", string(data.Value.([]byte)))
	return core.Ok(nil)
}

func printTriagedIssue(issue Issue) {
	// #42 [core-bio] Fix avatar upload
	num := cli.TitleStyle.Render(cli.Sprintf("#%d", issue.Number))
	repo := dimStyle.Render(cli.Sprintf("[%s]", issue.RepoName))
	title := cli.ValueStyle.Render(truncate(issue.Title, 50))

	cli.Print("  %s %s %s", num, repo, title)

	// Add labels if priority-related
	var importantLabels []string
	for _, l := range issue.Labels.Nodes {
		name := core.Lower(l.Name)
		if core.Contains(name, "priority") || core.Contains(name, "critical") ||
			name == "good-first-issue" || name == "agent:ready" || name == "agentic" {
			importantLabels = append(importantLabels, l.Name)
		}
	}
	if len(importantLabels) > 0 {
		slices.Sort(importantLabels)
		cli.Print(" %s", warningStyle.Render("["+core.Join(", ", importantLabels...)+"]"))
	}

	// Add age
	age := cli.FormatAge(issue.UpdatedAt)
	cli.Print(" %s\n", dimStyle.Render(age))

	// Add action hint if present
	if issue.ActionHint != "" {
		cli.Print("      %s %s\n", dimStyle.Render("->"), issue.ActionHint)
	}
}
