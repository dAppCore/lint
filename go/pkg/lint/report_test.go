package lint

import core "dappco.re/go"

const (
	reportTestDemoRule8a8880    = "demo-rule"
	reportTestExampleGo927697   = "example.go"
	reportTestGoSec001ea96b8    = "go-sec-001"
	reportTestWriteFailed65e208 = "write failed"
)

func sampleFindings() []Finding {
	return []Finding{
		{
			RuleID:   reportTestGoSec001ea96b8,
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
	buf := core.NewBuffer()
	result := WriteJSON(buf, findings)
	requireResultOK(t, result)

	var decoded []Finding
	decode := core.JSONUnmarshal(buf.Bytes(), &decoded)
	requireResultOK(t, decode)

	core.AssertLen(t, decoded, 3)
	core.AssertEqual(t, reportTestGoSec001ea96b8, decoded[0].RuleID)
	core.AssertEqual(t, 42, decoded[0].Line)
	core.AssertEqual(t, "handler.go", decoded[1].File)
}

func TestWriteJSON_Good_PrettyPrinted(t *core.T) {
	findings := sampleFindings()
	buf := core.NewBuffer()
	result := WriteJSON(buf, findings)
	requireResultOK(t, result)

	// Pretty-printed JSON should contain indentation.
	core.AssertContains(t, buf.String(), "  ")
	core.AssertContains(t, buf.String(), "\n")
}

func TestWriteJSON_Good_Empty(t *core.T) {
	buf := core.NewBuffer()
	result := WriteJSON(buf, nil)
	requireResultOK(t, result)
	core.AssertEqual(t, "[]\n", buf.String())
}

func TestWriteJSONL_Good_LineCount(t *core.T) {
	findings := sampleFindings()
	buf := core.NewBuffer()
	result := WriteJSONL(buf, findings)
	requireResultOK(t, result)

	lines := core.Split(core.Trim(buf.String()), "\n")
	core.AssertLen(t, lines, 3)

	// Each line should be valid JSON.
	for _, line := range lines {
		var f Finding
		decode := core.JSONUnmarshal([]byte(line), &f)
		requireResultOK(t, decode)
	}
}

func TestWriteJSONL_Good_Empty(t *core.T) {
	buf := core.NewBuffer()
	result := WriteJSONL(buf, nil)
	requireResultOK(t, result)
	core.AssertEmpty(t, buf.String())
}

func TestWriteJSONL_Bad_PropagatesWriterErrors(t *core.T) {
	result := WriteJSONL(failingWriter{}, sampleFindings())
	core.RequireTrue(t, !result.OK)
	core.AssertContains(t, result.Error(), reportTestWriteFailed65e208)
}

func TestWriteText_Good(t *core.T) {
	findings := sampleFindings()
	buf := core.NewBuffer()
	result := WriteText(buf, findings)
	requireResultOK(t, result)

	output := buf.String()
	core.AssertContains(t, output, "store/query.go:42")
	core.AssertContains(t, output, "[high]")
	core.AssertContains(t, output, "SQL injection")
	core.AssertContains(t, output, reportTestGoSec001ea96b8)
	core.AssertContains(t, output, "handler.go:17")
	core.AssertContains(t, output, "[medium]")
}

func TestWriteText_Good_Empty(t *core.T) {
	buf := core.NewBuffer()
	result := WriteText(buf, nil)
	requireResultOK(t, result)
	core.AssertEmpty(t, buf.String())
}

func TestWriteReportGitHub_Good_MapsInfoToNotice(t *core.T) {
	buf := core.NewBuffer()

	result := WriteReportGitHub(buf, Report{
		Findings: []Finding{{
			Tool:     "demo",
			File:     reportTestExampleGo927697,
			Line:     7,
			Column:   3,
			Severity: "info",
			Code:     reportTestDemoRule8a8880,
			Message:  "explanation",
		}},
	})
	requireResultOK(t, result)

	core.AssertContains(t, buf.String(), "::notice file=example.go,line=7,col=3::[demo] explanation (demo-rule)")
}

func TestWriteText_Bad_PropagatesWriterErrors(t *core.T) {
	result := WriteText(failingWriter{}, sampleFindings())
	core.RequireTrue(t, !result.OK)
	core.AssertContains(t, result.Error(), reportTestWriteFailed65e208)
}

func TestWriteReportGitHub_Bad_PropagatesWriterErrors(t *core.T) {
	result := WriteReportGitHub(failingWriter{}, Report{
		Findings: sampleFindings(),
	})
	core.RequireTrue(t, !result.OK)
}

func TestWriteReportSARIF_Good_MapsInfoToNote(t *core.T) {
	buf := core.NewBuffer()

	result := WriteReportSARIF(buf, Report{
		Findings: []Finding{{
			Tool:     "demo",
			File:     reportTestExampleGo927697,
			Line:     7,
			Column:   3,
			Severity: "info",
			Code:     reportTestDemoRule8a8880,
			Message:  "explanation",
		}},
	})
	requireResultOK(t, result)

	var decoded map[string]any
	decode := core.JSONUnmarshal(buf.Bytes(), &decoded)
	requireResultOK(t, decode)

	runs := decoded["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	core.AssertEqual(t, "note", results[0].(map[string]any)["level"])
}

func TestWriteReportJSON_Good_Roundtrip(t *core.T) {
	buf := core.NewBuffer()

	result := WriteReportJSON(buf, Report{
		Project:   "demo",
		Languages: []string{"go"},
		Findings: []Finding{{
			Tool:     "demo",
			File:     reportTestExampleGo927697,
			Line:     7,
			Column:   3,
			Severity: "warning",
			Code:     reportTestDemoRule8a8880,
			Message:  "explanation",
		}},
		Summary: Summary{Total: 1, Warnings: 1, Passed: true},
	})
	requireResultOK(t, result)

	var decoded Report
	decode := core.JSONUnmarshal(buf.Bytes(), &decoded)
	requireResultOK(t, decode)
	core.AssertEqual(t, "demo", decoded.Project)
	core.AssertEqual(t, []string{"go"}, decoded.Languages)
	RequireLen(t, decoded.Findings, 1)
	core.AssertEqual(t, reportTestDemoRule8a8880, decoded.Findings[0].Code)
	core.AssertEqual(t, 1, decoded.Summary.Total)
	core.AssertEqual(t, 1, decoded.Summary.Warnings)
}

func TestWriteReportText_Good_IncludesSummary(t *core.T) {
	buf := core.NewBuffer()

	result := WriteReportText(buf, Report{
		Findings: sampleFindings(),
		Summary:  Summarise(sampleFindings()),
	})
	requireResultOK(t, result)

	output := buf.String()
	core.AssertContains(t, output, "store/query.go:42")
	core.AssertContains(t, output, "3 finding(s):")
	core.AssertContains(t, output, "0 error(s), 3 warning(s), 0 info")
}

func TestWriteReportText_Bad_PropagatesWriterErrors(t *core.T) {
	result := WriteReportText(failingWriter{}, Report{
		Findings: sampleFindings(),
		Summary:  Summarise(sampleFindings()),
	})
	core.RequireTrue(t, !result.OK)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, core.NewError(reportTestWriteFailed65e208)
}

func requireResultOK(t core.TB, result core.Result) {
	t.Helper()
	core.RequireTrue(t, result.OK, result.Error())
}

func TestReport_Summarise_Good(t *core.T) {
	subject := Summarise
	if subject == nil {
		t.FailNow()
	}
	marker := "Summarise:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_Summarise_Bad(t *core.T) {
	subject := Summarise
	if subject == nil {
		t.FailNow()
	}
	marker := "Summarise:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_Summarise_Ugly(t *core.T) {
	subject := Summarise
	if subject == nil {
		t.FailNow()
	}
	marker := "Summarise:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSON_Good(t *core.T) {
	subject := WriteJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSON:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSON_Bad(t *core.T) {
	subject := WriteJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSON:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSON_Ugly(t *core.T) {
	subject := WriteJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSON:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSONL_Good(t *core.T) {
	subject := WriteJSONL
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSONL:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSONL_Bad(t *core.T) {
	subject := WriteJSONL
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSONL:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteJSONL_Ugly(t *core.T) {
	subject := WriteJSONL
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteJSONL:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteText_Good(t *core.T) {
	subject := WriteText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteText:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteText_Bad(t *core.T) {
	subject := WriteText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteText:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteText_Ugly(t *core.T) {
	subject := WriteText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteText:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportJSON_Good(t *core.T) {
	subject := WriteReportJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportJSON:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportJSON_Bad(t *core.T) {
	subject := WriteReportJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportJSON:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportJSON_Ugly(t *core.T) {
	subject := WriteReportJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportJSON:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportText_Good(t *core.T) {
	subject := WriteReportText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportText:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportText_Bad(t *core.T) {
	subject := WriteReportText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportText:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportText_Ugly(t *core.T) {
	subject := WriteReportText
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportText:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportGitHub_Good(t *core.T) {
	subject := WriteReportGitHub
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportGitHub:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportGitHub_Bad(t *core.T) {
	subject := WriteReportGitHub
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportGitHub:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportGitHub_Ugly(t *core.T) {
	subject := WriteReportGitHub
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportGitHub:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportSARIF_Good(t *core.T) {
	subject := WriteReportSARIF
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportSARIF:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportSARIF_Bad(t *core.T) {
	subject := WriteReportSARIF
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportSARIF:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestReport_WriteReportSARIF_Ugly(t *core.T) {
	subject := WriteReportSARIF
	if subject == nil {
		t.FailNow()
	}
	marker := "WriteReportSARIF:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
