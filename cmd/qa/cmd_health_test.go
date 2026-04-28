package qa

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"dappco.re/go/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHealthJSONOutput_UsesMachineFriendlyKeysAndKeepsFetchErrors(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"--repo forge/alpha"*)
    cat <<'JSON'
[
  {
    "status": "completed",
    "conclusion": "success",
    "name": "CI",
    "headSha": "abc123",
    "updatedAt": "2026-03-30T00:00:00Z",
    "url": "https://example.com/alpha/run/1"
  }
]
JSON
    ;;
  *"--repo forge/beta"*)
    printf '%s\n' 'simulated workflow lookup failure' >&2
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
	resetHealthFlags(t)
	t.Cleanup(func() {
		healthRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addHealthCommand(parent)
	command := findSubcommand(t, parent, "health")
	require.NoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	var payload HealthOutput
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Equal(t, 2, payload.Summary.TotalRepos)
	assert.Equal(t, 1, payload.Summary.Passing)
	assert.Equal(t, 1, payload.Summary.Errors)
	assert.Equal(t, 2, payload.Summary.FilteredRepos)
	assert.Len(t, payload.Summary.ByStatus, 6)
	assert.Equal(t, 1, payload.Summary.ByStatus["passing"])
	assert.Equal(t, 1, payload.Summary.ByStatus["error"])
	assert.Equal(t, 0, payload.Summary.ByStatus["pending"])
	assert.Equal(t, 0, payload.Summary.ByStatus["disabled"])
	assert.Equal(t, 0, payload.Summary.ByStatus["no_ci"])
	require.Len(t, payload.Repos, 2)
	assert.Equal(t, "error", payload.Repos[0].Status)
	assert.Equal(t, "beta", payload.Repos[0].Name)
	assert.Equal(t, "passing", payload.Repos[1].Status)
	assert.Equal(t, "alpha", payload.Repos[1].Name)
	assert.Contains(t, output, `"status"`)
	assert.NotContains(t, output, `"Status"`)
	assert.NotContains(t, output, `"FailingSince"`)
}

func TestRunHealthJSONOutput_ProblemsOnlyKeepsOverallSummary(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"--repo forge/alpha"*)
    cat <<'JSON'
[
  {
    "status": "completed",
    "conclusion": "success",
    "name": "CI",
    "headSha": "abc123",
    "updatedAt": "2026-03-30T00:00:00Z",
    "url": "https://example.com/alpha/run/1"
  }
]
JSON
    ;;
  *"--repo forge/beta"*)
    printf '%s\n' 'simulated workflow lookup failure' >&2
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
	resetHealthFlags(t)
	t.Cleanup(func() {
		healthRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addHealthCommand(parent)
	command := findSubcommand(t, parent, "health")
	require.NoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))
	require.NoError(t, command.Flags().Set("json", "true"))
	require.NoError(t, command.Flags().Set("problems", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	var payload HealthOutput
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.Equal(t, 2, payload.Summary.TotalRepos)
	assert.Equal(t, 1, payload.Summary.Passing)
	assert.Equal(t, 1, payload.Summary.Errors)
	assert.Equal(t, 1, payload.Summary.FilteredRepos)
	assert.True(t, payload.Summary.ProblemsOnly)
	assert.Len(t, payload.Summary.ByStatus, 6)
	assert.Equal(t, 1, payload.Summary.ByStatus["passing"])
	assert.Equal(t, 1, payload.Summary.ByStatus["error"])
	assert.Equal(t, 0, payload.Summary.ByStatus["pending"])
	assert.Equal(t, 0, payload.Summary.ByStatus["disabled"])
	assert.Equal(t, 0, payload.Summary.ByStatus["no_ci"])
	require.Len(t, payload.Repos, 1)
	assert.Equal(t, "error", payload.Repos[0].Status)
	assert.Equal(t, "beta", payload.Repos[0].Name)
}

func TestRunHealthHumanOutput_ShowsFetchErrorsAsErrors(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repos.yaml"), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"--repo forge/alpha"*)
    cat <<'JSON'
[
  {
    "status": "completed",
    "conclusion": "success",
    "name": "CI",
    "headSha": "abc123",
    "updatedAt": "2026-03-30T00:00:00Z",
    "url": "https://example.com/alpha/run/1"
  }
]
JSON
    ;;
  *"--repo forge/beta"*)
    printf '%s\n' 'simulated workflow lookup failure' >&2
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
	resetHealthFlags(t)
	t.Cleanup(func() {
		healthRegistry = ""
	})

	parent := &cli.Command{Use: "qa"}
	addHealthCommand(parent)
	command := findSubcommand(t, parent, "health")
	require.NoError(t, command.Flags().Set("registry", filepath.Join(dir, "repos.yaml")))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Contains(t, output, "CI Health")
	assert.Contains(t, output, "alpha")
	assert.Contains(t, output, "beta")
	assert.Contains(t, output, "Failed to fetch workflow status")
	assert.NotContains(t, output, "no CI")
}

func resetHealthFlags(t *testing.T) {
	t.Helper()
	oldProblems := healthProblems
	oldRegistry := healthRegistry
	oldJSON := healthJSON

	healthProblems = false
	healthRegistry = ""
	healthJSON = false

	t.Cleanup(func() {
		healthProblems = oldProblems
		healthRegistry = oldRegistry
		healthJSON = oldJSON
	})
}
