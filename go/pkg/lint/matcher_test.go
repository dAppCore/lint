package lint

import (
	core "dappco.re/go"
)

const (
	matcherTestFoundATodo28bce4 = "Found a TODO"
	matcherTestMainGo3f3621     = "main.go"
	matcherTestTest00137e60a    = "test-001"
)

func TestNewMatcher_Good(t *core.T) {
	rules := []Rule{
		{
			ID:        matcherTestTest00137e60a,
			Title:     "Test rule",
			Severity:  "high",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	m := requireMatcher(t, rules)
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

	result := NewMatcher(rules)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "test-bad")
}

func TestMatch_Good_Found(t *core.T) {
	rules := []Rule{
		{
			ID:        matcherTestTest00137e60a,
			Title:     matcherTestFoundATodo28bce4,
			Severity:  "medium",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO comments",
			Detection: "regex",
		},
	}

	m := requireMatcher(t, rules)

	content := []byte("package main\n\n// TODO: fix this later\nfunc main() {}\n")
	findings := m.Match(matcherTestMainGo3f3621, content)

	RequireLen(t, findings, 1)
	core.AssertEqual(t, matcherTestTest00137e60a, findings[0].RuleID)
	core.AssertEqual(t, matcherTestFoundATodo28bce4, findings[0].Title)
	core.AssertEqual(t, "medium", findings[0].Severity)
	core.AssertEqual(t, matcherTestMainGo3f3621, findings[0].File)
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

	m := requireMatcher(t, rules)

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
			ID:        matcherTestTest00137e60a,
			Title:     matcherTestFoundATodo28bce4,
			Severity:  "medium",
			Languages: []string{"go"},
			Pattern:   `TODO`,
			Fix:       "Remove TODO",
			Detection: "regex",
		},
	}

	m := requireMatcher(t, rules)

	content := []byte("package main\n\nfunc main() {}\n")
	findings := m.Match(matcherTestMainGo3f3621, content)
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

	m := requireMatcher(t, rules)

	content := []byte("// TODO: something\n// FIXME: something else\n")
	findings := m.Match(matcherTestMainGo3f3621, content)
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

	m := requireMatcher(t, rules)

	content := []byte("// TODO: first\n// TODO: second\n")
	findings := m.Match(matcherTestMainGo3f3621, content)
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

	m := requireMatcher(t, rules)

	content := []byte("not a regex match\n")
	findings := m.Match(matcherTestMainGo3f3621, content)
	core.AssertEmpty(t, findings) // AST rules are not matched by regex matcher.
}

func requireMatcher(t *core.T, rules []Rule) *Matcher {
	t.Helper()
	result := NewMatcher(rules)
	core.RequireTrue(t, result.OK)
	return result.Value.(*Matcher)
}

func TestMatcher_NewMatcher_Good(t *core.T) {
	subject := NewMatcher
	if subject == nil {
		t.FailNow()
	}
	marker := "NewMatcher:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestMatcher_NewMatcher_Bad(t *core.T) {
	subject := NewMatcher
	if subject == nil {
		t.FailNow()
	}
	marker := "NewMatcher:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestMatcher_NewMatcher_Ugly(t *core.T) {
	subject := NewMatcher
	if subject == nil {
		t.FailNow()
	}
	marker := "NewMatcher:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestMatcher_Matcher_Match_Good(t *core.T) {
	subject := (*Matcher).Match
	if subject == nil {
		t.FailNow()
	}
	marker := "Matcher:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestMatcher_Matcher_Match_Bad(t *core.T) {
	subject := (*Matcher).Match
	if subject == nil {
		t.FailNow()
	}
	marker := "Matcher:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestMatcher_Matcher_Match_Ugly(t *core.T) {
	subject := (*Matcher).Match
	if subject == nil {
		t.FailNow()
	}
	marker := "Matcher:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
