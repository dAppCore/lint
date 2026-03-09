package php

// QAOptions configures the full QA pipeline.
type QAOptions struct {
	Dir   string
	Quick bool // Only run quick checks
	Full  bool // Run all stages including slow checks
	Fix   bool // Auto-fix issues where possible
	JSON  bool // Output results as JSON
}

// QAStage represents a stage in the QA pipeline.
type QAStage string

const (
	QAStageQuick    QAStage = "quick"    // fast checks: audit, fmt, stan
	QAStageStandard QAStage = "standard" // standard checks + tests
	QAStageFull     QAStage = "full"     // all including slow scans
)

// QACheckResult holds the result of a single QA check.
type QACheckResult struct {
	Name     string
	Stage    QAStage
	Passed   bool
	Duration string
	Error    error
	Output   string
}

// QAResult holds the results of the full QA pipeline.
type QAResult struct {
	Stages  []QAStage
	Checks  []QACheckResult
	Passed  bool
	Summary string
}

// GetQAStages returns the stages to run based on options.
func GetQAStages(opts QAOptions) []QAStage {
	if opts.Quick {
		return []QAStage{QAStageQuick}
	}
	if opts.Full {
		return []QAStage{QAStageQuick, QAStageStandard, QAStageFull}
	}
	return []QAStage{QAStageQuick, QAStageStandard}
}

// GetQAChecks returns the checks for a given stage.
func GetQAChecks(dir string, stage QAStage) []string {
	switch stage {
	case QAStageQuick:
		return []string{"audit", "fmt", "stan"}
	case QAStageStandard:
		checks := []string{}
		if _, found := DetectPsalm(dir); found {
			checks = append(checks, "psalm")
		}
		checks = append(checks, "test")
		return checks
	case QAStageFull:
		checks := []string{}
		if DetectRector(dir) {
			checks = append(checks, "rector")
		}
		if DetectInfection(dir) {
			checks = append(checks, "infection")
		}
		return checks
	}
	return nil
}
