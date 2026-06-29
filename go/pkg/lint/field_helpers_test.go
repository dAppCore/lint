package lint

import (
	core "dappco.re/go"
)

// TestIntField_Good_AcceptsNumericKinds resolves the first present key across
// int, int64 and float64 (the JSON default numeric kind).
func TestIntField_Good_AcceptsNumericKinds(t *core.T) {
	core.AssertEqual(t, 7, intField(map[string]any{"line": 7}, "line"))
	core.AssertEqual(t, 8, intField(map[string]any{"line": int64(8)}, "line"))
	core.AssertEqual(t, 9, intField(map[string]any{"line": float64(9)}, "line"))
}

// TestIntField_Ugly_ParsesDecimalStringAndPrefersFirstKey parses a decimal
// string and honours key precedence.
func TestIntField_Ugly_ParsesDecimalStringAndPrefersFirstKey(t *core.T) {
	core.AssertEqual(t, 42, intField(map[string]any{"col": "42"}, "col"))
	core.AssertEqual(t, 3, intField(map[string]any{"line_from": 3, "line": 99}, "line_from", "line"))
}

// TestIntField_Bad_MissingOrUnparsableReturnsZero returns zero when no key
// resolves to an int-like value.
func TestIntField_Bad_MissingOrUnparsableReturnsZero(t *core.T) {
	core.AssertEqual(t, 0, intField(map[string]any{}, "line"))
	core.AssertEqual(t, 0, intField(map[string]any{"line": "not-a-number"}, "line"))
	core.AssertEqual(t, 0, intField(map[string]any{"line": true}, "line"))
}

// TestStringField_GoodBadUgly returns the first non-empty string across keys
// and empty when none resolve.
func TestStringField_GoodBadUgly(t *core.T) {
	core.AssertEqual(t, "a.php", stringField(map[string]any{"file_name": "a.php"}, "file_name", "file_path"))
	core.AssertEqual(t, "b.php", stringField(map[string]any{"file_name": "  ", "file_path": "b.php"}, "file_name", "file_path"))
	core.AssertEqual(t, "", stringField(map[string]any{"file_name": 123}, "file_name"))
	core.AssertEqual(t, "", stringField(map[string]any{}, "missing"))
}

// TestGithubAnnotationLevel_GoodBadUgly maps severities to GitHub annotation
// levels with a warning fallback (including the empty severity).
func TestGithubAnnotationLevel_GoodBadUgly(t *core.T) {
	core.AssertEqual(t, "error", githubAnnotationLevel("error"))
	core.AssertEqual(t, "notice", githubAnnotationLevel("info"))
	core.AssertEqual(t, "warning", githubAnnotationLevel("warning"))
	core.AssertEqual(t, "warning", githubAnnotationLevel(""))
	core.AssertEqual(t, "warning", githubAnnotationLevel("weird"))
}
