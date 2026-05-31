package qa

import (
	. "dappco.re/go"
)

// TestQaFailure_GoodBadUgly wraps a nil error as an OK result and a non-nil
// error as a failed result.
func TestQaFailure_GoodBadUgly(t *T) {
	AssertTrue(t, qaFailure(nil).OK)
	failed := qaFailure(Errorf("boom"))
	AssertFalse(t, failed.OK)
	AssertContains(t, failed.Error(), "boom")
}

// TestQaLabel_Good_RepoIsLiteral returns the literal "Repo" label for the repo
// key (it has no i18n entry).
func TestQaLabel_Good_RepoIsLiteral(t *T) {
	AssertEqual(t, "Repo", qaLabel("repo"))
}

// TestQaLabel_Ugly_UnknownKeyFallsBackToText returns the resolved i18n text (or
// the key itself) for a non-repo label.
func TestQaLabel_Ugly_UnknownKeyFallsBackToText(t *T) {
	AssertNotEmpty(t, qaLabel("success"))
}
