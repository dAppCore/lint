package lint

import (
	"fmt"
	"regexp"
	"slices"

	"gopkg.in/yaml.v3"
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
func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule validation: id must not be empty")
	}
	if r.Title == "" {
		return fmt.Errorf("rule %s: title must not be empty", r.ID)
	}
	if r.Severity == "" {
		return fmt.Errorf("rule %s: severity must not be empty", r.ID)
	}
	if !slices.Contains(validSeverities, r.Severity) {
		return fmt.Errorf("rule %s: severity %q is not valid (want one of %v)", r.ID, r.Severity, validSeverities)
	}
	if len(r.Languages) == 0 {
		return fmt.Errorf("rule %s: languages must not be empty", r.ID)
	}
	if r.Pattern == "" {
		return fmt.Errorf("rule %s: pattern must not be empty", r.ID)
	}
	if r.Detection == "" {
		return fmt.Errorf("rule %s: detection must not be empty", r.ID)
	}

	// Only validate regex compilation when detection type is regex.
	if r.Detection == "regex" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return fmt.Errorf("rule %s: pattern does not compile: %w", r.ID, err)
		}
		if r.ExcludePattern != "" {
			if _, err := regexp.Compile(r.ExcludePattern); err != nil {
				return fmt.Errorf("rule %s: exclude_pattern does not compile: %w", r.ID, err)
			}
		}
	}

	return nil
}

// ParseRules unmarshals YAML data into a slice of Rule.
func ParseRules(data []byte) ([]Rule, error) {
	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parsing rules: %w", err)
	}
	return rules, nil
}
