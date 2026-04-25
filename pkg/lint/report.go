package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Summary holds aggregate counts for a set of findings.
type Summary struct {
	Total      int            `json:"total"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
	Info       int            `json:"info"`
	Passed     bool           `json:"passed"`
	BySeverity map[string]int `json:"by_severity,omitempty"`
}

// Summarise counts findings by severity.
//
//	summary := lint.Summarise(findings)
func Summarise(findings []Finding) Summary {
	summary := Summary{
		Total:      len(findings),
		BySeverity: make(map[string]int),
	}
	for _, finding := range findings {
		severity := strings.TrimSpace(finding.Severity)
		if severity == "" {
			severity = "warning"
		}
		summary.BySeverity[severity]++
		switch severity {
		case "error":
			summary.Errors++
		case "info":
			summary.Info++
		default:
			summary.Warnings++
		}
	}
	summary.Passed = summary.Errors == 0
	return summary
}

// WriteJSON writes findings as a pretty-printed JSON array.
//
//	_ = lint.WriteJSON(os.Stdout, findings)
func WriteJSON(w io.Writer, findings []Finding) error {
	if findings == nil {
		findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(findings)
}

// WriteJSONL writes findings as newline-delimited JSON (one object per line).
//
//	_ = lint.WriteJSONL(os.Stdout, findings)
func WriteJSONL(w io.Writer, findings []Finding) error {
	for _, f := range findings {
		data, err := json.Marshal(f)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
			return err
		}
	}
	return nil
}

// WriteText writes findings in a human-readable format.
//
//	lint.WriteText(os.Stdout, findings)
func WriteText(w io.Writer, findings []Finding) error {
	for _, finding := range findings {
		message := finding.Message
		if message == "" {
			message = finding.Title
		}
		code := finding.Code
		if code == "" {
			code = finding.RuleID
		}
		if _, err := fmt.Fprintf(w, "%s:%d [%s] %s (%s)\n", finding.File, finding.Line, finding.Severity, message, code); err != nil {
			return err
		}
	}
	return nil
}

// WriteReportJSON writes the RFC report document as pretty-printed JSON.
//
//	_ = lint.WriteReportJSON(os.Stdout, report)
func WriteReportJSON(w io.Writer, report Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// WriteReportText writes report findings followed by a short summary.
//
//	lint.WriteReportText(os.Stdout, report)
func WriteReportText(w io.Writer, report Report) error {
	if err := WriteText(w, report.Findings); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\n%d finding(s): %d error(s), %d warning(s), %d info\n", report.Summary.Total, report.Summary.Errors, report.Summary.Warnings, report.Summary.Info)
	return err
}

// WriteReportGitHub writes GitHub Actions annotation lines.
//
//	lint.WriteReportGitHub(os.Stdout, report)
func WriteReportGitHub(w io.Writer, report Report) error {
	for _, finding := range report.Findings {
		level := githubAnnotationLevel(finding.Severity)

		location := ""
		if finding.File != "" {
			location = fmt.Sprintf(" file=%s", finding.File)
			if finding.Line > 0 {
				location += fmt.Sprintf(",line=%d", finding.Line)
			}
			if finding.Column > 0 {
				location += fmt.Sprintf(",col=%d", finding.Column)
			}
		}

		message := finding.Message
		if message == "" {
			message = finding.Title
		}
		code := finding.Code
		if code == "" {
			code = finding.RuleID
		}
		if _, err := fmt.Fprintf(w, "::%s%s::[%s] %s (%s)\n", level, location, finding.Tool, message, code); err != nil {
			return err
		}
	}
	return nil
}

// WriteReportSARIF writes a minimal SARIF document for code scanning tools.
//
//	_ = lint.WriteReportSARIF(os.Stdout, report)
func WriteReportSARIF(w io.Writer, report Report) error {
	type sarifMessage struct {
		Text string `json:"text"`
	}
	type sarifRegion struct {
		StartLine   int `json:"startLine,omitempty"`
		StartColumn int `json:"startColumn,omitempty"`
	}
	type sarifArtifactLocation struct {
		URI string `json:"uri,omitempty"`
	}
	type sarifPhysicalLocation struct {
		ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
		Region           sarifRegion           `json:"region,omitempty"`
	}
	type sarifLocation struct {
		PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
	}
	type sarifResult struct {
		RuleID    string          `json:"ruleId,omitempty"`
		Level     string          `json:"level,omitempty"`
		Message   sarifMessage    `json:"message"`
		Locations []sarifLocation `json:"locations,omitempty"`
	}
	type sarifRun struct {
		Tool struct {
			Driver struct {
				Name string `json:"name"`
			} `json:"driver"`
		} `json:"tool"`
		Results []sarifResult `json:"results"`
	}
	type sarifLog struct {
		Version string     `json:"version"`
		Schema  string     `json:"$schema"`
		Runs    []sarifRun `json:"runs"`
	}

	sarifRunValue := sarifRun{}
	sarifRunValue.Tool.Driver.Name = "core-lint"

	for _, finding := range report.Findings {
		message := finding.Message
		if message == "" {
			message = finding.Title
		}
		ruleID := finding.Code
		if ruleID == "" {
			ruleID = finding.RuleID
		}

		result := sarifResult{
			RuleID:  ruleID,
			Level:   sarifLevel(finding.Severity),
			Message: sarifMessage{Text: message},
		}
		if finding.File != "" {
			result.Locations = []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: finding.File},
					Region: sarifRegion{
						StartLine:   finding.Line,
						StartColumn: finding.Column,
					},
				},
			}}
		}
		sarifRunValue.Results = append(sarifRunValue.Results, result)
	}

	return json.NewEncoder(w).Encode(sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    []sarifRun{sarifRunValue},
	})
}

func githubAnnotationLevel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return "error"
	case "info":
		return "notice"
	case "warning", "":
		return "warning"
	default:
		return "warning"
	}
}

func sarifLevel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return "error"
	case "warning":
		return "warning"
	case "info":
		return "note"
	default:
		return "warning"
	}
}
