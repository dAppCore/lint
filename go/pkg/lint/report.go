package lint

import (
	"io"

	core "dappco.re/go"
)

// Summary holds aggregate counts for a set of findings.
type Summary struct {
	Total      int            `json:"total"`
	Errors     int            `json:"error_count"`
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
		severity := core.Trim(finding.Severity)
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
//	_ = lint.WriteJSON(core.Stdout(), findings)
func WriteJSON(w io.Writer, findings []Finding) core.Result {
	if findings == nil {
		findings = []Finding{}
	}
	data := core.JSONMarshalIndent(findings, "", "  ")
	if !data.OK {
		return data
	}
	return core.WriteString(w, core.Concat(string(data.Value.([]byte)), "\n"))
}

// WriteJSONL writes findings as newline-delimited JSON (one object per line).
//
//	_ = lint.WriteJSONL(core.Stdout(), findings)
func WriteJSONL(w io.Writer, findings []Finding) core.Result {
	for _, f := range findings {
		data := core.JSONMarshal(f)
		if !data.OK {
			return data
		}
		if written := core.WriteString(w, core.Sprintf("%s\n", data.Value.([]byte))); !written.OK {
			return written
		}
	}
	return core.Ok(nil)
}

// WriteText writes findings in a human-readable format.
//
//	lint.WriteText(core.Stdout(), findings)
func WriteText(w io.Writer, findings []Finding) core.Result {
	for _, finding := range findings {
		message := finding.Message
		if message == "" {
			message = finding.Title
		}
		code := finding.Code
		if code == "" {
			code = finding.RuleID
		}
		written := core.WriteString(w, core.Sprintf("%s:%d [%s] %s (%s)\n", finding.File, finding.Line, finding.Severity, message, code))
		if !written.OK {
			return written
		}
	}
	return core.Ok(nil)
}

// WriteReportJSON writes the RFC report document as pretty-printed JSON.
//
//	_ = lint.WriteReportJSON(core.Stdout(), report)
func WriteReportJSON(w io.Writer, report Report) core.Result {
	data := core.JSONMarshalIndent(report, "", "  ")
	if !data.OK {
		return data
	}
	return core.WriteString(w, core.Concat(string(data.Value.([]byte)), "\n"))
}

// WriteReportText writes report findings followed by a short summary.
//
//	lint.WriteReportText(core.Stdout(), report)
func WriteReportText(w io.Writer, report Report) core.Result {
	if written := WriteText(w, report.Findings); !written.OK {
		return written
	}
	return core.WriteString(w, core.Sprintf("\n%d finding(s): %d error(s), %d warning(s), %d info\n", report.Summary.Total, report.Summary.Errors, report.Summary.Warnings, report.Summary.Info))
}

// WriteReportGitHub writes GitHub Actions annotation lines.
//
//	lint.WriteReportGitHub(core.Stdout(), report)
func WriteReportGitHub(w io.Writer, report Report) core.Result {
	for _, finding := range report.Findings {
		level := githubAnnotationLevel(finding.Severity)

		location := ""
		if finding.File != "" {
			location = core.Sprintf(" file=%s", finding.File)
			if finding.Line > 0 {
				location += core.Sprintf(",line=%d", finding.Line)
			}
			if finding.Column > 0 {
				location += core.Sprintf(",col=%d", finding.Column)
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
		written := core.WriteString(w, core.Sprintf("::%s%s::[%s] %s (%s)\n", level, location, finding.Tool, message, code))
		if !written.OK {
			return written
		}
	}
	return core.Ok(nil)
}

// WriteReportSARIF writes a minimal SARIF document for code scanning tools.
//
//	_ = lint.WriteReportSARIF(core.Stdout(), report)
func WriteReportSARIF(w io.Writer, report Report) core.Result {
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

	data := core.JSONMarshal(sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    []sarifRun{sarifRunValue},
	})
	if !data.OK {
		return data
	}
	return core.WriteString(w, core.Concat(string(data.Value.([]byte)), "\n"))
}

func githubAnnotationLevel(severity string) string {
	switch core.Lower(core.Trim(severity)) {
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
	switch core.Lower(core.Trim(severity)) {
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
