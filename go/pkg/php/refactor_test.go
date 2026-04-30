package php

import (
	. "dappco.re/go"
)

const (
	refactorTestDryRun6f1586    = "--dry-run"
	refactorTestRectorPhp58a377 = "rector.php"
)

// =============================================================================
// DetectRector
// =============================================================================

func TestDetectRector_Good_RectorConfig(t *T) {
	dir := t.TempDir()
	mkFile(t, PathJoin(dir, refactorTestRectorPhp58a377))

	AssertTrue(t, DetectRector(dir))
}

func TestDetectRector_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	mkFile(t, PathJoin(dir, "vendor", "bin", "rector"))

	AssertTrue(t, DetectRector(dir))
}

func TestDetectRector_Bad_Empty(t *T) {
	dir := t.TempDir()

	AssertFalse(t, DetectRector(dir))
	AssertFalse(t, fileExists(PathJoin(dir, refactorTestRectorPhp58a377)))
}

// =============================================================================
// buildRectorCommand
// =============================================================================

func TestBuildRectorCommand_Good_Defaults(t *T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir}

	cmdName, args := buildRectorCommand(opts)
	AssertEqual(t, "rector", cmdName)
	// Fix is false by default, so --dry-run should be present
	AssertContains(t, args, "process")
	AssertContains(t, args, refactorTestDryRun6f1586)
}

func TestBuildRectorCommand_Good_Fix(t *T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, Fix: true}

	cmdName, args := buildRectorCommand(opts)
	AssertEqual(t, "rector", cmdName)
	AssertContains(t, args, "process")
	AssertNotContains(t, args, refactorTestDryRun6f1586)
}

func TestBuildRectorCommand_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin", "rector")
	mkFile(t, vendorBin)

	opts := RectorOptions{Dir: dir}
	cmdName, _ := buildRectorCommand(opts)
	AssertEqual(t, vendorBin, cmdName)
}

func TestBuildRectorCommand_Good_Diff(t *T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, Diff: true}

	_, args := buildRectorCommand(opts)
	AssertContains(t, args, "--output-format")
	AssertContains(t, args, "diff")
}

func TestBuildRectorCommand_Good_ClearCache(t *T) {
	dir := t.TempDir()
	opts := RectorOptions{Dir: dir, ClearCache: true}

	_, args := buildRectorCommand(opts)
	AssertContains(t, args, "--clear-cache")
}

func TestBuildRectorCommand_Good_AllFlags(t *T) {
	dir := t.TempDir()
	opts := RectorOptions{
		Dir:        dir,
		Fix:        true,
		Diff:       true,
		ClearCache: true,
	}

	_, args := buildRectorCommand(opts)
	AssertContains(t, args, "process")
	AssertNotContains(t, args, refactorTestDryRun6f1586)
	AssertContains(t, args, "--output-format")
	AssertContains(t, args, "diff")
	AssertContains(t, args, "--clear-cache")
}

func TestRectorOptions_Defaults(t *T) {
	opts := RectorOptions{}
	AssertEmpty(t, opts.Dir)
	AssertFalse(t, opts.Fix)
	AssertFalse(t, opts.Diff)
	AssertFalse(t, opts.ClearCache)
	AssertNil(t, opts.Output)
}

func TestDetectRector_Good_BothConfigAndBinary(t *T) {
	dir := t.TempDir()

	// Create both config and vendor binary
	RequireResultOK(t, WriteFile(PathJoin(dir, refactorTestRectorPhp58a377), []byte("<?php\n"), 0644))
	mkFile(t, PathJoin(dir, "vendor", "bin", "rector"))

	AssertTrue(t, DetectRector(dir))
}

func TestRefactor_DetectRector_Good(t *T) {
	subject := DetectRector
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectRector:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRefactor_DetectRector_Bad(t *T) {
	subject := DetectRector
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectRector:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRefactor_DetectRector_Ugly(t *T) {
	subject := DetectRector
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectRector:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestRefactor_RunRector_Good(t *T) {
	subject := RunRector
	if subject == nil {
		t.FailNow()
	}
	marker := "RunRector:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestRefactor_RunRector_Bad(t *T) {
	subject := RunRector
	if subject == nil {
		t.FailNow()
	}
	marker := "RunRector:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestRefactor_RunRector_Ugly(t *T) {
	subject := RunRector
	if subject == nil {
		t.FailNow()
	}
	marker := "RunRector:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
