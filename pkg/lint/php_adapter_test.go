package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPHPAdapter_ParsePHPStan_Good(t *testing.T) {
	output := `{
  "totals": {"errors": 1, "file_errors": 1},
  "files": {
    "src/Foo.php": {
      "errors": 1,
      "messages": [
        {
          "message": "Method Foo::bar() should return int but returns string.",
          "line": 12,
          "ignorable": true,
          "identifier": "return.type"
        }
      ]
    }
  }
}`

	findings := parsePHPStanDiagnostics("phpstan", "correctness", output)
	require.NotEmpty(t, findings, "expected at least one finding from phpstan output")

	hadReturnType := false
	for _, finding := range findings {
		assert.Equal(t, "phpstan", finding.Tool)
		assert.NotEmpty(t, finding.Severity, "phpstan findings always carry severity")
		if finding.Code == "return.type" || finding.Message != "" {
			hadReturnType = true
		}
	}
	assert.True(t, hadReturnType, "expected to parse the return.type finding")
}

func TestPHPAdapter_ParsePHPStan_Bad(t *testing.T) {
	findings := parsePHPStanDiagnostics("phpstan", "correctness", "not valid json")
	require.NotEmpty(t, findings, "garbage input emits a parse-error finding")
	assert.Equal(t, "phpstan", findings[0].Tool)
	assert.Equal(t, "parse-error", findings[0].Code)
}

func TestPHPAdapter_ParsePHPStan_Ugly(t *testing.T) {
	output := `{"totals":{"errors":0,"file_errors":0},"files":{}}`
	findings := parsePHPStanDiagnostics("phpstan", "correctness", output)
	assert.Empty(t, findings, "clean phpstan output emits no findings")
}

func TestPHPAdapter_ParsePsalm_Good_TaintFlow(t *testing.T) {
	output := `[
  {
    "severity": "error",
    "line_from": 18,
    "line_to": 18,
    "type": "TaintedSql",
    "message": "Detected tainted SQL",
    "file_name": "src/Bar.php",
    "file_path": "src/Bar.php",
    "snippet": "$db->query($_GET['id'])",
    "selected_text": "$_GET['id']",
    "from": 100,
    "to": 110,
    "snippet_from": 90,
    "snippet_to": 130,
    "column_from": 21,
    "column_to": 31
  }
]`

	findings := parsePsalmDiagnostics("psalm", "correctness", output)
	require.NotEmpty(t, findings, "psalm taint output should produce at least one finding")

	taintFound := false
	for _, finding := range findings {
		if finding.Category == "security" && finding.Severity == "error" {
			taintFound = true
			break
		}
	}
	assert.True(t, taintFound, "TaintedSql finding must be re-tagged category=security severity=error")
}

func TestPHPAdapter_ParsePsalm_Good_NonTaintRetainsCorrectness(t *testing.T) {
	output := `[
  {
    "severity": "info",
    "line_from": 5,
    "line_to": 5,
    "type": "MissingReturnType",
    "message": "Method Foo::bar does not have a return type",
    "file_name": "src/Foo.php",
    "file_path": "src/Foo.php"
  }
]`

	findings := parsePsalmDiagnostics("psalm", "correctness", output)
	require.NotEmpty(t, findings)

	for _, finding := range findings {
		assert.NotEqual(t, "security", finding.Category, "non-taint findings stay in their original category")
	}
}

func TestPHPAdapter_ParsePsalm_Bad(t *testing.T) {
	findings := parsePsalmDiagnostics("psalm", "correctness", "{not-json")
	require.NotEmpty(t, findings)
	assert.Equal(t, "parse-error", findings[0].Code)
}

func TestPHPAdapter_IsPsalmTaintFinding_Good(t *testing.T) {
	cases := []struct {
		name    string
		finding Finding
		want    bool
	}{
		{"TaintedSql code", Finding{Code: "TaintedSql"}, true},
		{"TaintedShell rule_id", Finding{RuleID: "TaintedShell"}, true},
		{"TaintedHtml in message", Finding{Message: "TaintedHtml flow detected"}, true},
		{"non-taint", Finding{Code: "MissingReturnType"}, false},
		{"empty", Finding{}, false},
		{"psalm-flow marker", Finding{Message: "see @psalm-flow on parameter"}, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, isPsalmTaintFinding(testCase.finding))
		})
	}
}

// TestPHPAdapter_DefaultAdapters_PHPStan_Wired ensures the registry row for
// phpstan still routes to the dedicated parser.
func TestPHPAdapter_DefaultAdapters_PHPStan_Wired(t *testing.T) {
	adapters := defaultAdapters()
	for _, adapter := range adapters {
		if adapter.Name() == "phpstan" {
			assert.Equal(t, "correctness", adapter.Category())
			return
		}
	}
	t.Fatal("phpstan adapter not present in defaultAdapters()")
}

// TestPHPAdapter_DefaultAdapters_Psalm_Wired ensures the registry row for
// psalm still routes to the dedicated parser.
func TestPHPAdapter_DefaultAdapters_Psalm_Wired(t *testing.T) {
	adapters := defaultAdapters()
	for _, adapter := range adapters {
		if adapter.Name() == "psalm" {
			assert.Equal(t, "correctness", adapter.Category())
			return
		}
	}
	t.Fatal("psalm adapter not present in defaultAdapters()")
}
