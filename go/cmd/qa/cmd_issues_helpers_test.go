package qa

import (
	. "dappco.re/go"
)

// TestFilterCategory_Good_ReturnsMatchingCategory narrows a categorised map to
// the single requested category when it has issues.
func TestFilterCategory_Good_ReturnsMatchingCategory(t *T) {
	categorised := map[string][]Issue{
		"ready":   {{}, {}},
		"blocked": {{}},
	}
	filtered := filterCategory(categorised, "ready")
	RequireLen(t, filtered, 1)
	RequireLen(t, filtered["ready"], 2)
}

// TestFilterCategory_Bad_MissingCategoryReturnsEmpty returns an empty map when
// the requested category is absent.
func TestFilterCategory_Bad_MissingCategoryReturnsEmpty(t *T) {
	filtered := filterCategory(map[string][]Issue{"ready": {{}}}, "triage")
	RequireLen(t, filtered, 0)
}

// TestFilterCategory_Ugly_PresentButEmptyReturnsEmpty returns an empty map when
// the category exists with no issues.
func TestFilterCategory_Ugly_PresentButEmptyReturnsEmpty(t *T) {
	filtered := filterCategory(map[string][]Issue{"ready": {}}, "ready")
	RequireLen(t, filtered, 0)
}

// TestPrintCategorisedIssues_Good_RendersNonEmptyCategories prints a header with
// a count for each populated category and skips empty ones.
func TestPrintCategorisedIssues_Good_RendersNonEmptyCategories(t *T) {
	output := captureStdout(t, func() {
		printCategorisedIssues(map[string][]Issue{
			"ready":  {{}, {}},
			"triage": {},
		})
	})
	AssertContains(t, output, "(2)")
}
