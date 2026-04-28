package lint

import (
	"bytes"
	core "dappco.re/go"
	"encoding/json"
	"errors"
	"strings"
)

func sampleFindings() []Finding {
	return []Finding{
		{
			RuleID:   "go-sec-001",
			Title:    "SQL injection",
			Severity: "high",
			File:     "store/query.go",
			Line:     42,
			Match:    `db.Query("SELECT * FROM users WHERE name LIKE ?", "%"+input+"%")`,
			Fix:      "Use parameterised LIKE with EscapeLike()",
		},
		{
			RuleID:   "go-cor-003",
			Title:    "Silent error swallowing",
			Severity: "medium",
			File:     "handler.go",
			Line:     17,
			Match:    `_ = service.Process(data)`,
			Fix:      "Handle the error",
		},
		{
			RuleID:   "go-sec-004",
			Title:    "Non-constant-time auth",
			Severity: "high",
			File:     "auth/check.go",
			Line:     88,
			Match:    `if token == expectedToken {`,
			Fix:      "Use subtle.ConstantTimeCompare",
		},
	}
}

func TestSummarise_Good(t *core.T) {
	findings := sampleFindings()
	summary := Summarise(findings)

	core.AssertEqual(t, 3, summary.Total)
	core.AssertEqual(t, 2, summary.BySeverity["high"])
	core.AssertEqual(t, 1, summary.BySeverity["medium"])
}

func TestSummarise_Good_Empty(t *core.T) {
	summary := Summarise(nil)
	core.AssertEqual(t, 0, summary.Total)
	core.AssertEmpty(t, summary.BySeverity)
}

func TestSummarise_Bad_BlankSeverityDefaultsToWarning(t *core.T) {
	summary := Summarise([]Finding{
		{Severity: ""},
		{Severity: "info"},
	})

	core.AssertEqual(t, 2, summary.Total)
	core.AssertEqual(t, 1, summary.Warnings)
	core.AssertEqual(t, 1, summary.Info)
	core.AssertEqual(t, 0, summary.Errors)
	core.AssertTrue(t, summary.Passed)
}

func TestWriteJSON_Good_Roundtrip(t *core.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSON(&buf, findings)
	core.RequireNoError(t, err)

	var decoded []Finding
	err = json.Unmarshal(buf.Bytes(), &decoded)
	core.RequireNoError(t, err)

	core.AssertLen(t, decoded, 3)
	core.AssertEqual(t, "go-sec-001", decoded[0].RuleID)
	core.AssertEqual(t, 42, decoded[0].Line)
	core.AssertEqual(t, "handler.go", decoded[1].File)
}

func TestWriteJSON_Good_PrettyPrinted(t *core.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSON(&buf, findings)
	core.RequireNoError(t, err)

	// Pretty-printed JSON should contain indentation.
	core.AssertContains(t, buf.String(), "  ")
	core.AssertContains(t, buf.String(), "\n")
}

func TestWriteJSON_Good_Empty(t *core.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "[]\n", buf.String())
}

func TestWriteJSONL_Good_LineCount(t *core.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSONL(&buf, findings)
	core.RequireNoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	core.AssertLen(t, lines, 3)

	// Each line should be valid JSON.
	for _, line := range lines {
		var f Finding
		err := json.Unmarshal([]byte(line), &f)
		core.RequireNoError(t, err)
	}
}

func TestWriteJSONL_Good_Empty(t *core.T) {
	var buf bytes.Buffer
	err := WriteJSONL(&buf, nil)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, buf.String())
}

func TestWriteJSONL_Bad_PropagatesWriterErrors(t *core.T) {
	err := WriteJSONL(failingWriter{}, sampleFindings())
	RequireError(t, err)
}

func TestWriteText_Good(t *core.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteText(&buf, findings)
	core.RequireNoError(t, err)

	output := buf.String()
	core.AssertContains(t, output, "store/query.go:42")
	core.AssertContains(t, output, "[high]")
	core.AssertContains(t, output, "SQL injection")
	core.AssertContains(t, output, "go-sec-001")
	core.AssertContains(t, output, "handler.go:17")
	core.AssertContains(t, output, "[medium]")
}

func TestWriteText_Good_Empty(t *core.T) {
	var buf bytes.Buffer
	err := WriteText(&buf, nil)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, buf.String())
}

func TestWriteReportGitHub_Good_MapsInfoToNotice(t *core.T) {
	var buf bytes.Buffer

	err := WriteReportGitHub(&buf, Report{
		Findings: []Finding{{
			Tool:     "demo",
			File:     "example.go",
			Line:     7,
			Column:   3,
			Severity: "info",
			Code:     "demo-rule",
			Message:  "explanation",
		}},
	})
	core.RequireNoError(t, err)

	core.AssertContains(t, buf.String(), "::notice file=example.go,line=7,col=3::[demo] explanation (demo-rule)")
}

func TestWriteText_Bad_PropagatesWriterErrors(t *core.T) {
	err := WriteText(failingWriter{}, sampleFindings())
	RequireError(t, err)
}

func TestWriteReportGitHub_Bad_PropagatesWriterErrors(t *core.T) {
	err := WriteReportGitHub(failingWriter{}, Report{
		Findings: sampleFindings(),
	})
	RequireError(t, err)
}

func TestWriteReportSARIF_Good_MapsInfoToNote(t *core.T) {
	var buf bytes.Buffer

	err := WriteReportSARIF(&buf, Report{
		Findings: []Finding{{
			Tool:     "demo",
			File:     "example.go",
			Line:     7,
			Column:   3,
			Severity: "info",
			Code:     "demo-rule",
			Message:  "explanation",
		}},
	})
	core.RequireNoError(t, err)

	var decoded map[string]any
	core.RequireNoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	runs := decoded["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	core.AssertEqual(t, "note", results[0].(map[string]any)["level"])
}

func TestWriteReportJSON_Good_Roundtrip(t *core.T) {
	var buf bytes.Buffer

	err := WriteReportJSON(&buf, Report{
		Project:   "demo",
		Languages: []string{"go"},
		Findings: []Finding{{
			Tool:     "demo",
			File:     "example.go",
			Line:     7,
			Column:   3,
			Severity: "warning",
			Code:     "demo-rule",
			Message:  "explanation",
		}},
		Summary: Summary{Total: 1, Warnings: 1, Passed: true},
	})
	core.RequireNoError(t, err)

	var decoded Report
	core.RequireNoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	core.AssertEqual(t, "demo", decoded.Project)
	core.AssertEqual(t, []string{"go"}, decoded.Languages)
	RequireLen(t, decoded.Findings, 1)
	core.AssertEqual(t, "demo-rule", decoded.Findings[0].Code)
	core.AssertEqual(t, 1, decoded.Summary.Total)
	core.AssertEqual(t, 1, decoded.Summary.Warnings)
}

func TestWriteReportText_Good_IncludesSummary(t *core.T) {
	var buf bytes.Buffer

	err := WriteReportText(&buf, Report{
		Findings: sampleFindings(),
		Summary:  Summarise(sampleFindings()),
	})
	core.RequireNoError(t, err)

	output := buf.String()
	core.AssertContains(t, output, "store/query.go:42")
	core.AssertContains(t, output, "3 finding(s):")
	core.AssertContains(t, output, "0 error(s), 3 warning(s), 0 info")
}

func TestWriteReportText_Bad_PropagatesWriterErrors(t *core.T) {
	err := WriteReportText(failingWriter{}, Report{
		Findings: sampleFindings(),
		Summary:  Summarise(sampleFindings()),
	})
	RequireError(t, err)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
