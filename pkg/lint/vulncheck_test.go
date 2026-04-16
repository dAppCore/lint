package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVulnCheckJSON_WithFindings(t *testing.T) {
	stdout := `{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
{"osv":{"id":"GO-2024-1234","aliases":["CVE-2024-1234"],"summary":"Buffer overflow in foo","affected":[{"ranges":[{"events":[{"fixed":"1.2.3"}]}]}]}}
{"finding":{"osv":"GO-2024-1234","trace":[{"module":"example.com/app","package":"example.com/app/cmd","function":"main","version":"v0.1.0"},{"module":"example.com/foo","package":"example.com/foo","function":"Bar","version":"v1.0.0"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	require.NoError(t, err)
	assert.Equal(t, "example.com/app", result.Module)
	require.Len(t, result.Findings, 1)

	f := result.Findings[0]
	assert.Equal(t, "GO-2024-1234", f.ID)
	assert.Equal(t, "Buffer overflow in foo", f.Description)
	assert.Contains(t, f.Aliases, "CVE-2024-1234")
	assert.Equal(t, "example.com/foo", f.Package)
	assert.Equal(t, "Bar", f.CalledFunction)
	assert.Equal(t, "v0.1.0", f.FixedVersion)
}

func TestParseVulnCheckJSON_NoFindings(t *testing.T) {
	stdout := `{"config":{"go_version":"go1.26","module_path":"example.com/clean"}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	require.NoError(t, err)
	assert.Equal(t, "example.com/clean", result.Module)
	assert.Empty(t, result.Findings)
}

func TestParseVulnCheckJSON_MalformedLines(t *testing.T) {
	stdout := `not json at all
{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
also not json
{"osv":{"id":"GO-2024-5678","summary":"Test vuln","aliases":[],"affected":[]}}
{"finding":{"osv":"GO-2024-5678","trace":[{"package":"example.com/dep","function":"Fn"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestParseVulnCheckJSON_Empty(t *testing.T) {
	result, err := ParseVulnCheckJSON("", "")
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.Empty(t, result.Module)
}

func TestParseVulnCheckJSON_MultipleFindings(t *testing.T) {
	stdout := `{"osv":{"id":"GO-2024-001","summary":"Vuln 1","aliases":[],"affected":[]}}
{"osv":{"id":"GO-2024-002","summary":"Vuln 2","aliases":[],"affected":[]}}
{"finding":{"osv":"GO-2024-001","trace":[{"package":"pkg1"}]}}
{"finding":{"osv":"GO-2024-002","trace":[{"package":"pkg2"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	require.NoError(t, err)
	assert.Len(t, result.Findings, 2)
	assert.Equal(t, "GO-2024-001", result.Findings[0].ID)
	assert.Equal(t, "GO-2024-002", result.Findings[1].ID)
}

func TestParseVulnCheckJSON_FixedVersionFromOSV(t *testing.T) {
	stdout := `{"osv":{"id":"GO-2024-999","summary":"Fix version test","aliases":[],"affected":[{"ranges":[{"events":[{"fixed":"2.0.0"}]}]}]}}
{"finding":{"osv":"GO-2024-999","trace":[{"package":"example.com/lib"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "2.0.0", result.Findings[0].FixedVersion)
}
