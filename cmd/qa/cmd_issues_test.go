package qa

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"forge.lthn.ai/core/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunQAIssuesJSONOutput_UsesMachineFriendlyKeys(t *testing.T) {
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
	require.NoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	var payload IssuesOutput
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Equal(t, 1, payload.TotalIssues)
	assert.Equal(t, 1, payload.FilteredIssues)
	require.Len(t, payload.Categories, 4)
	require.Len(t, payload.Categories[0].Issues, 1)

	issue := payload.Categories[0].Issues[0]
	assert.Equal(t, "needs_response", payload.Categories[0].Category)
	assert.Equal(t, "alpha", issue.RepoName)
	assert.Equal(t, 10, issue.Priority)
	assert.Equal(t, "needs_response", issue.Category)
	assert.Equal(t, "@carol cmd.qa.issues.hint.needs_response", issue.ActionHint)
	assert.Contains(t, output, `"repo_name"`)
	assert.Contains(t, output, `"action_hint"`)
	assert.NotContains(t, output, `"RepoName"`)
	assert.NotContains(t, output, `"ActionHint"`)
}

func resetIssuesFlags(t *testing.T) {
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
