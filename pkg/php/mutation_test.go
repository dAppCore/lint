package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

// =============================================================================
// DetectInfection
// =============================================================================

func TestDetectInfection_Good_InfectionJSON(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json"))

	AssertTrue(t, DetectInfection(dir))
}

func TestDetectInfection_Good_InfectionJSON5(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json5"))

	AssertTrue(t, DetectInfection(dir))
}

func TestDetectInfection_Good_InfectionJSONDist(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json.dist"))

	AssertTrue(t, DetectInfection(dir))
}

func TestDetectInfection_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "infection"))

	AssertTrue(t, DetectInfection(dir))
}

func TestDetectInfection_Bad_Empty(t *T) {
	dir := t.TempDir()

	AssertFalse(t, DetectInfection(dir))
}

// =============================================================================
// buildInfectionCommand
// =============================================================================

func TestBuildInfectionCommand_Good_Defaults(t *T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir}

	cmdName, args := buildInfectionCommand(opts)
	AssertEqual(t, "infection", cmdName)
	// Defaults: minMSI=50, minCoveredMSI=70, threads=4
	AssertContains(t, args, "--min-msi=50")
	AssertContains(t, args, "--min-covered-msi=70")
	AssertContains(t, args, "--threads=4")
}

func TestBuildInfectionCommand_Good_CustomThresholds(t *T) {
	dir := t.TempDir()
	opts := InfectionOptions{
		Dir:           dir,
		MinMSI:        80,
		MinCoveredMSI: 90,
		Threads:       8,
	}

	_, args := buildInfectionCommand(opts)
	AssertContains(t, args, "--min-msi=80")
	AssertContains(t, args, "--min-covered-msi=90")
	AssertContains(t, args, "--threads=8")
}

func TestBuildInfectionCommand_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "infection")
	mkFile(t, vendorBin)

	opts := InfectionOptions{Dir: dir}
	cmdName, _ := buildInfectionCommand(opts)
	AssertEqual(t, vendorBin, cmdName)
}

func TestBuildInfectionCommand_Good_Filter(t *T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir, Filter: "src/Models"}

	_, args := buildInfectionCommand(opts)
	AssertContains(t, args, "--filter=src/Models")
}

func TestBuildInfectionCommand_Good_OnlyCovered(t *T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir, OnlyCovered: true}

	_, args := buildInfectionCommand(opts)
	AssertContains(t, args, "--only-covered")
}

func TestBuildInfectionCommand_Good_AllFlags(t *T) {
	dir := t.TempDir()
	opts := InfectionOptions{
		Dir:           dir,
		MinMSI:        60,
		MinCoveredMSI: 80,
		Threads:       2,
		Filter:        "app/",
		OnlyCovered:   true,
	}

	_, args := buildInfectionCommand(opts)
	AssertContains(t, args, "--min-msi=60")
	AssertContains(t, args, "--min-covered-msi=80")
	AssertContains(t, args, "--threads=2")
	AssertContains(t, args, "--filter=app/")
	AssertContains(t, args, "--only-covered")
}

func TestInfectionOptions_Defaults(t *T) {
	opts := InfectionOptions{}
	AssertEmpty(t, opts.Dir)
	AssertEqual(t, 0, opts.MinMSI)
	AssertEqual(t, 0, opts.MinCoveredMSI)
	AssertEqual(t, 0, opts.Threads)
	AssertEmpty(t, opts.Filter)
	AssertFalse(t, opts.OnlyCovered)
	AssertNil(t, opts.Output)
}

func TestDetectInfection_Good_BothConfigAndBinary(t *T) {
	dir := t.TempDir()

	// Create both config and vendor binary
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "infection.json5"), []byte("{}"), 0644))
	mkFile(t, filepath.Join(dir, "vendor", "bin", "infection"))

	AssertTrue(t, DetectInfection(dir))
}
