package qa

import (
	. "dappco.re/go"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"dappco.re/go/cli/pkg/cli"
)

func TestRunQAIssuesJSONOutput_UsesMachineFriendlyKeys(t *T) {
	dir := t.TempDir()
	commentTime := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), fmt.Sprintf(`#!/bin/sh
case "$*" in
  *"api user"*)
    printf '%%s\n' 'alice'
    ;;
  *"issue list --repo forge/alpha"*)
    cat <<JSON
[
  {
    "number": 7,
    "title": "Clarify agent output",
    "state": "OPEN",
    "body": "Explain behaviour",
    "createdAt": "2026-03-30T00:00:00Z",
    "updatedAt": %q,
    "author": {"login": "bob"},
    "assignees": {"nodes": []},
    "labels": {"nodes": [{"name": "agent:ready"}]},
    "comments": {
      "totalCount": 1,
      "nodes": [
        {
          "author": {"login": "carol"},
          "createdAt": %q
        }
      ]
    },
    "url": "https://example.com/issues/7"
  }
]
JSON
    ;;
  *)
    printf '%%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`, updatedAt, commentTime))

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetIssuesFlags(t)
	t.Cleanup(func() {
		issuesRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addIssuesCommand(parent)
	command := findSubcommand(t, parent, "issues")
	RequireNoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	RequireNoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		RequireNoError(t, command.RunE(command, nil))
	})

	var payload IssuesOutput
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
	AssertEqual(t, 1, payload.TotalIssues)
	AssertEqual(t, 1, payload.FilteredIssues)
	RequireLen(t, payload.Categories, 4)
	RequireLen(t, payload.Categories[0].Issues, 1)

	issue := payload.Categories[0].Issues[0]
	AssertEqual(t, "needs_response", payload.Categories[0].Category)
	AssertEqual(t, "alpha", issue.RepoName)
	AssertEqual(t, 10, issue.Priority)
	AssertEqual(t, "needs_response", issue.Category)
	AssertEqual(t, "@carol awaiting response", issue.ActionHint)
	AssertContains(t, output, `"repo_name"`)
	AssertContains(t, output, `"action_hint"`)
	AssertNotContains(t, output, `"RepoName"`)
	AssertNotContains(t, output, `"ActionHint"`)
}

func TestRunQAIssuesJSONOutput_SortsFetchErrorsByRepoName(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  beta:
    type: module
  alpha:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"issue list --repo forge/alpha"*)
    printf '%s\n' 'alpha failed' >&2
    exit 1
    ;;
  *"issue list --repo forge/beta"*)
    printf '%s\n' 'beta failed' >&2
    exit 1
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetIssuesFlags(t)
	t.Cleanup(func() {
		issuesRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addIssuesCommand(parent)
	command := findSubcommand(t, parent, "issues")
	RequireNoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	RequireNoError(t, command.Flags().Set("json", "true"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	RequireError(t, runErr)
	var payload IssuesOutput
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
	RequireLen(t, payload.FetchErrors, 2)
	AssertEqual(t, "alpha", payload.FetchErrors[0].Repo)
	AssertEqual(t, "beta", payload.FetchErrors[1].Repo)
}

func TestRunQAIssuesJSONOutput_ReturnsErrorWhenAllFetchesFail(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  beta:
    type: module
  alpha:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"issue list --repo forge/alpha"*)
    printf '%s\n' 'alpha failed' >&2
    exit 1
    ;;
  *"issue list --repo forge/beta"*)
    printf '%s\n' 'beta failed' >&2
    exit 1
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetIssuesFlags(t)
	t.Cleanup(func() {
		issuesRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addIssuesCommand(parent)
	command := findSubcommand(t, parent, "issues")
	RequireNoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	RequireNoError(t, command.Flags().Set("json", "true"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	RequireError(t, runErr)

	var payload IssuesOutput
	RequireNoError(t, json.Unmarshal([]byte(output), &payload))
	RequireLen(t, payload.Categories, 4)
	AssertEmpty(t, payload.Categories[0].Issues)
	RequireLen(t, payload.FetchErrors, 2)
	AssertEqual(t, "alpha", payload.FetchErrors[0].Repo)
	AssertEqual(t, "beta", payload.FetchErrors[1].Repo)
}

func TestRunQAIssuesHumanOutput_ReturnsErrorWhenAllFetchesFail(t *T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  beta:
    type: module
  alpha:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"issue list --repo forge/alpha"*)
    printf '%s\n' 'alpha failed' >&2
    exit 1
    ;;
  *"issue list --repo forge/beta"*)
    printf '%s\n' 'beta failed' >&2
    exit 1
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetIssuesFlags(t)
	t.Cleanup(func() {
		issuesRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addIssuesCommand(parent)
	command := findSubcommand(t, parent, "issues")
	RequireNoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	RequireError(t, runErr)
	AssertNotContains(t, output, "cmd.qa.issues.no_issues")
}

func TestCalculatePriority_UsesMostUrgentLabelRegardlessOfOrder(t *T) {
	labelsA := []string{"low", "critical"}
	labelsB := []string{"critical", "low"}

	AssertEqual(t, 1, calculatePriority(labelsA))
	AssertEqual(t, 1, calculatePriority(labelsB))
}

func TestPrintTriagedIssue_SortsImportantLabels(t *T) {
	var issue Issue
	RequireNoError(t, json.Unmarshal([]byte(`{
		"number": 7,
		"title": "Stabilise output",
		"updatedAt": "2026-03-30T00:00:00Z",
		"labels": {
			"nodes": [
				{"name": "priority:urgent"},
				{"name": "agent:ready"}
			]
		}
	}`), &issue))
	issue.RepoName = "alpha"

	output := captureStdout(t, func() {
		printTriagedIssue(issue)
	})

	AssertContains(t, output, "[agent:ready, priority:urgent]")
	AssertNotContains(t, output, "[priority:urgent, agent:ready]")
}

func resetIssuesFlags(t *T) {
	t.Helper()
	oldMine := issuesMine
	oldTriage := issuesTriage
	oldBlocked := issuesBlocked
	oldRegistry := issuesRegistry
	oldLimit := issuesLimit
	oldJSON := issuesJSON

	issuesMine = false
	issuesTriage = false
	issuesBlocked = false
	issuesRegistry = ""
	issuesLimit = 50
	issuesJSON = false

	t.Cleanup(func() {
		issuesMine = oldMine
		issuesTriage = oldTriage
		issuesBlocked = oldBlocked
		issuesRegistry = oldRegistry
		issuesLimit = oldLimit
		issuesJSON = oldJSON
	})
}
