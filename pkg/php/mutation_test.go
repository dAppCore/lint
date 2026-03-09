package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DetectInfection
// =============================================================================

func TestDetectInfection_Good_InfectionJSON(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json"))

	assert.True(t, DetectInfection(dir))
}

func TestDetectInfection_Good_InfectionJSON5(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json5"))

	assert.True(t, DetectInfection(dir))
}

func TestDetectInfection_Good_InfectionJSONDist(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "infection.json.dist"))

	assert.True(t, DetectInfection(dir))
}

func TestDetectInfection_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "infection"))

	assert.True(t, DetectInfection(dir))
}

func TestDetectInfection_Bad_Empty(t *testing.T) {
	dir := t.TempDir()

	assert.False(t, DetectInfection(dir))
}

// =============================================================================
// buildInfectionCommand
// =============================================================================

func TestBuildInfectionCommand_Good_Defaults(t *testing.T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir}

	cmdName, args := buildInfectionCommand(opts)
	assert.Equal(t, "infection", cmdName)
	// Defaults: minMSI=50, minCoveredMSI=70, threads=4
	assert.Contains(t, args, "--min-msi=50")
	assert.Contains(t, args, "--min-covered-msi=70")
	assert.Contains(t, args, "--threads=4")
}

func TestBuildInfectionCommand_Good_CustomThresholds(t *testing.T) {
	dir := t.TempDir()
	opts := InfectionOptions{
		Dir:           dir,
		MinMSI:        80,
		MinCoveredMSI: 90,
		Threads:       8,
	}

	_, args := buildInfectionCommand(opts)
	assert.Contains(t, args, "--min-msi=80")
	assert.Contains(t, args, "--min-covered-msi=90")
	assert.Contains(t, args, "--threads=8")
}

func TestBuildInfectionCommand_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "infection")
	mkFile(t, vendorBin)

	opts := InfectionOptions{Dir: dir}
	cmdName, _ := buildInfectionCommand(opts)
	assert.Equal(t, vendorBin, cmdName)
}

func TestBuildInfectionCommand_Good_Filter(t *testing.T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir, Filter: "src/Models"}

	_, args := buildInfectionCommand(opts)
	assert.Contains(t, args, "--filter=src/Models")
}

func TestBuildInfectionCommand_Good_OnlyCovered(t *testing.T) {
	dir := t.TempDir()
	opts := InfectionOptions{Dir: dir, OnlyCovered: true}

	_, args := buildInfectionCommand(opts)
	assert.Contains(t, args, "--only-covered")
}

func TestBuildInfectionCommand_Good_AllFlags(t *testing.T) {
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
	assert.Contains(t, args, "--min-msi=60")
	assert.Contains(t, args, "--min-covered-msi=80")
	assert.Contains(t, args, "--threads=2")
	assert.Contains(t, args, "--filter=app/")
	assert.Contains(t, args, "--only-covered")
}

func TestInfectionOptions_Defaults(t *testing.T) {
	opts := InfectionOptions{}
	assert.Empty(t, opts.Dir)
	assert.Equal(t, 0, opts.MinMSI)
	assert.Equal(t, 0, opts.MinCoveredMSI)
	assert.Equal(t, 0, opts.Threads)
	assert.Empty(t, opts.Filter)
	assert.False(t, opts.OnlyCovered)
	assert.Nil(t, opts.Output)
}

func TestDetectInfection_Good_BothConfigAndBinary(t *testing.T) {
	dir := t.TempDir()

	// Create both config and vendor binary
	require.NoError(t, os.WriteFile(filepath.Join(dir, "infection.json5"), []byte("{}"), 0644))
	mkFile(t, filepath.Join(dir, "vendor", "bin", "infection"))

	assert.True(t, DetectInfection(dir))
}
