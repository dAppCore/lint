package qa

import (
	. "dappco.re/go"
)

// TestCountWorkflowRuns_Good_AllCompleteSuccess counts a set of completed,
// successful runs and asserts allComplete is reported with zero pending.
func TestCountWorkflowRuns_Good_AllCompleteSuccess(t *T) {
	runs := []WorkflowRun{
		{Status: "completed", Conclusion: "success"},
		{Status: "completed", Conclusion: "success"},
	}
	counts := countWorkflowRuns(runs)
	AssertTrue(t, counts.allComplete)
	AssertEqual(t, 2, counts.success)
	AssertEqual(t, 0, counts.pending)
	AssertEqual(t, 0, counts.failed)
}

// TestCountWorkflowRuns_Bad_FailedConclusionCounted counts a completed run with
// a non-success conclusion as failed while still complete.
func TestCountWorkflowRuns_Bad_FailedConclusionCounted(t *T) {
	runs := []WorkflowRun{
		{Status: "completed", Conclusion: "failure"},
		{Status: "completed", Conclusion: "success"},
	}
	counts := countWorkflowRuns(runs)
	AssertTrue(t, counts.allComplete)
	AssertEqual(t, 1, counts.failed)
	AssertEqual(t, 1, counts.success)
}

// TestCountWorkflowRuns_Ugly_PendingBlocksComplete asserts an in-progress run
// clears allComplete and increments pending.
func TestCountWorkflowRuns_Ugly_PendingBlocksComplete(t *T) {
	runs := []WorkflowRun{
		{Status: "in_progress"},
		{Status: "completed", Conclusion: "success"},
	}
	counts := countWorkflowRuns(runs)
	AssertFalse(t, counts.allComplete)
	AssertEqual(t, 1, counts.pending)
	AssertEqual(t, 1, counts.success)
}

// TestFormatWorkflowStatus_Good_RendersEachBucket asserts the status line names
// the run total and includes the running, passed and failed counts.
func TestFormatWorkflowStatus_Good_RendersEachBucket(t *T) {
	status := formatWorkflowStatus(3, workflowRunCounts{pending: 1, success: 1, failed: 1})
	AssertContains(t, status, "3 workflow(s)")
	AssertContains(t, status, "1 running")
	AssertContains(t, status, "1 passed")
	AssertContains(t, status, "1 failed")
}

// TestFormatWorkflowStatus_Ugly_OmitsEmptyBuckets asserts buckets with zero
// counts are not rendered.
func TestFormatWorkflowStatus_Ugly_OmitsEmptyBuckets(t *T) {
	status := formatWorkflowStatus(2, workflowRunCounts{success: 2})
	AssertContains(t, status, "2 passed")
	AssertNotContains(t, status, "running")
	AssertNotContains(t, status, "failed")
}

// TestResolveRepo_Good_FullNamePassesThrough asserts an org/name argument is
// returned verbatim without consulting git.
func TestResolveRepo_Good_FullNamePassesThrough(t *T) {
	resolved := resolveRepo("core/lint")
	RequireResultOK(t, resolved)
	AssertEqual(t, "core/lint", resolved.Value.(string))
}

// TestPrintWatchStatus_GoodBadUgly prints only when the status changes and
// echoes back the most recent status string.
func TestPrintWatchStatus_GoodBadUgly(t *T) {
	output := captureStdout(t, func() {
		latest := printWatchStatus("a", "")
		AssertEqual(t, "a", latest)
		// Unchanged status returns the previous value and prints nothing new.
		latest = printWatchStatus("a", "a")
		AssertEqual(t, "a", latest)
		latest = printWatchStatus("b", "a")
		AssertEqual(t, "b", latest)
	})
	AssertContains(t, output, "a")
	AssertContains(t, output, "b")
}

// TestParseGitHubRepo_Good_ParsesSSHURL extracts org/repo from an SSH remote.
func TestParseGitHubRepo_Good_ParsesSSHURL(t *T) {
	result := parseGitHubRepo("git@github.com:dappcore/lint.git")
	RequireResultOK(t, result)
	AssertEqual(t, "dappcore/lint", result.Value.(string))
}

// TestParseGitHubRepo_Ugly_ParsesHTTPSURL extracts org/repo from an HTTPS
// remote, stripping the .git suffix.
func TestParseGitHubRepo_Ugly_ParsesHTTPSURL(t *T) {
	result := parseGitHubRepo("https://github.com/dappcore/lint.git")
	RequireResultOK(t, result)
	AssertEqual(t, "dappcore/lint", result.Value.(string))
}

// TestParseGitHubRepo_Bad_NonGitHubURLFails fails on a URL that is not a GitHub
// remote.
func TestParseGitHubRepo_Bad_NonGitHubURLFails(t *T) {
	result := parseGitHubRepo("https://gitlab.com/foo/bar.git")
	AssertFalse(t, result.OK)
}

// TestResolveCommit_Good_SpecifiedPassesThrough returns the supplied commit
// without consulting git.
func TestResolveCommit_Good_SpecifiedPassesThrough(t *T) {
	result := resolveCommit("abc1234")
	RequireResultOK(t, result)
	AssertEqual(t, "abc1234", result.Value.(string))
}

// TestPrintWatchHeader_GoodBadUgly prints the repo name and an abbreviated
// commit SHA (truncated to 8 chars when longer).
func TestPrintWatchHeader_GoodBadUgly(t *T) {
	output := captureStdout(t, func() {
		printWatchHeader("dappcore/lint", "0123456789abcdef")
	})
	AssertContains(t, output, "dappcore/lint")
	AssertContains(t, output, "01234567")
	AssertNotContains(t, output, "0123456789abcdef")
}
