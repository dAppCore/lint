package qa

import (
	. "dappco.re/go"
)

const (
	cmdHealthTestReposYaml3b1ae7 = "repos.yaml"
)

func TestRunHealthJSONOutput_UsesMachineFriendlyKeysAndKeepsFetchErrors(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdHealthTestReposYaml3b1ae7), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	healthRegistry = PathJoin(dir, cmdHealthTestReposYaml3b1ae7)
	healthJSON = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runHealth())
	})

	var payload HealthOutput
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertEqual(t, 2, payload.Summary.TotalRepos)
	AssertEqual(t, 1, payload.Summary.Passing)
	AssertEqual(t, 1, payload.Summary.Errors)
	AssertEqual(t, 2, payload.Summary.FilteredRepos)
	AssertLen(t, payload.Summary.ByStatus, 6)
	AssertEqual(t, 1, payload.Summary.ByStatus["passing"])
	AssertEqual(t, 1, payload.Summary.ByStatus["error"])
	AssertEqual(t, 0, payload.Summary.ByStatus["pending"])
	AssertEqual(t, 0, payload.Summary.ByStatus["disabled"])
	AssertEqual(t, 0, payload.Summary.ByStatus["no_ci"])
	RequireLen(t, payload.Repos, 2)
	AssertEqual(t, "error", payload.Repos[0].Status)
	AssertEqual(t, "beta", payload.Repos[0].Name)
	AssertEqual(t, "passing", payload.Repos[1].Status)
	AssertEqual(t, "alpha", payload.Repos[1].Name)
	AssertContains(t, output, `"status"`)
	AssertNotContains(t, output, `"Status"`)
	AssertNotContains(t, output, `"FailingSince"`)
}

func TestRunHealthJSONOutput_ProblemsOnlyKeepsOverallSummary(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdHealthTestReposYaml3b1ae7), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	healthRegistry = PathJoin(dir, cmdHealthTestReposYaml3b1ae7)
	healthJSON = true
	healthProblems = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runHealth())
	})

	var payload HealthOutput
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertEqual(t, 2, payload.Summary.TotalRepos)
	AssertEqual(t, 1, payload.Summary.Passing)
	AssertEqual(t, 1, payload.Summary.Errors)
	AssertEqual(t, 1, payload.Summary.FilteredRepos)
	AssertTrue(t, payload.Summary.ProblemsOnly)
	AssertLen(t, payload.Summary.ByStatus, 6)
	AssertEqual(t, 1, payload.Summary.ByStatus["passing"])
	AssertEqual(t, 1, payload.Summary.ByStatus["error"])
	AssertEqual(t, 0, payload.Summary.ByStatus["pending"])
	AssertEqual(t, 0, payload.Summary.ByStatus["disabled"])
	AssertEqual(t, 0, payload.Summary.ByStatus["no_ci"])
	RequireLen(t, payload.Repos, 1)
	AssertEqual(t, "error", payload.Repos[0].Status)
	AssertEqual(t, "beta", payload.Repos[0].Name)
}

func TestRunHealthHumanOutput_ShowsFetchErrorsAsErrors(t *T) {
	dir := t.TempDir()
	writeTestFile(t, PathJoin(dir, cmdHealthTestReposYaml3b1ae7), `version: 1
org: forge
base_path: .
repos:
  alpha:
    type: module
  beta:
    type: module
`)
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	healthRegistry = PathJoin(dir, cmdHealthTestReposYaml3b1ae7)

	output := captureStdout(t, func() {
		RequireResultOK(t, runHealth())
	})

	AssertContains(t, output, "CI Health")
	AssertContains(t, output, "alpha")
	AssertContains(t, output, "beta")
	AssertContains(t, output, "Failed to fetch workflow status")
	AssertNotContains(t, output, "no CI")
}

func resetHealthFlags(t *T) {
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
