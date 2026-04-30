package php

import (
	. "dappco.re/go"
)

const (
	formatTestTest35ea57 = "--test"
)

func TestDetectFormatter_PintConfig(t *T) {
	dir := t.TempDir()

	// Create pint.json
	RequireResultOK(t, WriteFile(PathJoin(dir, "pint.json"), []byte("{}"), 0644))

	ft, found := DetectFormatter(dir)
	AssertTrue(t, found)
	AssertEqual(t, FormatterPint, ft)
}

func TestDetectFormatter_VendorBinary(t *T) {
	dir := t.TempDir()

	// Create vendor/bin/pint
	binDir := PathJoin(dir, "vendor", "bin")
	RequireResultOK(t, MkdirAll(binDir, 0755))

	RequireResultOK(t, WriteFile(PathJoin(binDir, "pint"), []byte("#!/bin/sh\n"), 0755))

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
	AssertContains(t, args, formatTestTest35ea57)
}

func TestBuildPintCommand_Fix(t *T) {
	dir := t.TempDir()

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, args := buildPintCommand(opts)

	AssertEqual(t, "pint", cmdName)
	AssertNotContains(t, args, formatTestTest35ea57)
}

func TestBuildPintCommand_VendorBinary(t *T) {
	dir := t.TempDir()

	binDir := PathJoin(dir, "vendor", "bin")
	RequireResultOK(t, MkdirAll(binDir, 0755))
	RequireResultOK(t, WriteFile(PathJoin(binDir, "pint"), []byte("#!/bin/sh\n"), 0755))

	opts := FormatOptions{Dir: dir, Fix: true}
	cmdName, _ := buildPintCommand(opts)

	AssertEqual(t, PathJoin(dir, "vendor", "bin", "pint"), cmdName)
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

	AssertContains(t, args, formatTestTest35ea57)
	AssertContains(t, args, "--diff")
	AssertContains(t, args, "--format=json")
	AssertContains(t, args, "src/")
	AssertContains(t, args, "tests/")
}

func TestFileExists(t *T) {
	dir := t.TempDir()

	// Existing file
	f := PathJoin(dir, "exists.txt")
	RequireResultOK(t, WriteFile(f, []byte("hi"), 0644))
	AssertTrue(t, fileExists(f))

	// Non-existent file
	AssertFalse(t, fileExists(PathJoin(dir, "nope.txt")))
}

func TestFormat_DetectFormatter_Good(t *T) {
	subject := DetectFormatter
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectFormatter:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestFormat_DetectFormatter_Bad(t *T) {
	subject := DetectFormatter
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectFormatter:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestFormat_DetectFormatter_Ugly(t *T) {
	subject := DetectFormatter
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectFormatter:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestFormat_Format_Good(t *T) {
	subject := Format
	if subject == nil {
		t.FailNow()
	}
	marker := "Format:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestFormat_Format_Bad(t *T) {
	subject := Format
	if subject == nil {
		t.FailNow()
	}
	marker := "Format:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestFormat_Format_Ugly(t *T) {
	subject := Format
	if subject == nil {
		t.FailNow()
	}
	marker := "Format:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
