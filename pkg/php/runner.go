package php

import (
	"path/filepath"
)

type RunSpec struct {
	Name         string
	Command      string
	Args         []string
	Dir          string
	Env          []string
	After        []string
	AllowFailure bool
}

// QARunner builds process run specs for PHP QA checks.
type QARunner struct {
	dir string
	fix bool
}

// NewQARunner creates a QA runner for the given directory.
func NewQARunner(dir string, fix bool) *QARunner {
	return &QARunner{dir: dir, fix: fix}
}

// BuildSpecs creates RunSpecs for the given QA checks.
func (r *QARunner) BuildSpecs(checks []string) []RunSpec {
	specs := make([]RunSpec, 0, len(checks))
	for _, check := range checks {
		spec := r.buildSpec(check)
		if spec != nil {
			specs = append(specs, *spec)
		}
	}
	filterSpecDependencies(specs)
	return specs
}

// filterSpecDependencies drops dependencies for checks that were not requested.
func filterSpecDependencies(specs []RunSpec) {
	available := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		available[spec.Name] = struct{}{}
	}

	for index := range specs {
		filtered := specs[index].After[:0]
		for _, dependency := range specs[index].After {
			if _, ok := available[dependency]; ok {
				filtered = append(filtered, dependency)
			}
		}
		if len(filtered) == 0 {
			specs[index].After = nil
			continue
		}
		specs[index].After = filtered
	}
}

// buildSpec creates a RunSpec for a single check.
func (r *QARunner) buildSpec(check string) *RunSpec {
	switch check {
	case "audit":
		return r.auditSpec()
	case "fmt":
		return r.fmtSpec()
	case "stan":
		return r.stanSpec()
	case "psalm":
		return r.psalmSpec()
	case "test":
		return r.testSpec()
	case "rector":
		return r.rectorSpec()
	case "infection":
		return r.infectionSpec()
	}
	return nil
}

func (r *QARunner) auditSpec() *RunSpec {
	return &RunSpec{
		Name:    "audit",
		Command: "composer",
		Args:    []string{"audit", "--format=summary"},
		Dir:     r.dir,
	}
}

func (r *QARunner) fmtSpec() *RunSpec {
	_, found := DetectFormatter(r.dir)
	if !found {
		return nil
	}
	args := []string{}
	if !r.fix {
		args = append(args, "--test")
	}
	return &RunSpec{
		Name:    "fmt",
		Command: vendorBinOrDefault(r.dir, "pint"),
		Args:    args,
		Dir:     r.dir,
		After:   []string{"audit"},
	}
}

func (r *QARunner) stanSpec() *RunSpec {
	_, found := DetectAnalyser(r.dir)
	if !found {
		return nil
	}
	return &RunSpec{
		Name:    "stan",
		Command: vendorBinOrDefault(r.dir, "phpstan"),
		Args:    []string{"analyse", "--no-progress"},
		Dir:     r.dir,
		After:   []string{"fmt"},
	}
}

func (r *QARunner) psalmSpec() *RunSpec {
	_, found := DetectPsalm(r.dir)
	if !found {
		return nil
	}
	args := []string{"--no-progress"}
	if r.fix {
		args = append(args, "--alter", "--issues=all")
	}
	return &RunSpec{
		Name:    "psalm",
		Command: vendorBinOrDefault(r.dir, "psalm"),
		Args:    args,
		Dir:     r.dir,
		After:   []string{"stan"},
	}
}

func (r *QARunner) testSpec() *RunSpec {
	cmd := firstExistingVendorBin(r.dir, "pest", "phpunit")
	if cmd == "" {
		return nil
	}
	after := []string{"stan"}
	if _, found := DetectPsalm(r.dir); found {
		after = []string{"psalm"}
	}
	return &RunSpec{
		Name:    "test",
		Command: cmd,
		Args:    []string{},
		Dir:     r.dir,
		After:   after,
	}
}

func (r *QARunner) rectorSpec() *RunSpec {
	if !DetectRector(r.dir) {
		return nil
	}
	args := []string{"process"}
	if !r.fix {
		args = append(args, "--dry-run")
	}
	return &RunSpec{
		Name:         "rector",
		Command:      vendorBinOrDefault(r.dir, "rector"),
		Args:         args,
		Dir:          r.dir,
		After:        []string{"test"},
		AllowFailure: true,
	}
}

func (r *QARunner) infectionSpec() *RunSpec {
	if !DetectInfection(r.dir) {
		return nil
	}
	return &RunSpec{
		Name:         "infection",
		Command:      vendorBinOrDefault(r.dir, "infection"),
		Args:         []string{"--min-msi=50", "--min-covered-msi=70", "--threads=4"},
		Dir:          r.dir,
		After:        []string{"test"},
		AllowFailure: true,
	}
}

func vendorBinOrDefault(dir string, name string) string {
	vendorBin := filepath.Join(dir, "vendor", "bin", name)
	if fileExists(vendorBin) {
		return vendorBin
	}
	return name
}

func firstExistingVendorBin(dir string, names ...string) string {
	for _, name := range names {
		vendorBin := filepath.Join(dir, "vendor", "bin", name)
		if fileExists(vendorBin) {
			return vendorBin
		}
	}
	return ""
}

// QARunResult holds the results of running QA checks.
type QARunResult struct {
	Passed       bool               `json:"passed"`
	Duration     string             `json:"duration"`
	Results      []QACheckRunResult `json:"results"`
	PassedCount  int                `json:"passed_count"`
	FailedCount  int                `json:"failed_count"`
	SkippedCount int                `json:"skipped_count"`
}

// QACheckRunResult holds the result of a single QA check.
type QACheckRunResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Skipped  bool   `json:"skipped"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
	Output   string `json:"output,omitempty"`
}

// GetIssueMessage returns a human-readable issue description for a failed check.
func (r QACheckRunResult) GetIssueMessage() string {
	if r.Passed || r.Skipped {
		return ""
	}
	switch r.Name {
	case "audit":
		return "found vulnerabilities"
	case "fmt":
		return "found style issues"
	case "stan":
		return "found analysis errors"
	case "psalm":
		return "found type errors"
	case "test":
		return "tests failed"
	case "rector":
		return "found refactoring suggestions"
	case "infection":
		return "mutation testing did not pass"
	default:
		return "found issues"
	}
}
