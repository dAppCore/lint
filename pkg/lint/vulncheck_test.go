package lint

import (
	core "dappco.re/go"
)

func TestParseVulnCheckJSON_WithFindings(t *core.T) {
	stdout := `{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
{"osv":{"id":"GO-2024-1234","aliases":["CVE-2024-1234"],"summary":"Buffer overflow in foo","affected":[{"ranges":[{"events":[{"fixed":"1.2.3"}]}]}]}}
{"finding":{"osv":"GO-2024-1234","trace":[{"module":"example.com/app","package":"example.com/app/cmd","function":"main","version":"v0.1.0"},{"module":"example.com/foo","package":"example.com/foo","function":"Bar","version":"v1.0.0"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "example.com/app", result.Module)
	RequireLen(t, result.Findings, 1)

	f := result.Findings[0]
	core.AssertEqual(t, "GO-2024-1234", f.ID)
	core.AssertEqual(t, "Buffer overflow in foo", f.Description)
	core.AssertContains(t, f.Aliases, "CVE-2024-1234")
	core.AssertEqual(t, "example.com/foo", f.Package)
	core.AssertEqual(t, "Bar", f.CalledFunction)
	core.AssertEqual(t, "v0.1.0", f.FixedVersion)
}

func TestParseVulnCheckJSON_NoFindings(t *core.T) {
	stdout := `{"config":{"go_version":"go1.26","module_path":"example.com/clean"}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "example.com/clean", result.Module)
	core.AssertEmpty(t, result.Findings)
}

func TestParseVulnCheckJSON_MalformedLines(t *core.T) {
	stdout := `not json at all
{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
also not json
{"osv":{"id":"GO-2024-5678","summary":"Test vuln","aliases":[],"affected":[]}}
{"finding":{"osv":"GO-2024-5678","trace":[{"package":"example.com/dep","function":"Fn"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	RequireError(t, err)
	core.AssertNil(t, result)
}

func TestParseVulnCheckJSON_Empty(t *core.T) {
	result, err := ParseVulnCheckJSON("", "")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, result.Findings)
	core.AssertEmpty(t, result.Module)
}

func TestParseVulnCheckJSON_MultipleFindings(t *core.T) {
	stdout := `{"osv":{"id":"GO-2024-001","summary":"Vuln 1","aliases":[],"affected":[]}}
{"osv":{"id":"GO-2024-002","summary":"Vuln 2","aliases":[],"affected":[]}}
{"finding":{"osv":"GO-2024-001","trace":[{"package":"pkg1"}]}}
{"finding":{"osv":"GO-2024-002","trace":[{"package":"pkg2"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	core.RequireNoError(t, err)
	core.AssertLen(t, result.Findings, 2)
	core.AssertEqual(t, "GO-2024-001", result.Findings[0].ID)
	core.AssertEqual(t, "GO-2024-002", result.Findings[1].ID)
}

func TestParseVulnCheckJSON_FixedVersionFromOSV(t *core.T) {
	stdout := `{"osv":{"id":"GO-2024-999","summary":"Fix version test","aliases":[],"affected":[{"ranges":[{"events":[{"fixed":"2.0.0"}]}]}]}}
{"finding":{"osv":"GO-2024-999","trace":[{"package":"example.com/lib"}]}}
`
	result, err := ParseVulnCheckJSON(stdout, "")
	core.RequireNoError(t, err)
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "2.0.0", result.Findings[0].FixedVersion)
}
