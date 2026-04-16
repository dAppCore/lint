package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSummarise_Good(t *testing.T) {
	findings := sampleFindings()
	summary := Summarise(findings)

	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.BySeverity["high"])
	assert.Equal(t, 1, summary.BySeverity["medium"])
}

func TestSummarise_Good_Empty(t *testing.T) {
	summary := Summarise(nil)
	assert.Equal(t, 0, summary.Total)
	assert.Empty(t, summary.BySeverity)
}

func TestWriteJSON_Good_Roundtrip(t *testing.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSON(&buf, findings)
	require.NoError(t, err)

	var decoded []Finding
	err = json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded, 3)
	assert.Equal(t, "go-sec-001", decoded[0].RuleID)
	assert.Equal(t, 42, decoded[0].Line)
	assert.Equal(t, "handler.go", decoded[1].File)
}

func TestWriteJSON_Good_PrettyPrinted(t *testing.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSON(&buf, findings)
	require.NoError(t, err)

	// Pretty-printed JSON should contain indentation.
	assert.Contains(t, buf.String(), "  ")
	assert.Contains(t, buf.String(), "\n")
}

func TestWriteJSON_Good_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, nil)
	require.NoError(t, err)
	assert.Equal(t, "[]\n", buf.String())
}

func TestWriteJSONL_Good_LineCount(t *testing.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteJSONL(&buf, findings)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 3)

	// Each line should be valid JSON.
	for _, line := range lines {
		var f Finding
		err := json.Unmarshal([]byte(line), &f)
		require.NoError(t, err)
	}
}

func TestWriteJSONL_Good_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSONL(&buf, nil)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteText_Good(t *testing.T) {
	findings := sampleFindings()
	var buf bytes.Buffer
	err := WriteText(&buf, findings)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "store/query.go:42")
	assert.Contains(t, output, "[high]")
	assert.Contains(t, output, "SQL injection")
	assert.Contains(t, output, "go-sec-001")
	assert.Contains(t, output, "handler.go:17")
	assert.Contains(t, output, "[medium]")
}

func TestWriteText_Good_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteText(&buf, nil)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteReportGitHub_Good_MapsInfoToNotice(t *testing.T) {
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
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "::notice file=example.go,line=7,col=3::[demo] explanation (demo-rule)")
}

func TestWriteText_Bad_PropagatesWriterErrors(t *testing.T) {
	err := WriteText(failingWriter{}, sampleFindings())
	require.Error(t, err)
}

func TestWriteReportGitHub_Bad_PropagatesWriterErrors(t *testing.T) {
	err := WriteReportGitHub(failingWriter{}, Report{
		Findings: sampleFindings(),
	})
	require.Error(t, err)
}

func TestWriteReportSARIF_Good_MapsInfoToNote(t *testing.T) {
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
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	runs := decoded["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	assert.Equal(t, "note", results[0].(map[string]any)["level"])
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
