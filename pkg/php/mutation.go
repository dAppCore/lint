package php

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	coreerr "forge.lthn.ai/core/go-log"
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
		if fileExists(filepath.Join(dir, config)) {
			return true
		}
	}

	// Check for vendor binary
	infectionBin := filepath.Join(dir, "vendor", "bin", "infection")
	if fileExists(infectionBin) {
		return true
	}

	return false
}

// RunInfection runs Infection mutation testing.
func RunInfection(ctx context.Context, opts InfectionOptions) error {
	if opts.Dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return coreerr.E("php.RunInfection", "get working directory", err)
		}
		opts.Dir = cwd
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Build command
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "infection")
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

	args = append(args, fmt.Sprintf("--min-msi=%d", minMSI))
	args = append(args, fmt.Sprintf("--min-covered-msi=%d", minCoveredMSI))
	args = append(args, fmt.Sprintf("--threads=%d", threads))

	if opts.Filter != "" {
		args = append(args, "--filter="+opts.Filter)
	}

	if opts.OnlyCovered {
		args = append(args, "--only-covered")
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = opts.Dir
	cmd.Stdout = opts.Output
	cmd.Stderr = opts.Output

	return cmd.Run()
}

// buildInfectionCommand builds the command for running Infection (exported for testing).
func buildInfectionCommand(opts InfectionOptions) (string, []string) {
	vendorBin := filepath.Join(opts.Dir, "vendor", "bin", "infection")
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

	args = append(args, fmt.Sprintf("--min-msi=%d", minMSI))
	args = append(args, fmt.Sprintf("--min-covered-msi=%d", minCoveredMSI))
	args = append(args, fmt.Sprintf("--threads=%d", threads))

	if opts.Filter != "" {
		args = append(args, "--filter="+opts.Filter)
	}

	if opts.OnlyCovered {
		args = append(args, "--only-covered")
	}

	return cmdName, args
}
