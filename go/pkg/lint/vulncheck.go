package lint

import core "dappco.re/go"

// VulnFinding represents a single vulnerability found by govulncheck.
type VulnFinding struct {
	ID             string   `json:"id"`              // e.g. GO-2024-1234
	Aliases        []string `json:"aliases"`         // CVE/GHSA aliases
	Package        string   `json:"package"`         // Affected package path
	CalledFunction string   `json:"called_function"` // Function in call stack (empty if not called)
	Description    string   `json:"description"`     // Human-readable summary
	Severity       string   `json:"severity"`        // "HIGH", "MEDIUM", "LOW", or empty
	FixedVersion   string   `json:"fixed_version"`   // Version that contains the fix
	ModulePath     string   `json:"module_path"`     // Go module path
}

// VulnResult holds the complete output of a vulnerability scan.
type VulnResult struct {
	Findings []VulnFinding `json:"findings"`
	Module   string        `json:"module"`
}

// --- govulncheck JSON wire types ---

type govulncheckMessage struct {
	Config   *govulncheckConfig `json:"config,omitempty"`
	OSV      *govulncheckOSV    `json:"osv,omitempty"`
	Finding  *govulncheckFind   `json:"finding,omitempty"`
	Progress any                `json:"progress,omitempty"`
}

type govulnMessagesResult struct {
	Module   string
	OSVMap   map[string]*govulncheckOSV
	Findings []govulncheckFind
}

type govulnMessageLineResult struct {
	Message govulncheckMessage
	OK      bool
}

type govulncheckConfig struct {
	GoVersion  string `json:"go_version"`
	ModulePath string `json:"module_path"`
}

type govulncheckOSV struct {
	ID       string              `json:"id"`
	Aliases  []string            `json:"aliases"`
	Summary  string              `json:"summary"`
	Affected []govulncheckAffect `json:"affected"`
}

type govulncheckAffect struct {
	Package  *govulncheckPkg       `json:"package,omitempty"`
	Ranges   []govulncheckRange    `json:"ranges,omitempty"`
	Severity []govulncheckSeverity `json:"database_specific,omitempty"`
}

type govulncheckPkg struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type govulncheckRange struct {
	Events []govulncheckEvent `json:"events"`
}

type govulncheckEvent struct {
	Fixed string `json:"fixed,omitempty"`
}

type govulncheckSeverity struct {
	Severity string `json:"severity,omitempty"`
}

type govulncheckFind struct {
	OSV   string             `json:"osv"`
	Trace []govulncheckTrace `json:"trace"`
}

type govulncheckTrace struct {
	Module   string `json:"module,omitempty"`
	Package  string `json:"package,omitempty"`
	Function string `json:"function,omitempty"`
	Version  string `json:"version,omitempty"`
}

// VulnCheck runs govulncheck -json on the given module path and parses
// the output into structured VulnFindings.
func (t *Toolkit) VulnCheck(modulePath string) core.Result {
	if modulePath == "" {
		modulePath = "./..."
	}

	run := t.Run("govulncheck", "-json", modulePath).Value.(CommandOutput)
	if run.Err != nil && run.ExitCode == -1 {
		return core.Fail(core.E("Toolkit.VulnCheck", "govulncheck not installed or not available", run.Err))
	}

	return ParseVulnCheckJSON(run.Stdout, run.Stderr)
}

// ParseVulnCheckJSON parses govulncheck -json output (newline-delimited JSON messages).
func ParseVulnCheckJSON(stdout, stderr string) core.Result {
	result := &VulnResult{}
	parsed := parseGovulnMessages(stdout)
	if !parsed.OK {
		return parsed
	}
	messages := parsed.Value.(govulnMessagesResult)
	result.Module = messages.Module

	for _, f := range messages.Findings {
		result.Findings = append(result.Findings, vulnFindingFromGovuln(f, messages.OSVMap))
	}

	return core.Ok(result)
}

func parseGovulnMessages(stdout string) core.Result {
	osvMap := make(map[string]*govulncheckOSV)
	var findings []govulncheckFind
	var module string
	for _, line := range core.Split(stdout, "\n") {
		parsed := parseGovulnMessageLine(line)
		if !parsed.OK {
			return parsed
		}
		lineResult := parsed.Value.(govulnMessageLineResult)
		if !lineResult.OK {
			continue
		}
		msg := lineResult.Message
		if msg.Config != nil {
			module = msg.Config.ModulePath
		}
		if msg.OSV != nil {
			osvMap[msg.OSV.ID] = msg.OSV
		}
		if msg.Finding != nil {
			findings = append(findings, *msg.Finding)
		}
	}
	return core.Ok(govulnMessagesResult{Module: module, OSVMap: osvMap, Findings: findings})
}

func parseGovulnMessageLine(line string) core.Result {
	line = core.Trim(line)
	if line == "" {
		return core.Ok(govulnMessageLineResult{})
	}
	var msg govulncheckMessage
	if r := core.JSONUnmarshal([]byte(line), &msg); !r.OK {
		err, _ := r.Value.(error)
		return core.Fail(core.E("ParseVulnCheckJSON", "invalid govulncheck JSON output", err))
	}
	return core.Ok(govulnMessageLineResult{Message: msg, OK: true})
}

func vulnFindingFromGovuln(f govulncheckFind, osvMap map[string]*govulncheckOSV) VulnFinding {
	finding := VulnFinding{ID: f.OSV}
	applyGovulnTrace(&finding, f.Trace)
	if osv, ok := osvMap[f.OSV]; ok {
		applyGovulnOSV(&finding, osv)
	}
	return finding
}

func applyGovulnTrace(finding *VulnFinding, trace []govulncheckTrace) {
	if len(trace) == 0 {
		return
	}
	last := trace[len(trace)-1]
	finding.Package = last.Package
	finding.CalledFunction = last.Function
	finding.ModulePath = last.Module
	finding.FixedVersion = firstTraceFixedVersion(trace)
}

func firstTraceFixedVersion(trace []govulncheckTrace) string {
	for _, tr := range trace {
		if tr.Version != "" {
			return tr.Version
		}
	}
	return ""
}

func applyGovulnOSV(finding *VulnFinding, osv *govulncheckOSV) {
	finding.Description = osv.Summary
	finding.Aliases = osv.Aliases
	if finding.FixedVersion == "" {
		finding.FixedVersion = firstOSVFixedVersion(osv)
	}
}

func firstOSVFixedVersion(osv *govulncheckOSV) string {
	for _, aff := range osv.Affected {
		for _, r := range aff.Ranges {
			for _, ev := range r.Events {
				if ev.Fixed != "" {
					return ev.Fixed
				}
			}
		}
	}
	return ""
}
