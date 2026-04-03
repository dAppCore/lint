package qa

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"forge.lthn.ai/core/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunReviewJSONOutput_PreservesPartialResultsAndFetchErrors(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"author:@me"*)
    printf '%s\n' 'simulated author query failure' >&2
    exit 1
    ;;
  *"review-requested:@me"*)
    cat <<'JSON'
[
  {
    "number": 42,
    "title": "Refine agent output",
    "author": {"login": "alice"},
    "state": "OPEN",
    "isDraft": false,
    "mergeable": "MERGEABLE",
    "reviewDecision": "",
    "url": "https://example.com/pull/42",
    "headRefName": "feature/agent-output",
    "createdAt": "2026-03-30T00:00:00Z",
    "updatedAt": "2026-03-30T00:00:00Z",
    "additions": 12,
    "deletions": 3,
    "changedFiles": 2,
    "reviewRequests": {"nodes": []},
    "reviews": []
  }
]
JSON
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetReviewFlags(t)
	t.Cleanup(func() {
		reviewRepo = ""
	})

	parent := &cli.Command{Use: "qa"}
	addReviewCommand(parent)
	command := findSubcommand(t, parent, "review")
	require.NoError(t, command.Flags().Set("repo", "forge/example"))
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	var payload reviewOutput
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.True(t, payload.ShowingMine)
	assert.True(t, payload.ShowingRequested)
	require.Len(t, payload.Mine, 0)
	require.Len(t, payload.Requested, 1)
	assert.Equal(t, 42, payload.Requested[0].Number)
	assert.Equal(t, "Refine agent output", payload.Requested[0].Title)
	require.Len(t, payload.FetchErrors, 1)
	assert.Equal(t, "forge/example", payload.FetchErrors[0].Repo)
	assert.Equal(t, "mine", payload.FetchErrors[0].Scope)
	assert.Contains(t, payload.FetchErrors[0].Error, "simulated author query failure")
}

func TestRunReviewJSONOutput_ReturnsErrorWhenAllFetchesFail(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"author:@me"*)
    printf '%s\n' 'simulated author query failure' >&2
    exit 1
    ;;
  *"review-requested:@me"*)
    printf '%s\n' 'simulated requested query failure' >&2
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
	resetReviewFlags(t)
	t.Cleanup(func() {
		reviewRepo = ""
	})

	parent := &cli.Command{Use: "qa"}
	addReviewCommand(parent)
	command := findSubcommand(t, parent, "review")
	require.NoError(t, command.Flags().Set("repo", "forge/example"))
	require.NoError(t, command.Flags().Set("json", "true"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	require.Error(t, runErr)

	var payload reviewOutput
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Empty(t, payload.Mine)
	assert.Empty(t, payload.Requested)
	require.Len(t, payload.FetchErrors, 2)
	assert.Equal(t, "mine", payload.FetchErrors[0].Scope)
	assert.Equal(t, "requested", payload.FetchErrors[1].Scope)
}

func TestRunReviewHumanOutput_PreservesSuccessfulSectionWhenOneFetchFails(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"author:@me"*)
    printf '%s\n' 'simulated author query failure' >&2
    exit 1
    ;;
  *"review-requested:@me"*)
    cat <<'JSON'
[
  {
    "number": 42,
    "title": "Refine agent output",
    "author": {"login": "alice"},
    "state": "OPEN",
    "isDraft": false,
    "mergeable": "MERGEABLE",
    "reviewDecision": "",
    "url": "https://example.com/pull/42",
    "headRefName": "feature/agent-output",
    "createdAt": "2026-03-30T00:00:00Z",
    "updatedAt": "2026-03-30T00:00:00Z",
    "additions": 12,
    "deletions": 3,
    "changedFiles": 2,
    "reviewRequests": {"nodes": []},
    "reviews": []
  }
]
JSON
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	restoreWorkingDir(t, dir)
	prependPath(t, dir)
	resetReviewFlags(t)
	t.Cleanup(func() {
		reviewRepo = ""
	})

	parent := &cli.Command{Use: "qa"}
	addReviewCommand(parent)
	command := findSubcommand(t, parent, "review")
	require.NoError(t, command.Flags().Set("repo", "forge/example"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Contains(t, output, "#42 Refine agent output")
	assert.Contains(t, output, "gh pr checkout 42")
	assert.NotContains(t, output, "Your pull requests")
	assert.NotContains(t, output, "cmd.qa.review.no_prs")
}

func TestRunReviewHumanOutput_ReturnsErrorWhenAllFetchesFail(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"author:@me"*)
    printf '%s\n' 'simulated author query failure' >&2
    exit 1
    ;;
  *"review-requested:@me"*)
    printf '%s\n' 'simulated requested query failure' >&2
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
	resetReviewFlags(t)
	t.Cleanup(func() {
		reviewRepo = ""
	})

	parent := &cli.Command{Use: "qa"}
	addReviewCommand(parent)
	command := findSubcommand(t, parent, "review")
	require.NoError(t, command.Flags().Set("repo", "forge/example"))

	var runErr error
	output := captureStdout(t, func() {
		runErr = command.RunE(command, nil)
	})

	require.Error(t, runErr)
	assert.NotContains(t, output, "Your pull requests")
	assert.NotContains(t, output, "Review requested")
}

func TestAnalyzePRStatus_UsesDeterministicFailedCheckName(t *testing.T) {
	pr := PullRequest{
		Mergeable:      "MERGEABLE",
		ReviewDecision: "",
		StatusChecks: &StatusCheckRollup{
			Contexts: []StatusContext{
				{State: "FAILURE", Conclusion: "failure", Name: "Zulu"},
				{State: "FAILURE", Conclusion: "failure", Name: "Alpha"},
			},
		},
	}

	status, _, action := analyzePRStatus(pr)

	assert.Equal(t, "✗", status)
	assert.Equal(t, "CI failed: Alpha", action)
}

func resetReviewFlags(t *testing.T) {
	t.Helper()
	oldMine := reviewMine
	oldRequested := reviewRequested
	oldRepo := reviewRepo
	oldJSON := reviewJSON

	reviewMine = false
	reviewRequested = false
	reviewRepo = ""
	reviewJSON = false

	t.Cleanup(func() {
		reviewMine = oldMine
		reviewRequested = oldRequested
		reviewRepo = oldRepo
		reviewJSON = oldJSON
	})
}
