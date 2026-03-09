package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormatter_PintConfig(t *testing.T) {
	dir := t.TempDir()

	// Create pint.json
	err := os.WriteFile(filepath.Join(dir, "pint.json"), []byte("{}"), 0644)
	require.NoError(t, err)

	ft, found := DetectFormatter(dir)
	assert.True(t, found)
	assert.Equal(t, FormatterPint, ft)
}

func TestDetectFormatter_VendorBinary(t *testing.T) {
	dir := t.TempDir()

	// Create vendor/bin/pint
	binDir := filepath.Join(dir, "vendor", "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(binDir, "pint"), []byte("#!/bin/sh\n"), 0755)
	require.NoError(t, err)

	ft, found := DetectFormatter(dir)
	assert.True(t, found)
	assert.Equal(t, FormatterPint, ft)
}

func TestDetectFormatter_Empty(t *testing.T) {
	dir := t.TempDir()

	ft, found := DetectFormatter(dir)
	assert.False(t, found)
	assert.Equal(t, FormatterType(""), ft)
}

func TestBuildPintCommand_Defaults(t *testing.T) {
	dir := t.TempDir()

	opts := FormatOptions{Dir: dir}
	cmdName, args := buildPintCommand(opts)

	// No vendor binary, so fallback to bare "pint"
	assert.Equal(t, "pint", cmdName)
	// Fix is false by default, so --test should be present
	assert.Contains(t, args, "--test")
}

func TestBuildPintCommand_Fix(t *testing.T) {
	dir := t.TempDir()

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, args := buildPintCommand(opts)

	assert.Equal(t, "pint", cmdName)
	assert.NotContains(t, args, "--test")
}

func TestBuildPintCommand_VendorBinary(t *testing.T) {
	dir := t.TempDir()

	binDir := filepath.Join(dir, "vendor", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "pint"), []byte("#!/bin/sh\n"), 0755))

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, _ := buildPintCommand(opts)

	assert.Equal(t, filepath.Join(dir, "vendor", "bin", "pint"), cmdName)
}

func TestBuildPintCommand_AllFlags(t *testing.T) {
	dir := t.TempDir()

	opts := FormatOptions{
		Dir:   dir,
		Fix:   false,
		Diff:  true,
		JSON:  true,
		Paths: []string{"src/", "tests/"},
	}
	_, args := buildPintCommand(opts)

	assert.Contains(t, args, "--test")
	assert.Contains(t, args, "--diff")
	assert.Contains(t, args, "--format=json")
	assert.Contains(t, args, "src/")
	assert.Contains(t, args, "tests/")
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	// Existing file
	f := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0644))
	assert.True(t, fileExists(f))

	// Non-existent file
	assert.False(t, fileExists(filepath.Join(dir, "nope.txt")))
}
