package lint

import (
	core "dappco.re/go"
)

func TestNewMatcher_Good(t *core.T) {
	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Test rule",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)
	core.AssertNotNil(t, m)
}

func TestNewMatcher_Bad_InvalidRegex(t *core.T) {
	rules := []Rule{
		{
			ID:        "test-bad",
			Title:     "Bad regex",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   `[invalid(`,
			Fix:       "Fix it",
			Detection: "regex",
		},
	}

	_, err := NewMatcher(rules)
	core.AssertError(t, err)
	core.AssertError(t, err, "test-bad")
}

func TestMatch_Good_Found(t *core.T) {
	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "medium",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO comments",
			Detection: "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	content := []byte("package main\n\n// TODO: fix this later\nfunc main() {}\n")
	findings := m.Match("main.go", content)

	RequireLen(t, findings, 1)
	core.AssertEqual(t, "test-001", findings[0].RuleID)
	core.AssertEqual(t, "Found a TODO", findings[0].Title)
	core.AssertEqual(t, "medium", findings[0].Severity)
	core.AssertEqual(t, "main.go", findings[0].File)
	core.AssertEqual(t, 3, findings[0].Line)
	core.AssertContains(t, findings[0].Match, "TODO")
	core.AssertEqual(t, "Remove TODO comments", findings[0].Fix)
}

func TestMatch_Good_ExcludePattern(t *core.T) {
	rules := []Rule{
		{
			ID:             "test-002",
			Title:          "Panic in library code",
			Severity:       "high",
			Languages:      []string{"go"},
			Pattern:        `\bpanic\(`,
			ExcludePattern: `_test\.go|// unreachable`,
			Fix:            "Return an error instead",
			Detection:      "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	// Should not match in test files (filename-based exclude).
	content := []byte("panic(\"oh no\")\n")
	findings := m.Match("foo_test.go", content)
	core.AssertEmpty(t, findings)

	// Should not match lines with exclude pattern.
	content2 := []byte("panic(\"unreachable\") // unreachable\n")
	findings2 := m.Match("foo.go", content2)
	core.AssertEmpty(t, findings2)

	// Should match lines without exclude pattern in non-test files.
	content3 := []byte("func fail() {\n\tpanic(\"boom\")\n}\n")
	findings3 := m.Match("foo.go", content3)
	RequireLen(t, findings3, 1)
	core.AssertEqual(t, 2, findings3[0].Line)
}

func TestMatch_Good_NoMatch(t *core.T) {
	rules := []Rule{
		{
			ID:        "test-001",
			Title:     "Found a TODO",
			Severity:  "medium",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	content := []byte("package main\n\nfunc main() {}\n")
	findings := m.Match("main.go", content)
	core.AssertEmpty(t, findings)
}

func TestMatch_Good_MultipleRules(t *core.T) {
	rules := []Rule{
		{
			ID:        "rule-a",
			Title:     "Rule A",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Fix A",
			Detection: "regex",
		},
		{
			ID:        "rule-b",
			Title:     "Rule B",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   `FIXME`,
			Fix:       "Fix B",
			Detection: "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	content := []byte("// TODO: something\n// FIXME: something else\n")
	findings := m.Match("main.go", content)
	core.AssertLen(t, findings, 2)
}

func TestMatch_Good_MultipleMatchesSameRule(t *core.T) {
	rules := []Rule{
		{
			ID:        "rule-a",
			Title:     "Rule A",
			Severity:  "low",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Fix A",
			Detection: "regex",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	content := []byte("// TODO: first\n// TODO: second\n")
	findings := m.Match("main.go", content)
	core.AssertLen(t, findings, 2)
	core.AssertEqual(t, 1, findings[0].Line)
	core.AssertEqual(t, 2, findings[1].Line)
}

func TestNewMatcher_Good_SkipsNonRegex(t *core.T) {
	rules := []Rule{
		{
			ID:        "ast-rule",
			Title:     "AST rule",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   "not a regex",
			Fix:       "N/A",
			Detection: "ast",
		},
	}

	m, err := NewMatcher(rules)
	core.RequireNoError(t, err)

	content := []byte("not a regex match\n")
	findings := m.Match("main.go", content)
	core.AssertEmpty(t, findings) // AST rules are not matched by regex matcher.
}
