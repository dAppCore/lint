package lint

import (
	"bytes"
	"regexp"
	"strings"

	coreerr "forge.lthn.ai/core/go-log"
)

// Finding represents a single match of a rule against a source file.
type Finding struct {
	Tool     string `json:"tool,omitempty"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	Category string `json:"category,omitempty"`
	Fix      string `json:"fix,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
	Title    string `json:"title,omitempty"`
	Match    string `json:"match,omitempty"`
	Repo     string `json:"repo,omitempty"`
}

// compiledRule pairs a Rule with its pre-compiled regex patterns.
type compiledRule struct {
	rule    Rule
	pattern *regexp.Regexp
	exclude *regexp.Regexp
}

// Matcher holds compiled rules and performs line-by-line regex matching.
type Matcher struct {
	rules []compiledRule
}

// NewMatcher compiles all regex-detection rules and returns a Matcher.
// Non-regex rules are silently skipped. Returns an error if any regex fails to compile.
func NewMatcher(rules []Rule) (*Matcher, error) {
	var compiled []compiledRule

	for _, r := range rules {
		if r.Detection != "regex" {
			continue
		}

		pat, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, coreerr.E("NewMatcher", "compiling pattern for rule "+r.ID, err)
		}

		var excl *regexp.Regexp
		if r.ExcludePattern != "" {
			excl, err = regexp.Compile(r.ExcludePattern)
			if err != nil {
				return nil, coreerr.E("NewMatcher", "compiling exclude pattern for rule "+r.ID, err)
			}
		}

		compiled = append(compiled, compiledRule{
			rule:    r,
			pattern: pat,
			exclude: excl,
		})
	}

	return &Matcher{rules: compiled}, nil
}

// Match scans the file content line by line against all compiled rules.
// Lines matching a rule's exclude pattern are skipped.
// The filename is also checked against exclude patterns (e.g. _test.go).
func (m *Matcher) Match(filename string, content []byte) []Finding {
	lines := bytes.Split(content, []byte("\n"))
	var findings []Finding

	for _, cr := range m.rules {
		// Check if the filename itself matches the exclude pattern.
		if cr.exclude != nil && cr.exclude.MatchString(filename) {
			continue
		}

		for i, line := range lines {
			lineStr := string(line)

			if !cr.pattern.MatchString(lineStr) {
				continue
			}

			// Skip if the line matches the exclude pattern.
			if cr.exclude != nil && cr.exclude.MatchString(lineStr) {
				continue
			}

			findings = append(findings, Finding{
				RuleID:   cr.rule.ID,
				Title:    cr.rule.Title,
				Severity: cr.rule.Severity,
				File:     filename,
				Line:     i + 1,
				Match:    strings.TrimSpace(lineStr),
				Fix:      cr.rule.Fix,
			})
		}
	}

	return findings
}
