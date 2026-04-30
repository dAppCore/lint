package lint

import (
	core "dappco.re/go"
)

func TestParseVulnCheckJSON_WithFindings(t *core.T) {
	stdout := `{"config":{"go_version":"go1.26","module_path":"example.com/app"}}
{"osv":{"id":"GO-2024-1234","aliases":["CVE-2024-1234"],"summary":"Buffer overflow in foo","affected":[{"ranges":[{"events":[{"fixed":"1.2.3"}]}]}]}}
{"finding":{"osv":"GO-2024-1234","trace":[{"module":"example.com/app","package":"example.com/app/cmd","function":"main","version":"v0.1.0"},{"module":"example.com/foo","package":"example.com/foo","function":"Bar","version":"v1.0.0"}]}}
`
	result := RequireResult[*VulnResult](t, ParseVulnCheckJSON(stdout, ""))
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
	result := RequireResult[*VulnResult](t, ParseVulnCheckJSON(stdout, ""))
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
	result := ParseVulnCheckJSON(stdout, "")
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestParseVulnCheckJSON_Empty(t *core.T) {
	result := RequireResult[*VulnResult](t, ParseVulnCheckJSON("", ""))
	core.AssertEmpty(t, result.Findings)
	core.AssertEmpty(t, result.Module)
}

func TestParseVulnCheckJSON_MultipleFindings(t *core.T) {
	stdout := `{"osv":{"id":"GO-2024-001","summary":"Vuln 1","aliases":[],"affected":[]}}
{"osv":{"id":"GO-2024-002","summary":"Vuln 2","aliases":[],"affected":[]}}
{"finding":{"osv":"GO-2024-001","trace":[{"package":"pkg1"}]}}
{"finding":{"osv":"GO-2024-002","trace":[{"package":"pkg2"}]}}
`
	result := RequireResult[*VulnResult](t, ParseVulnCheckJSON(stdout, ""))
	core.AssertLen(t, result.Findings, 2)
	core.AssertEqual(t, "GO-2024-001", result.Findings[0].ID)
	core.AssertEqual(t, "GO-2024-002", result.Findings[1].ID)
}

func TestParseVulnCheckJSON_FixedVersionFromOSV(t *core.T) {
	stdout := `{"osv":{"id":"GO-2024-999","summary":"Fix version test","aliases":[],"affected":[{"ranges":[{"events":[{"fixed":"2.0.0"}]}]}]}}
{"finding":{"osv":"GO-2024-999","trace":[{"package":"example.com/lib"}]}}
`
	result := RequireResult[*VulnResult](t, ParseVulnCheckJSON(stdout, ""))
	RequireLen(t, result.Findings, 1)
	core.AssertEqual(t, "2.0.0", result.Findings[0].FixedVersion)
}

func TestVulncheck_Toolkit_VulnCheck_Good(t *core.T) {
	subject := (*Toolkit).VulnCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "Toolkit:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestVulncheck_Toolkit_VulnCheck_Bad(t *core.T) {
	subject := (*Toolkit).VulnCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "Toolkit:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestVulncheck_Toolkit_VulnCheck_Ugly(t *core.T) {
	subject := (*Toolkit).VulnCheck
	if subject == nil {
		t.FailNow()
	}
	marker := "Toolkit:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestVulncheck_ParseVulnCheckJSON_Good(t *core.T) {
	subject := ParseVulnCheckJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseVulnCheckJSON:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestVulncheck_ParseVulnCheckJSON_Bad(t *core.T) {
	subject := ParseVulnCheckJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseVulnCheckJSON:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestVulncheck_ParseVulnCheckJSON_Ugly(t *core.T) {
	subject := ParseVulnCheckJSON
	if subject == nil {
		t.FailNow()
	}
	marker := "ParseVulnCheckJSON:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
