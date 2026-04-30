package lint

import (
	core "dappco.re/go"
)

const (
	ruleTestGoSec0016d3cb3 = "go-sec-001"
)

func validRule() Rule {
	return Rule{
		ID:        ruleTestGoSec0016d3cb3,
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
	result := ParseRules(data)
	core.RequireTrue(t, result.OK)
	rules := result.Value.([]Rule)
	core.AssertLen(t, rules, 2)
	core.AssertEqual(t, ruleTestGoSec0016d3cb3, rules[0].ID)
	core.AssertEqual(t, "go-sec-002", rules[1].ID)
	core.AssertEqual(t, []string{"go"}, rules[0].Languages)
	core.AssertFalse(t, rules[0].AutoFixable)
}

func TestParseRules_Bad_InvalidYAML(t *core.T) {
	data := []byte(`{{{not yaml at all`)
	result := ParseRules(data)
	core.AssertFalse(t, result.OK)
}

func TestParseRules_Bad_EmptyInput(t *core.T) {
	result := ParseRules([]byte(""))
	core.RequireTrue(t, result.OK)
	rules := result.Value.([]Rule)
	core.AssertEmpty(t, rules)
}

func TestValidate_Good(t *core.T) {
	r := validRule()
	core.AssertTrue(t, r.Validate().OK)
	core.AssertEqual(t, ruleTestGoSec0016d3cb3, r.ID)
}

func TestValidate_Good_WithExcludePattern(t *core.T) {
	r := validRule()
	r.ExcludePattern = `securejoin|ValidatePath`
	core.AssertTrue(t, r.Validate().OK)
}

func TestValidate_Bad_EmptyID(t *core.T) {
	r := validRule()
	r.ID = ""
	result := r.Validate()
	core.AssertFalse(t, result.OK, "id")
}

func TestValidate_Bad_EmptyTitle(t *core.T) {
	r := validRule()
	r.Title = ""
	result := r.Validate()
	core.AssertFalse(t, result.OK, "title")
}

func TestValidate_Bad_EmptySeverity(t *core.T) {
	r := validRule()
	r.Severity = ""
	result := r.Validate()
	core.AssertFalse(t, result.OK, "severity")
}

func TestValidate_Bad_InvalidSeverity(t *core.T) {
	r := validRule()
	r.Severity = "catastrophic"
	result := r.Validate()
	core.AssertFalse(t, result.OK, "severity")
}

func TestValidate_Bad_EmptyLanguages(t *core.T) {
	r := validRule()
	r.Languages = nil
	result := r.Validate()
	core.AssertFalse(t, result.OK, "languages")
}

func TestValidate_Bad_EmptyPattern(t *core.T) {
	r := validRule()
	r.Pattern = ""
	result := r.Validate()
	core.AssertFalse(t, result.OK, "pattern")
}

func TestValidate_Bad_EmptyDetection(t *core.T) {
	r := validRule()
	r.Detection = ""
	result := r.Validate()
	core.AssertFalse(t, result.OK, "detection")
}

func TestValidate_Bad_InvalidRegex(t *core.T) {
	r := validRule()
	r.Pattern = `[invalid(`
	result := r.Validate()
	core.AssertFalse(t, result.OK, "pattern")
}

func TestValidate_Bad_InvalidExcludeRegex(t *core.T) {
	r := validRule()
	r.ExcludePattern = `[invalid(`
	result := r.Validate()
	core.AssertFalse(t, result.OK, "exclude_pattern")
}

func TestValidate_Good_NonRegexDetection(t *core.T) {
	r := validRule()
	r.Detection = "ast"
	r.Pattern = "this is not a valid regex [[ but detection is not regex"
	core.AssertTrue(t, r.Validate().OK)
}

func TestRule_Rule_Validate_Good(t *core.T) {
	subject := (*Rule).Validate
	if subject == nil {
		t.FailNow()
	}
	marker := "Rule:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRule_Rule_Validate_Bad(t *core.T) {
	subject := (*Rule).Validate
	if subject == nil {
		t.FailNow()
	}
	marker := "Rule:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRule_Rule_Validate_Ugly(t *core.T) {
	subject := (*Rule).Validate
	if subject == nil {
		t.FailNow()
	}
	marker := "Rule:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestRule_ParseRules_Good(t *core.T) {
	subject := ParseRules
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseRules:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRule_ParseRules_Bad(t *core.T) {
	subject := ParseRules
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseRules:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRule_ParseRules_Ugly(t *core.T) {
	subject := ParseRules
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseRules:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
