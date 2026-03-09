package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DetectRector
// =============================================================================

func TestDetectRector_Good_RectorConfig(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "rector.php"))

	assert.True(t, DetectRector(dir))
}

func TestDetectRector_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "rector"))

	assert.True(t, DetectRector(dir))
}

func TestDetectRector_Bad_Empty(t *testing.T) {
	dir := t.TempDir()

	assert.False(t, DetectRector(dir))
}

// =============================================================================
// buildRectorCommand
// =============================================================================

func TestBuildRectorCommand_Good_Defaults(t *testing.T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir}

	cmdName, args := buildRectorCommand(opts)
	assert.Equal(t, "rector", cmdName)
	// Fix is false by default, so --dry-run should be present
	assert.Contains(t, args, "process")
	assert.Contains(t, args, "--dry-run")
}

func TestBuildRectorCommand_Good_Fix(t *testing.T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, Fix: true}

	cmdName, args := buildRectorCommand(opts)
	assert.Equal(t, "rector", cmdName)
	assert.Contains(t, args, "process")
	assert.NotContains(t, args, "--dry-run")
}

func TestBuildRectorCommand_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "rector")
	mkFile(t, vendorBin)

	opts := RectorOptions{Dir: dir}
	cmdName, _ := buildRectorCommand(opts)
	assert.Equal(t, vendorBin, cmdName)
}

func TestBuildRectorCommand_Good_Diff(t *testing.T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, Diff: true}

	_, args := buildRectorCommand(opts)
	assert.Contains(t, args, "--output-format")
	assert.Contains(t, args, "diff")
}

func TestBuildRectorCommand_Good_ClearCache(t *testing.T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, ClearCache: true}

	_, args := buildRectorCommand(opts)
	assert.Contains(t, args, "--clear-cache")
}

func TestBuildRectorCommand_Good_AllFlags(t *testing.T) {
	dir := t.TempDir()
	opts := RectorOptions{
		Dir:        dir,
		Fix:        true,
		Diff:       true,
		ClearCache: true,
	}

	_, args := buildRectorCommand(opts)
	assert.Contains(t, args, "process")
	assert.NotContains(t, args, "--dry-run")
	assert.Contains(t, args, "--output-format")
	assert.Contains(t, args, "diff")
	assert.Contains(t, args, "--clear-cache")
}

func TestRectorOptions_Defaults(t *testing.T) {
	opts := RectorOptions{}
	assert.Empty(t, opts.Dir)
	assert.False(t, opts.Fix)
	assert.False(t, opts.Diff)
	assert.False(t, opts.ClearCache)
	assert.Nil(t, opts.Output)
}

func TestDetectRector_Good_BothConfigAndBinary(t *testing.T) {
	dir := t.TempDir()

	// Create both config and vendor binary
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rector.php"), []byte("<?php\n"), 0644))
	mkFile(t, filepath.Join(dir, "vendor", "bin", "rector"))

	assert.True(t, DetectRector(dir))
}
