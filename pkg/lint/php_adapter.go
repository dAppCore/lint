package lint

import (
	"strconv"
	"strings"

	core "dappco.re/go/core"
)

// parsePHPStanDiagnostics parses phpstan --error-format=json into Finding values.
//
// Severity is derived from the embedded `identifier` prefix when available
// (any larastan.* finding inherits PHPStan severity rules). Default category
// is correctness; the parser preserves any RuleID so downstream consumers
// can route on `Code`.
//
//	parsed := parsePHPStanDiagnostics("phpstan", "correctness", stdout)
func parsePHPStanDiagnostics(tool string, category string, output string) []Finding {
	findings := parseJSONDiagnostics(tool, category, output)
	for index := range findings {
		if findings[index].Tool == "" {
			findings[index].Tool = tool
		}
		if findings[index].Category == "" {
			findings[index].Category = category
		}
		// PHPStan's level system maps level >= 5 to "error"; we have no level
		// at parse-time, so honour the existing severity if present and fall
		// back to "error" otherwise — matching the JSON output's own severity.
		if findings[index].Severity == "" {
			findings[index].Severity = "error"
		}
	}
	return findings
}

// parsePsalmDiagnostics parses psalm --output-format=json into Finding values
// with taint-class re-categorisation.
//
// Findings whose `type`, `code`, or `message` references a Psalm taint-class
// rule (TaintedSql, TaintedShell, TaintedHtml, TaintedInput, TaintedFile,
// TaintedHeader, TaintedLdap, TaintedXpath, TaintedEval, TaintedSSRF,
// TaintedTextWithMarkup, TaintedSystemSecret, TaintedUserSecret,
// TaintedCallable, TaintedCookie, TaintedCustom, TaintedUnserialize) are
// re-tagged as `category=security` with `severity=error` so the
// taint-analysis signal is preserved end-to-end through the aggregated
// Report.
//
//	parsed := parsePsalmDiagnostics("psalm", "correctness", stdout)
func parsePsalmDiagnostics(tool string, category string, output string) []Finding {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}

	var raw []map[string]any
	if result := core.JSONUnmarshal([]byte(trimmed), &raw); !result.OK {
		// Fall back to generic JSON walker for non-array payloads (e.g. errors).
		return tagPsalmFindings(parseJSONDiagnostics(tool, category, trimmed), tool, category)
	}

	findings := make([]Finding, 0, len(raw))
	for _, entry := range raw {
		findings = append(findings, psalmFindingFromMap(tool, category, entry))
	}
	return tagPsalmFindings(findings, tool, category)
}

// psalmFindingFromMap converts one Psalm JSON entry into a Finding. The
// JSON shape for psalm --output-format=json is documented at
// https://psalm.dev/docs/running_psalm/configuration/#output-format.
//
//	finding := psalmFindingFromMap("psalm", "correctness", entry)
func psalmFindingFromMap(tool string, category string, fields map[string]any) Finding {
	finding := Finding{
		Tool:     tool,
		Category: category,
		File:     stringField(fields, "file_name", "file_path"),
		Line:     intField(fields, "line_from", "line"),
		Column:   intField(fields, "column_from", "column"),
		Severity: normaliseSeverity(stringField(fields, "severity")),
		Code:     stringField(fields, "type", "shortcode"),
		Message:  stringField(fields, "message"),
	}
	if finding.Code != "" {
		finding.RuleID = finding.Code
	}
	if finding.Severity == "" {
		finding.Severity = defaultSeverityForCategory(category)
	}
	return finding
}

// tagPsalmFindings applies taint-class re-categorisation to a slice of psalm
// findings. Defaults the tool/category fields if a finding came back blank.
//
//	psalm := tagPsalmFindings(findings, "psalm", "correctness")
func tagPsalmFindings(findings []Finding, tool string, category string) []Finding {
	for index := range findings {
		if findings[index].Tool == "" {
			findings[index].Tool = tool
		}
		if isPsalmTaintFinding(findings[index]) {
			findings[index].Category = "security"
			findings[index].Severity = "error"
		} else if findings[index].Category == "" {
			findings[index].Category = category
		}
	}
	return findings
}

// stringField returns the first non-empty string value found at any of the
// provided keys (case-sensitive).
//
//	file := stringField(entry, "file_name", "file_path")
func stringField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

// intField returns the first int-like value found at any of the provided keys.
// Accepts int, int64, float64 (JSON default) and decimal strings.
//
//	line := intField(entry, "line_from", "line")
func intField(fields map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
				return parsed
			}
		}
	}
	return 0
}

// isPsalmTaintFinding reports whether the finding's RuleID, Code, or Message
// matches a Psalm taint-class rule.
//
//	if isPsalmTaintFinding(finding) { finding.Category = "security" }
func isPsalmTaintFinding(finding Finding) bool {
	candidates := []string{finding.Code, finding.RuleID, finding.Message, finding.Title}
	for _, candidate := range candidates {
		lowered := strings.ToLower(candidate)
		for _, marker := range psalmTaintMarkers {
			if strings.Contains(lowered, marker) {
				return true
			}
		}
	}
	return false
}

// psalmTaintMarkers is the lowercased list of Psalm taint-class identifiers
// recognised by isPsalmTaintFinding.
var psalmTaintMarkers = []string{
	"tainted",
	"@psalm-trust-input",
	"@psalm-flow",
	"@psalm-taint-source",
	"@psalm-taint-sink",
	"@psalm-taint-escape",
}
