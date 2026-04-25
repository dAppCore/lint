package php

import (
	"path/filepath"

	process "dappco.re/go/process"
)

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
func (r *QARunner) BuildSpecs(checks []string) []process.RunSpec {
	specs := make([]process.RunSpec, 0, len(checks))
	for _, check := range checks {
		spec := r.buildSpec(check)
		if spec != nil {
			specs = append(specs, *spec)
		}
	}
	return specs
}

// buildSpec creates a RunSpec for a single check.
func (r *QARunner) buildSpec(check string) *process.RunSpec {
	switch check {
	case "audit":
		return &process.RunSpec{
			Name:    "audit",
			Command: "composer",
			Args:    []string{"audit", "--format=summary"},
			Dir:     r.dir,
		}

	case "fmt":
		_, found := DetectFormatter(r.dir)
		if !found {
			return nil
		}
		vendorBin := filepath.Join(r.dir, "vendor", "bin", "pint")
		cmd := "pint"
		if fileExists(vendorBin) {
			cmd = vendorBin
		}
		args := []string{}
		if !r.fix {
			args = append(args, "--test")
		}
		return &process.RunSpec{
			Name:    "fmt",
			Command: cmd,
			Args:    args,
			Dir:     r.dir,
			After:   []string{"audit"},
		}

	case "stan":
		_, found := DetectAnalyser(r.dir)
		if !found {
			return nil
		}
		vendorBin := filepath.Join(r.dir, "vendor", "bin", "phpstan")
		cmd := "phpstan"
		if fileExists(vendorBin) {
			cmd = vendorBin
		}
		return &process.RunSpec{
			Name:    "stan",
			Command: cmd,
			Args:    []string{"analyse", "--no-progress"},
			Dir:     r.dir,
			After:   []string{"fmt"},
		}

	case "psalm":
		_, found := DetectPsalm(r.dir)
		if !found {
			return nil
		}
		vendorBin := filepath.Join(r.dir, "vendor", "bin", "psalm")
		cmd := "psalm"
		if fileExists(vendorBin) {
			cmd = vendorBin
		}
		args := []string{"--no-progress"}
		if r.fix {
			args = append(args, "--alter", "--issues=all")
		}
		return &process.RunSpec{
			Name:    "psalm",
			Command: cmd,
			Args:    args,
			Dir:     r.dir,
			After:   []string{"stan"},
		}

	case "test":
		pestBin := filepath.Join(r.dir, "vendor", "bin", "pest")
		phpunitBin := filepath.Join(r.dir, "vendor", "bin", "phpunit")
		var cmd string
		if fileExists(pestBin) {
			cmd = pestBin
		} else if fileExists(phpunitBin) {
			cmd = phpunitBin
		} else {
			return nil
		}
		after := []string{"stan"}
		if _, found := DetectPsalm(r.dir); found {
			after = []string{"psalm"}
		}
		return &process.RunSpec{
			Name:    "test",
			Command: cmd,
			Args:    []string{},
			Dir:     r.dir,
			After:   after,
		}

	case "rector":
		if !DetectRector(r.dir) {
			return nil
		}
		vendorBin := filepath.Join(r.dir, "vendor", "bin", "rector")
		cmd := "rector"
		if fileExists(vendorBin) {
			cmd = vendorBin
		}
		args := []string{"process"}
		if !r.fix {
			args = append(args, "--dry-run")
		}
		return &process.RunSpec{
			Name:         "rector",
			Command:      cmd,
			Args:         args,
			Dir:          r.dir,
			After:        []string{"test"},
			AllowFailure: true,
		}

	case "infection":
		if !DetectInfection(r.dir) {
			return nil
		}
		vendorBin := filepath.Join(r.dir, "vendor", "bin", "infection")
		cmd := "infection"
		if fileExists(vendorBin) {
			cmd = vendorBin
		}
		return &process.RunSpec{
			Name:         "infection",
			Command:      cmd,
			Args:         []string{"--min-msi=50", "--min-covered-msi=70", "--threads=4"},
			Dir:          r.dir,
			After:        []string{"test"},
			AllowFailure: true,
		}
	}
	return nil
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
