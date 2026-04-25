package lint

import (
	"encoding/json"
	"strings"

	coreerr "dappco.re/go/core/log"
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
		return nil, coreerr.E("Toolkit.VulnCheck", "govulncheck not installed or not available", err)
	}

	return ParseVulnCheckJSON(stdout, stderr)
}

// ParseVulnCheckJSON parses govulncheck -json output (newline-delimited JSON messages).
func ParseVulnCheckJSON(stdout, stderr string) (*VulnResult, error) {
	result := &VulnResult{}

	osvMap := make(map[string]*govulncheckOSV)
	var findings []govulncheckFind

	for line := range strings.SplitSeq(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg govulncheckMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, coreerr.E("ParseVulnCheckJSON", "invalid govulncheck JSON output", err)
		}

		if msg.Config != nil {
			result.Module = msg.Config.ModulePath
		}
		if msg.OSV != nil {
			osvMap[msg.OSV.ID] = msg.OSV
		}
		if msg.Finding != nil {
			findings = append(findings, *msg.Finding)
		}
	}

	for _, f := range findings {
		finding := VulnFinding{
			ID: f.OSV,
		}

		if len(f.Trace) > 0 {
			last := f.Trace[len(f.Trace)-1]
			finding.Package = last.Package
			finding.CalledFunction = last.Function
			finding.ModulePath = last.Module

			for _, tr := range f.Trace {
				if tr.Version != "" {
					finding.FixedVersion = tr.Version
					break
				}
			}
		}

		if osv, ok := osvMap[f.OSV]; ok {
			finding.Description = osv.Summary
			finding.Aliases = osv.Aliases

			for _, aff := range osv.Affected {
				for _, r := range aff.Ranges {
					for _, ev := range r.Events {
						if ev.Fixed != "" && finding.FixedVersion == "" {
							finding.FixedVersion = ev.Fixed
						}
					}
				}
			}
		}

		result.Findings = append(result.Findings, finding)
	}

	return result, nil
}
