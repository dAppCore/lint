package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRule() Rule {
	return Rule{
		ID:        "go-sec-001",
		Title:     "SQL wildcard injection in LIKE clauses",
		Severity:  "high",
		Languages: []string{"go"},
		Tags:      []string{"security"},
		Pattern:   `LIKE\s+\?`,
		Fix:       "Use parameterised LIKE with EscapeLike()",
		Detection: "regex",
	}
}

func TestParseRules_Good(t *testing.T) {
	data := []byte(`
- id: go-sec-001
  title: "SQL wildcard injection"
  severity: high
  languages: [go]
  tags: [security]
  pattern: 'LIKE\s+\?'
  fix: "Use parameterised LIKE"
  detection: regex
  auto_fixable: false
- id: go-sec-002
  title: "Path traversal"
  severity: high
  languages: [go]
  tags: [security]
  pattern: 'filepath\.Join'
  fix: "Use securejoin"
  detection: regex
`)
	rules, err := ParseRules(data)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, "go-sec-001", rules[0].ID)
	assert.Equal(t, "go-sec-002", rules[1].ID)
	assert.Equal(t, []string{"go"}, rules[0].Languages)
	assert.False(t, rules[0].AutoFixable)
}

func TestParseRules_Bad_InvalidYAML(t *testing.T) {
	data := []byte(`{{{not yaml at all`)
	_, err := ParseRules(data)
	assert.Error(t, err)
}

func TestParseRules_Bad_EmptyInput(t *testing.T) {
	rules, err := ParseRules([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestValidate_Good(t *testing.T) {
	r := validRule()
	assert.NoError(t, r.Validate())
}

func TestValidate_Good_WithExcludePattern(t *testing.T) {
	r := validRule()
	r.ExcludePattern = `securejoin|ValidatePath`
	assert.NoError(t, r.Validate())
}

func TestValidate_Bad_EmptyID(t *testing.T) {
	r := validRule()
	r.ID = ""
	err := r.Validate()
	assert.ErrorContains(t, err, "id")
}

func TestValidate_Bad_EmptyTitle(t *testing.T) {
	r := validRule()
	r.Title = ""
	err := r.Validate()
	assert.ErrorContains(t, err, "title")
}

func TestValidate_Bad_EmptySeverity(t *testing.T) {
	r := validRule()
	r.Severity = ""
	err := r.Validate()
	assert.ErrorContains(t, err, "severity")
}

func TestValidate_Bad_InvalidSeverity(t *testing.T) {
	r := validRule()
	r.Severity = "catastrophic"
	err := r.Validate()
	assert.ErrorContains(t, err, "severity")
}

func TestValidate_Bad_EmptyLanguages(t *testing.T) {
	r := validRule()
	r.Languages = nil
	err := r.Validate()
	assert.ErrorContains(t, err, "languages")
}

func TestValidate_Bad_EmptyPattern(t *testing.T) {
	r := validRule()
	r.Pattern = ""
	err := r.Validate()
	assert.ErrorContains(t, err, "pattern")
}

func TestValidate_Bad_EmptyDetection(t *testing.T) {
	r := validRule()
	r.Detection = ""
	err := r.Validate()
	assert.ErrorContains(t, err, "detection")
}

func TestValidate_Bad_InvalidRegex(t *testing.T) {
	r := validRule()
	r.Pattern = `[invalid(`
	err := r.Validate()
	assert.ErrorContains(t, err, "pattern")
}

func TestValidate_Bad_InvalidExcludeRegex(t *testing.T) {
	r := validRule()
	r.ExcludePattern = `[invalid(`
	err := r.Validate()
	assert.ErrorContains(t, err, "exclude_pattern")
}

func TestValidate_Good_NonRegexDetection(t *testing.T) {
	r := validRule()
	r.Detection = "ast"
	r.Pattern = "this is not a valid regex [[ but detection is not regex"
	assert.NoError(t, r.Validate())
}
