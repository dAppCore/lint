package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

// mkFile creates a file (and parent directories) for testing.
func mkFile(t *T, path string) {
	t.Helper()
	RequireNoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	RequireNoError(t, os.WriteFile(path, []byte("stub"), 0o755))
}

// =============================================================================
// DetectAnalyser
// =============================================================================

func TestDetectAnalyser_Good_PHPStanConfig(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon"))

	typ, found := DetectAnalyser(dir)
	AssertTrue(t, found)
	AssertEqual(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_PHPStanDistConfig(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon.dist"))

	typ, found := DetectAnalyser(dir)
	AssertTrue(t, found)
	AssertEqual(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_PHPStanBinary(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpstan"))

	typ, found := DetectAnalyser(dir)
	AssertTrue(t, found)
	AssertEqual(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_Larastan(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon"))
	mkFile(t, filepath.Join(dir, "vendor", "larastan", "larastan"))

	typ, found := DetectAnalyser(dir)
	AssertTrue(t, found)
	AssertEqual(t, AnalyserLarastan, typ)
}

func TestDetectAnalyser_Good_LarastanNunomaduro(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpstan"))
	mkFile(t, filepath.Join(dir, "vendor", "nunomaduro", "larastan"))

	typ, found := DetectAnalyser(dir)
	AssertTrue(t, found)
	AssertEqual(t, AnalyserLarastan, typ)
}

func TestDetectAnalyser_Bad_NoAnalyser(t *T) {
	dir := t.TempDir()

	typ, found := DetectAnalyser(dir)
	AssertFalse(t, found)
	AssertEqual(t, AnalyserType(""), typ)
}

// =============================================================================
// DetectPsalm
// =============================================================================

func TestDetectPsalm_Good_PsalmConfig(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "psalm.xml"))

	typ, found := DetectPsalm(dir)
	AssertTrue(t, found)
	AssertEqual(t, PsalmStandard, typ)
}

func TestDetectPsalm_Good_PsalmDistConfig(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "psalm.xml.dist"))

	typ, found := DetectPsalm(dir)
	AssertTrue(t, found)
	AssertEqual(t, PsalmStandard, typ)
}

func TestDetectPsalm_Good_PsalmBinary(t *T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "psalm"))

	typ, found := DetectPsalm(dir)
	AssertTrue(t, found)
	AssertEqual(t, PsalmStandard, typ)
}

func TestDetectPsalm_Bad_NoPsalm(t *T) {
	dir := t.TempDir()

	typ, found := DetectPsalm(dir)
	AssertFalse(t, found)
	AssertEqual(t, PsalmType(""), typ)
}

// =============================================================================
// buildPHPStanCommand
// =============================================================================

func TestBuildPHPStanCommand_Good_Defaults(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir}

	cmdName, args := buildPHPStanCommand(opts)
	AssertEqual(t, "phpstan", cmdName)
	AssertEqual(t, []string{"analyse"}, args)
}

func TestBuildPHPStanCommand_Good_VendorBinary(t *T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "phpstan")
	mkFile(t, vendorBin)

	opts := AnalyseOptions{Dir: dir}
	cmdName, args := buildPHPStanCommand(opts)
	AssertEqual(t, vendorBin, cmdName)
	AssertEqual(t, []string{"analyse"}, args)
}

func TestBuildPHPStanCommand_Good_WithLevel(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Level: 5}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "--level")
	AssertContains(t, args, "5")
}

func TestBuildPHPStanCommand_Good_WithMemory(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Memory: "2G"}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "--memory-limit")
	AssertContains(t, args, "2G")
}

func TestBuildPHPStanCommand_Good_SARIF(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, SARIF: true}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "--error-format=sarif")
}

func TestBuildPHPStanCommand_Good_JSON(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, JSON: true}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "--error-format=json")
}

func TestBuildPHPStanCommand_Good_SARIFPrecedence(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, SARIF: true, JSON: true}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "--error-format=sarif")
	AssertNotContains(t, args, "--error-format=json")
}

func TestBuildPHPStanCommand_Good_WithPaths(t *T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Paths: []string{"src", "app"}}

	_, args := buildPHPStanCommand(opts)
	AssertContains(t, args, "src")
	AssertContains(t, args, "app")
}
