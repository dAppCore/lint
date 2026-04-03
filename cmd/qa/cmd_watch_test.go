package qa

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintResults_SortsRunsAndUsesDeterministicDetails(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gh"), `#!/bin/sh
case "$*" in
  *"run view 2 --repo forge/alpha --json jobs"*)
    cat <<'JSON'
{"jobs":[
  {
    "databaseId": 20,
    "name": "Zulu Job",
    "status": "completed",
    "conclusion": "failure",
    "steps": [
      {"name": "Zulu Step", "status": "completed", "conclusion": "failure", "number": 2}
    ]
  },
  {
    "databaseId": 10,
    "name": "Alpha Job",
    "status": "completed",
    "conclusion": "failure",
    "steps": [
      {"name": "Zulu Step", "status": "completed", "conclusion": "failure", "number": 2},
      {"name": "Alpha Step", "status": "completed", "conclusion": "failure", "number": 1}
    ]
  }
]}
JSON
    ;;
  *"run view 2 --repo forge/alpha --log-failed"*)
    cat <<'EOF'
Alpha error detail
EOF
    ;;
  *"run view 4 --repo forge/alpha --json jobs"*)
    cat <<'JSON'
{"jobs":[
  {
    "databaseId": 40,
    "name": "Omega Job",
    "status": "completed",
    "conclusion": "failure",
    "steps": [
      {"name": "Omega Step", "status": "completed", "conclusion": "failure", "number": 1}
    ]
  }
]}
JSON
    ;;
  *"run view 4 --repo forge/alpha --log-failed"*)
    cat <<'EOF'
Omega error detail
EOF
    ;;
  *)
    printf '%s\n' "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`)

	prependPath(t, dir)

	runs := []WorkflowRun{
		{ID: 3, Name: "Zulu Build", Conclusion: "success", URL: "https://example.com/zulu"},
		{ID: 1, Name: "Alpha Build", Conclusion: "success", URL: "https://example.com/alpha"},
		{ID: 4, Name: "Omega Failure", Conclusion: "failure", URL: "https://example.com/omega"},
		{ID: 2, Name: "Beta Failure", Conclusion: "failure", URL: "https://example.com/beta"},
	}

	output := captureStdout(t, func() {
		err := printResults(context.Background(), "forge/alpha", runs)
		require.Error(t, err)
	})

	assert.NotContains(t, output, "\033[2K\r")
	alphaBuild := strings.Index(output, "Alpha Build")
	require.NotEqual(t, -1, alphaBuild)
	zuluBuild := strings.Index(output, "Zulu Build")
	require.NotEqual(t, -1, zuluBuild)
	assert.Less(t, alphaBuild, zuluBuild)

	betaFailure := strings.Index(output, "Beta Failure")
	require.NotEqual(t, -1, betaFailure)
	omegaFailure := strings.Index(output, "Omega Failure")
	require.NotEqual(t, -1, omegaFailure)
	assert.Less(t, betaFailure, omegaFailure)
	assert.Contains(t, output, "Job: Alpha Job (step: 1: Alpha Step)")
	assert.Contains(t, output, "Error: Alpha error detail")
	assert.NotContains(t, output, "Job: Zulu Job")
}
