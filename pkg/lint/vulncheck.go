package lint

import (
	"encoding/json"
	"strings"

	core "dappco.re/go"
)

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
	Progress *json.RawMessage   `json:"progress,omitempty"`
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
func (t *Toolkit) VulnCheck(modulePath string) (*VulnResult, error) {
	if modulePath == "" {
		modulePath = "./..."
	}

	stdout, stderr, exitCode, err := t.Run("govulncheck", "-json", modulePath)
	if err != nil && exitCode == -1 {
		return nil, core.E("Toolkit.VulnCheck", "govulncheck not installed or not available", err)
	}

	return ParseVulnCheckJSON(stdout, stderr)
}

// ParseVulnCheckJSON parses govulncheck -json output (newline-delimited JSON messages).
func ParseVulnCheckJSON(stdout, stderr string) (*VulnResult, error) {
	result := &VulnResult{}
	module, osvMap, findings, err := parseGovulnMessages(stdout)
	if err != nil {
		return nil, err
	}
	result.Module = module

	for _, f := range findings {
		result.Findings = append(result.Findings, vulnFindingFromGovuln(f, osvMap))
	}

	return result, nil
}

func parseGovulnMessages(stdout string) (string, map[string]*govulncheckOSV, []govulncheckFind, error) {
	osvMap := make(map[string]*govulncheckOSV)
	var findings []govulncheckFind
	var module string
	for line := range strings.SplitSeq(stdout, "\n") {
		msg, ok, err := parseGovulnMessageLine(line)
		if err != nil {
			return "", nil, nil, err
		}
		if !ok {
			continue
		}
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
	return module, osvMap, findings, nil
}

func parseGovulnMessageLine(line string) (govulncheckMessage, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return govulncheckMessage{}, false, nil
	}
	var msg govulncheckMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return govulncheckMessage{}, false, core.E("ParseVulnCheckJSON", "invalid govulncheck JSON output", err)
	}
	return msg, true, nil
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
