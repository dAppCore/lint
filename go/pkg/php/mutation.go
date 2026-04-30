package php

import (
	"context"
	"io"

	core "dappco.re/go"
)

// InfectionOptions configures Infection mutation testing.
type InfectionOptions struct {
	Dir           string
	MinMSI        int    // Minimum mutation score indicator (0-100)
	MinCoveredMSI int    // Minimum covered mutation score (0-100)
	Threads       int    // Number of parallel threads
	Filter        string // Filter files by pattern
	OnlyCovered   bool   // Only mutate covered code
	Output        io.Writer
}

// DetectInfection checks if Infection is available in the project.
func DetectInfection(dir string) bool {
	// Check for infection config files
	configs := []string{"infection.json", "infection.json5", "infection.json.dist"}
	for _, config := range configs {
		if fileExists(core.PathJoin(dir, config)) {
			return true
		}
	}

	// Check for vendor binary
	infectionBin := core.PathJoin(dir, "vendor", "bin", "infection")
	if fileExists(infectionBin) {
		return true
	}

	return false
}

// RunInfection runs Infection mutation testing.
func RunInfection(ctx context.Context, opts InfectionOptions) core.Result {
	if opts.Dir == "" {
		cwd := core.Getwd()
		if !cwd.OK {
			err, _ := cwd.Value.(error)
			return core.Fail(core.E("php.RunInfection", "get working directory", err))
		}
		opts.Dir = cwd.Value.(string)
	}

	if opts.Output == nil {
		opts.Output = core.Stdout()
	}

	// Build command
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "infection")
	cmdName := "infection"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	var args []string

	// Set defaults
	minMSI := opts.MinMSI
	if minMSI == 0 {
		minMSI = 50
	}
	minCoveredMSI := opts.MinCoveredMSI
	if minCoveredMSI == 0 {
		minCoveredMSI = 70
	}
	threads := opts.Threads
	if threads == 0 {
		threads = 4
	}

	args = append(args, core.Sprintf("--min-msi=%d", minMSI))
	args = append(args, core.Sprintf("--min-covered-msi=%d", minCoveredMSI))
	args = append(args, core.Sprintf("--threads=%d", threads))

	if opts.Filter != "" {
		args = append(args, "--filter="+opts.Filter)
	}

	if opts.OnlyCovered {
		args = append(args, "--only-covered")
	}

	return runPHPCommand(ctx, opts.Dir, cmdName, args, opts.Output, nil)
}

// buildInfectionCommand builds the command for running Infection (exported for testing).
func buildInfectionCommand(opts InfectionOptions) (string, []string) {
	vendorBin := core.PathJoin(opts.Dir, "vendor", "bin", "infection")
	cmdName := "infection"
	if fileExists(vendorBin) {
		cmdName = vendorBin
	}

	var args []string

	minMSI := opts.MinMSI
	if minMSI == 0 {
		minMSI = 50
	}
	minCoveredMSI := opts.MinCoveredMSI
	if minCoveredMSI == 0 {
		minCoveredMSI = 70
	}
	threads := opts.Threads
	if threads == 0 {
		threads = 4
	}

	args = append(args, core.Sprintf("--min-msi=%d", minMSI))
	args = append(args, core.Sprintf("--min-covered-msi=%d", minCoveredMSI))
	args = append(args, core.Sprintf("--threads=%d", threads))

	if opts.Filter != "" {
		args = append(args, "--filter="+opts.Filter)
	}

	if opts.OnlyCovered {
		args = append(args, "--only-covered")
	}

	return cmdName, args
}
