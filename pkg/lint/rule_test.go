package lint

import (
	core "dappco.re/go"
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

func TestParseRules_Good(t *core.T) {
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
	core.RequireNoError(t, err)
	core.AssertLen(t, rules, 2)
	core.AssertEqual(t, "go-sec-001", rules[0].ID)
	core.AssertEqual(t, "go-sec-002", rules[1].ID)
	core.AssertEqual(t, []string{"go"}, rules[0].Languages)
	core.AssertFalse(t, rules[0].AutoFixable)
}

func TestParseRules_Bad_InvalidYAML(t *core.T) {
	data := []byte(`{{{not yaml at all`)
	_, err := ParseRules(data)
	core.AssertError(t, err)
}

func TestParseRules_Bad_EmptyInput(t *core.T) {
	rules, err := ParseRules([]byte(""))
	core.RequireNoError(t, err)
	core.AssertEmpty(t, rules)
}

func TestValidate_Good(t *core.T) {
	r := validRule()
	core.AssertNoError(t, r.Validate())
}

func TestValidate_Good_WithExcludePattern(t *core.T) {
	r := validRule()
	r.ExcludePattern = `securejoin|ValidatePath`
	core.AssertNoError(t, r.Validate())
}

func TestValidate_Bad_EmptyID(t *core.T) {
	r := validRule()
	r.ID = ""
	err := r.Validate()
	core.AssertError(t, err, "id")
}

func TestValidate_Bad_EmptyTitle(t *core.T) {
	r := validRule()
	r.Title = ""
	err := r.Validate()
	core.AssertError(t, err, "title")
}

func TestValidate_Bad_EmptySeverity(t *core.T) {
	r := validRule()
	r.Severity = ""
	err := r.Validate()
	core.AssertError(t, err, "severity")
}

func TestValidate_Bad_InvalidSeverity(t *core.T) {
	r := validRule()
	r.Severity = "catastrophic"
	err := r.Validate()
	core.AssertError(t, err, "severity")
}

func TestValidate_Bad_EmptyLanguages(t *core.T) {
	r := validRule()
	r.Languages = nil
	err := r.Validate()
	core.AssertError(t, err, "languages")
}

func TestValidate_Bad_EmptyPattern(t *core.T) {
	r := validRule()
	r.Pattern = ""
	err := r.Validate()
	core.AssertError(t, err, "pattern")
}

func TestValidate_Bad_EmptyDetection(t *core.T) {
	r := validRule()
	r.Detection = ""
	err := r.Validate()
	core.AssertError(t, err, "detection")
}

func TestValidate_Bad_InvalidRegex(t *core.T) {
	r := validRule()
	r.Pattern = `[invalid(`
	err := r.Validate()
	core.AssertError(t, err, "pattern")
}

func TestValidate_Bad_InvalidExcludeRegex(t *core.T) {
	r := validRule()
	r.ExcludePattern = `[invalid(`
	err := r.Validate()
	core.AssertError(t, err, "exclude_pattern")
}

func TestValidate_Good_NonRegexDetection(t *core.T) {
	r := validRule()
	r.Detection = "ast"
	r.Pattern = "this is not a valid regex [[ but detection is not regex"
	core.AssertNoError(t, r.Validate())
}
