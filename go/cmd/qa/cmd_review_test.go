package qa

import (
	. "dappco.re/go"
)

const (
	cmdReviewTestForgeExamplec3bd82 = "forge/example"
)

func TestRunReviewJSONOutput_PreservesPartialResultsAndFetchErrors(t *T) {
	dir := t.TempDir()
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	reviewRepo = cmdReviewTestForgeExamplec3bd82
	reviewJSON = true

	output := captureStdout(t, func() {
		RequireResultOK(t, runReview())
	})

	var payload reviewOutput
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertTrue(t, payload.ShowingMine)
	AssertTrue(t, payload.ShowingRequested)
	RequireLen(t, payload.Mine, 0)
	RequireLen(t, payload.Requested, 1)
	AssertEqual(t, 42, payload.Requested[0].Number)
	AssertEqual(t, "Refine agent output", payload.Requested[0].Title)
	RequireLen(t, payload.FetchErrors, 1)
	AssertEqual(t, cmdReviewTestForgeExamplec3bd82, payload.FetchErrors[0].Repo)
	AssertEqual(t, "mine", payload.FetchErrors[0].Scope)
	AssertContains(t, payload.FetchErrors[0].Error, "simulated author query failure")
}

func TestRunReviewJSONOutput_ReturnsErrorWhenAllFetchesFail(t *T) {
	dir := t.TempDir()
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	reviewRepo = cmdReviewTestForgeExamplec3bd82
	reviewJSON = true

	var runResult Result
	output := captureStdout(t, func() {
		runResult = runReview()
	})

	RequireResultError(t, runResult)

	var payload reviewOutput
	RequireResultOK(t, JSONUnmarshal([]byte(output), &payload))
	AssertEmpty(t, payload.Mine)
	AssertEmpty(t, payload.Requested)
	RequireLen(t, payload.FetchErrors, 2)
	AssertEqual(t, "mine", payload.FetchErrors[0].Scope)
	AssertEqual(t, "requested", payload.FetchErrors[1].Scope)
}

func TestRunReviewHumanOutput_PreservesSuccessfulSectionWhenOneFetchFails(t *T) {
	dir := t.TempDir()
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	reviewRepo = cmdReviewTestForgeExamplec3bd82

	output := captureStdout(t, func() {
		RequireResultOK(t, runReview())
	})

	AssertContains(t, output, "#42 Refine agent output")
	AssertContains(t, output, "gh pr checkout 42")
	AssertNotContains(t, output, "Your pull requests")
	AssertNotContains(t, output, "cmd.qa.review.no_prs")
}

func TestRunReviewHumanOutput_ReturnsErrorWhenAllFetchesFail(t *T) {
	dir := t.TempDir()
	writeExecutable(t, PathJoin(dir, "gh"), `#!/bin/sh
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

	reviewRepo = cmdReviewTestForgeExamplec3bd82

	var runResult Result
	output := captureStdout(t, func() {
		runResult = runReview()
	})

	RequireResultError(t, runResult)
	AssertNotContains(t, output, "Your pull requests")
	AssertNotContains(t, output, "Review requested")
}

func TestAnalyzePRStatus_UsesDeterministicFailedCheckName(t *T) {
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

	AssertEqual(t, "✗", status)
	AssertEqual(t, "CI failed: Alpha", action)
}

func resetReviewFlags(t *T) {
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
