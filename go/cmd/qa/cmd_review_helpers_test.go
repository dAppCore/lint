package qa

import (
	. "dappco.re/go"
)

// TestPrintMyPRs_Good_RendersCountAndRows prints the PR count header and one
// row per pull request.
func TestPrintMyPRs_Good_RendersCountAndRows(t *T) {
	output := captureStdout(t, func() {
		RequireResultOK(t, printMyPRs([]PullRequest{
			{Number: 12, Title: "Add widget"},
			{Number: 34, Title: "Fix gadget"},
		}))
	})
	AssertContains(t, output, "(2)")
	AssertContains(t, output, "#12")
	AssertContains(t, output, "#34")
}

// TestPrintMyPRs_Ugly_EmptyRendersNoPRsMessage prints the no-PRs message for an
// empty list and still succeeds.
func TestPrintMyPRs_Ugly_EmptyRendersNoPRsMessage(t *T) {
	output := captureStdout(t, func() {
		RequireResultOK(t, printMyPRs(nil))
	})
	AssertNotEmpty(t, output)
}

// TestPrintPRStatus_Good_RendersNumberAndTitle prints the PR number and a
// truncated title.
func TestPrintPRStatus_Good_RendersNumberAndTitle(t *T) {
	output := captureStdout(t, func() {
		printPRStatus(PullRequest{Number: 7, Title: "Tidy the imports", Mergeable: "MERGEABLE"})
	})
	AssertContains(t, output, "#7")
	AssertContains(t, output, "Tidy the imports")
}
