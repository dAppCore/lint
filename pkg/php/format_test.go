package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

func TestDetectFormatter_PintConfig(t *T) {
	dir := t.TempDir()

	// Create pint.json
	err := os.WriteFile(filepath.Join(dir, "pint.json"), []byte("{}"), 0644)
	RequireNoError(t, err)

	ft, found := DetectFormatter(dir)
	AssertTrue(t, found)
	AssertEqual(t, FormatterPint, ft)
}

func TestDetectFormatter_VendorBinary(t *T) {
	dir := t.TempDir()

	// Create vendor/bin/pint
	binDir := filepath.Join(dir, "vendor", "bin")
	err := os.MkdirAll(binDir, 0755)
	RequireNoError(t, err)

	err = os.WriteFile(filepath.Join(binDir, "pint"), []byte("#!/bin/sh\n"), 0755)
	RequireNoError(t, err)

	ft, found := DetectFormatter(dir)
	AssertTrue(t, found)
	AssertEqual(t, FormatterPint, ft)
}

func TestDetectFormatter_Empty(t *T) {
	dir := t.TempDir()

	ft, found := DetectFormatter(dir)
	AssertFalse(t, found)
	AssertEqual(t, FormatterType(""), ft)
}

func TestBuildPintCommand_Defaults(t *T) {
	dir := t.TempDir()

	opts := FormatOptions{Dir: dir}
	cmdName, args := buildPintCommand(opts)

	// No vendor binary, so fallback to bare "pint"
	AssertEqual(t, "pint", cmdName)
	// Fix is false by default, so --test should be present
	AssertContains(t, args, "--test")
}

func TestBuildPintCommand_Fix(t *T) {
	dir := t.TempDir()

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, args := buildPintCommand(opts)

	AssertEqual(t, "pint", cmdName)
	AssertNotContains(t, args, "--test")
}

func TestBuildPintCommand_VendorBinary(t *T) {
	dir := t.TempDir()

	binDir := filepath.Join(dir, "vendor", "bin")
	RequireNoError(t, os.MkdirAll(binDir, 0755))
	RequireNoError(t, os.WriteFile(filepath.Join(binDir, "pint"), []byte("#!/bin/sh\n"), 0755))

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, _ := buildPintCommand(opts)

	AssertEqual(t, filepath.Join(dir, "vendor", "bin", "pint"), cmdName)
}

func TestBuildPintCommand_AllFlags(t *T) {
	dir := t.TempDir()

	opts := FormatOptions{
		Dir:   dir,
		Fix:   false,
		Diff:  true,
		JSON:  true,
		Paths: []string{"src/", "tests/"},
	}
	_, args := buildPintCommand(opts)

	AssertContains(t, args, "--test")
	AssertContains(t, args, "--diff")
	AssertContains(t, args, "--format=json")
	AssertContains(t, args, "src/")
	AssertContains(t, args, "tests/")
}

func TestFileExists(t *T) {
	dir := t.TempDir()

	// Existing file
	f := filepath.Join(dir, "exists.txt")
	RequireNoError(t, os.WriteFile(f, []byte("hi"), 0644))
	AssertTrue(t, fileExists(f))

	// Non-existent file
	AssertFalse(t, fileExists(filepath.Join(dir, "nope.txt")))
}
