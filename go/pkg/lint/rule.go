package lint

import (
	"regexp"
	"slices"

	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

const (
	ruleRuleValidate1b496e = "Rule.Validate"
)

// validSeverities defines the allowed severity levels, ordered from lowest to highest.
var validSeverities = []string{"info", "low", "medium", "high", "critical"}

// Rule represents a single lint rule loaded from a YAML catalog file.
type Rule struct {
	ID             string   `yaml:"id"              json:"id"`
	Title          string   `yaml:"title"           json:"title"`
	Severity       string   `yaml:"severity"        json:"severity"`
	Languages      []string `yaml:"languages"       json:"languages"`
	Tags           []string `yaml:"tags"            json:"tags"`
	Pattern        string   `yaml:"pattern"         json:"pattern"`
	ExcludePattern string   `yaml:"exclude_pattern" json:"exclude_pattern,omitempty"`
	Fix            string   `yaml:"fix"             json:"fix"`
	FoundIn        []string `yaml:"found_in"        json:"found_in,omitempty"`
	ExampleBad     string   `yaml:"example_bad"     json:"example_bad,omitempty"`
	ExampleGood    string   `yaml:"example_good"    json:"example_good,omitempty"`
	FirstSeen      string   `yaml:"first_seen"      json:"first_seen,omitempty"`
	Detection      string   `yaml:"detection"       json:"detection"`
	AutoFixable    bool     `yaml:"auto_fixable"    json:"auto_fixable"`
}

// Validate checks that the rule has all required fields and that regex patterns compile.
func (r *Rule) Validate() core.Result {
	if r.ID == "" {
		return core.Fail(core.E(ruleRuleValidate1b496e, "id must not be empty", nil))
	}
	if r.Title == "" {
		return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": title must not be empty", nil))
	}
	if r.Severity == "" {
		return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": severity must not be empty", nil))
	}
	if !slices.Contains(validSeverities, r.Severity) {
		return core.Fail(core.E(ruleRuleValidate1b496e, core.Sprintf("rule %s: severity %q is not valid (want one of %v)", r.ID, r.Severity, validSeverities), nil))
	}
	if len(r.Languages) == 0 {
		return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": languages must not be empty", nil))
	}
	if r.Pattern == "" {
		return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": pattern must not be empty", nil))
	}
	if r.Detection == "" {
		return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": detection must not be empty", nil))
	}

	// Only validate regex compilation when detection type is regex.
	if r.Detection == "regex" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": pattern does not compile", err))
		}
		if r.ExcludePattern != "" {
			if _, err := regexp.Compile(r.ExcludePattern); err != nil {
				return core.Fail(core.E(ruleRuleValidate1b496e, "rule "+r.ID+": exclude_pattern does not compile", err))
			}
		}
	}

	return core.Ok(nil)
}

// ParseRules unmarshals YAML data into a slice of Rule.
func ParseRules(data []byte) core.Result {
	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return core.Fail(core.E("ParseRules", "parsing rules", err))
	}
	return core.Ok(rules)
}
