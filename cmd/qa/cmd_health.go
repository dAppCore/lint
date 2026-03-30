// cmd_health.go implements the `qa health` command for aggregate CI health.
//
// Usage:
//   core qa health              # Show CI health summary
//   core qa health --problems   # Show only repos with problems

package qa

import (
	"cmp"
	"encoding/json"
	"os/exec"
	"slices"
	"strings"

	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-i18n"
	"forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-scm/repos"
)

// Health command flags.
var (
	healthProblems bool
	healthRegistry string
	healthJSON     bool
)

// HealthWorkflowRun represents a GitHub Actions workflow run.
type HealthWorkflowRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Name       string `json:"name"`
	HeadSha    string `json:"headSha"`
	UpdatedAt  string `json:"updatedAt"`
	URL        string `json:"url"`
}

// RepoHealth represents the CI health of a single repo.
type RepoHealth struct {
	Name         string `json:"name"`
	Status       string `json:"status"` // passing, failing, error, pending, no_ci, disabled
	Message      string `json:"message"`
	URL          string `json:"url"`
	FailingSince string `json:"failing_since,omitempty"`
}

// HealthSummary captures aggregate health counts.
type HealthSummary struct {
	TotalRepos    int            `json:"total_repos"`
	FilteredRepos int            `json:"filtered_repos"`
	Passing       int            `json:"passing"`
	Failing       int            `json:"failing"`
	Errors        int            `json:"errors"`
	Pending       int            `json:"pending"`
	Disabled      int            `json:"disabled"`
	NotConfigured int            `json:"not_configured"`
	PassingRate   int            `json:"passing_rate"`
	ProblemsOnly  bool           `json:"problems_only"`
	ByStatus      map[string]int `json:"by_status"`
}

// HealthOutput is the JSON payload for `qa health --json`.
type HealthOutput struct {
	Summary HealthSummary `json:"summary"`
	Repos   []RepoHealth  `json:"repos"`
}

func addHealthCommand(parent *cli.Command) {
	healthCmd := &cli.Command{
		Use:   "health",
		Short: i18n.T("cmd.qa.health.short"),
		Long:  i18n.T("cmd.qa.health.long"),
		RunE: func(cmd *cli.Command, args []string) error {
			return runHealth()
		},
	}

	healthCmd.Flags().BoolVarP(&healthProblems, "problems", "p", false, i18n.T("cmd.qa.health.flag.problems"))
	healthCmd.Flags().StringVar(&healthRegistry, "registry", "", i18n.T("common.flag.registry"))
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, i18n.T("common.flag.json"))

	parent.AddCommand(healthCmd)
}

func runHealth() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return log.E("qa.health", i18n.T("error.gh_not_found"), nil)
	}

	var reg *repos.Registry
	var err error
	if healthRegistry != "" {
		reg, err = repos.LoadRegistry(io.Local, healthRegistry)
	} else {
		registryPath, findErr := repos.FindRegistry(io.Local)
		if findErr != nil {
			return log.E("qa.health", i18n.T("error.registry_not_found"), nil)
		}
		reg, err = repos.LoadRegistry(io.Local, registryPath)
	}
	if err != nil {
		return log.E("qa.health", "failed to load registry", err)
	}

	repoList := reg.List()
	allHealthResults := make([]RepoHealth, 0, len(repoList))
	for _, repo := range repoList {
		health := fetchRepoHealth(reg.Org, repo.Name)
		allHealthResults = append(allHealthResults, health)
	}

	// Sort by severity first, then repo name for deterministic output.
	slices.SortFunc(allHealthResults, func(a, b RepoHealth) int {
		if p := cmp.Compare(healthPriority(a.Status), healthPriority(b.Status)); p != 0 {
			return p
		}
		return strings.Compare(a.Name, b.Name)
	})

	healthResults := allHealthResults
	if healthProblems {
		problems := make([]RepoHealth, 0, len(healthResults))
		for _, h := range healthResults {
			if h.Status != "passing" {
				problems = append(problems, h)
			}
		}
		healthResults = problems
	}

	summary := summariseHealthResults(len(repoList), len(healthResults), allHealthResults, healthProblems)
	if healthJSON {
		return printHealthJSON(summary, healthResults)
	}

	cli.Print("%s: %d/%d repos healthy (%d%%)\n\n",
		i18n.T("cmd.qa.health.summary"),
		summary.Passing,
		summary.TotalRepos,
		summary.PassingRate)

	if len(healthResults) == 0 {
		cli.Text(i18n.T("cmd.qa.health.all_healthy"))
		return nil
	}

	grouped := make(map[string][]RepoHealth)
	for _, h := range healthResults {
		grouped[h.Status] = append(grouped[h.Status], h)
	}

	printHealthGroup("failing", grouped["failing"], errorStyle)
	printHealthGroup("error", grouped["error"], errorStyle)
	printHealthGroup("pending", grouped["pending"], warningStyle)
	printHealthGroup("no_ci", grouped["no_ci"], dimStyle)
	printHealthGroup("disabled", grouped["disabled"], dimStyle)

	if !healthProblems {
		printHealthGroup("passing", grouped["passing"], successStyle)
	}

	return nil
}

func fetchRepoHealth(org, repoName string) RepoHealth {
	repoFullName := cli.Sprintf("%s/%s", org, repoName)
	args := []string{
		"run", "list",
		"--repo", repoFullName,
		"--limit", "1",
		"--json", "status,conclusion,name,headSha,updatedAt,url",
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "no workflows") || strings.Contains(stderr, "not found") {
				return RepoHealth{
					Name:    repoName,
					Status:  "no_ci",
					Message: i18n.T("cmd.qa.health.no_ci_configured"),
				}
			}
		}
		return RepoHealth{
			Name:    repoName,
			Status:  "error",
			Message: i18n.T("cmd.qa.health.fetch_error"),
		}
	}

	var runs []HealthWorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return RepoHealth{
			Name:    repoName,
			Status:  "error",
			Message: i18n.T("cmd.qa.health.parse_error"),
		}
	}

	if len(runs) == 0 {
		return RepoHealth{
			Name:    repoName,
			Status:  "no_ci",
			Message: i18n.T("cmd.qa.health.no_ci_configured"),
		}
	}

	run := runs[0]
	health := RepoHealth{
		Name: repoName,
		URL:  run.URL,
	}

	switch run.Status {
	case "completed":
		switch run.Conclusion {
		case "success":
			health.Status = "passing"
			health.Message = i18n.T("cmd.qa.health.passing")
		case "failure":
			health.Status = "failing"
			health.Message = i18n.T("cmd.qa.health.tests_failing")
		case "cancelled":
			health.Status = "pending"
			health.Message = i18n.T("cmd.qa.health.cancelled")
		case "skipped":
			health.Status = "passing"
			health.Message = i18n.T("cmd.qa.health.skipped")
		default:
			health.Status = "failing"
			health.Message = run.Conclusion
		}
	case "in_progress", "queued", "waiting":
		health.Status = "pending"
		health.Message = i18n.T("cmd.qa.health.running")
	default:
		health.Status = "no_ci"
		health.Message = run.Status
	}

	return health
}

func healthPriority(status string) int {
	switch status {
	case "failing":
		return 0
	case "error":
		return 1
	case "pending":
		return 2
	case "no_ci":
		return 3
	case "disabled":
		return 4
	case "passing":
		return 5
	default:
		return 6
	}
}

func summariseHealthResults(totalRepos int, filteredRepos int, results []RepoHealth, problemsOnly bool) HealthSummary {
	summary := HealthSummary{
		TotalRepos:    totalRepos,
		FilteredRepos: filteredRepos,
		ByStatus:      make(map[string]int),
		ProblemsOnly:  problemsOnly,
	}

	for _, health := range results {
		summary.ByStatus[health.Status]++
		switch health.Status {
		case "passing":
			summary.Passing++
		case "failing":
			summary.Failing++
		case "error":
			summary.Errors++
		case "pending":
			summary.Pending++
		case "disabled":
			summary.Disabled++
		case "no_ci":
			summary.NotConfigured++
		}
	}

	if summary.TotalRepos > 0 {
		summary.PassingRate = (summary.Passing * 100) / summary.TotalRepos
	}

	return summary
}

func printHealthJSON(summary HealthSummary, repos []RepoHealth) error {
	data, err := json.MarshalIndent(HealthOutput{
		Summary: summary,
		Repos:   repos,
	}, "", "  ")
	if err != nil {
		return err
	}
	cli.Print("%s\n", string(data))
	return nil
}

func printHealthGroup(status string, repos []RepoHealth, style *cli.AnsiStyle) {
	if len(repos) == 0 {
		return
	}

	slices.SortFunc(repos, func(a, b RepoHealth) int {
		return strings.Compare(a.Name, b.Name)
	})

	var label string
	switch status {
	case "failing":
		label = i18n.T("cmd.qa.health.count_failing")
	case "error":
		label = i18n.T("cmd.qa.health.count_error")
	case "pending":
		label = i18n.T("cmd.qa.health.count_pending")
	case "no_ci":
		label = i18n.T("cmd.qa.health.count_no_ci")
	case "disabled":
		label = i18n.T("cmd.qa.health.count_disabled")
	case "passing":
		label = i18n.T("cmd.qa.health.count_passing")
	}

	cli.Print("%s (%d):\n", style.Render(label), len(repos))
	for _, repo := range repos {
		cli.Print("  %s %s\n",
			cli.RepoStyle.Render(repo.Name),
			dimStyle.Render(repo.Message))
		if repo.URL != "" && status == "failing" {
			cli.Print("      -> %s\n", dimStyle.Render(repo.URL))
		}
	}
	cli.Blank()
}
