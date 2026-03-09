package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMatcher_Good(t *testing.T) {
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
	require.NoError(t, err)
	assert.NotNil(t, m)
}

func TestNewMatcher_Bad_InvalidRegex(t *testing.T) {
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
	assert.Error(t, err)
	assert.ErrorContains(t, err, "test-bad")
}

func TestMatch_Good_Found(t *testing.T) {
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
	require.NoError(t, err)

	content := []byte("package main\n\n// TODO: fix this later\nfunc main() {}\n")
	findings := m.Match("main.go", content)

	require.Len(t, findings, 1)
	assert.Equal(t, "test-001", findings[0].RuleID)
	assert.Equal(t, "Found a TODO", findings[0].Title)
	assert.Equal(t, "medium", findings[0].Severity)
	assert.Equal(t, "main.go", findings[0].File)
	assert.Equal(t, 3, findings[0].Line)
	assert.Contains(t, findings[0].Match, "TODO")
	assert.Equal(t, "Remove TODO comments", findings[0].Fix)
}

func TestMatch_Good_ExcludePattern(t *testing.T) {
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
	require.NoError(t, err)

	// Should not match in test files (filename-based exclude).
	content := []byte("panic(\"oh no\")\n")
	findings := m.Match("foo_test.go", content)
	assert.Empty(t, findings)

	// Should not match lines with exclude pattern.
	content2 := []byte("panic(\"unreachable\") // unreachable\n")
	findings2 := m.Match("foo.go", content2)
	assert.Empty(t, findings2)

	// Should match lines without exclude pattern in non-test files.
	content3 := []byte("func fail() {\n\tpanic(\"boom\")\n}\n")
	findings3 := m.Match("foo.go", content3)
	require.Len(t, findings3, 1)
	assert.Equal(t, 2, findings3[0].Line)
}

func TestMatch_Good_NoMatch(t *testing.T) {
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
	require.NoError(t, err)

	content := []byte("package main\n\nfunc main() {}\n")
	findings := m.Match("main.go", content)
	assert.Empty(t, findings)
}

func TestMatch_Good_MultipleRules(t *testing.T) {
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
	require.NoError(t, err)

	content := []byte("// TODO: something\n// FIXME: something else\n")
	findings := m.Match("main.go", content)
	assert.Len(t, findings, 2)
}

func TestMatch_Good_MultipleMatchesSameRule(t *testing.T) {
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
	require.NoError(t, err)

	content := []byte("// TODO: first\n// TODO: second\n")
	findings := m.Match("main.go", content)
	assert.Len(t, findings, 2)
	assert.Equal(t, 1, findings[0].Line)
	assert.Equal(t, 2, findings[1].Line)
}

func TestNewMatcher_Good_SkipsNonRegex(t *testing.T) {
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
	require.NoError(t, err)

	content := []byte("not a regex match\n")
	findings := m.Match("main.go", content)
	assert.Empty(t, findings) // AST rules are not matched by regex matcher.
}
