package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkFile creates a file (and parent directories) for testing.
func mkFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("stub"), 0o755))
}

// =============================================================================
// DetectAnalyser
// =============================================================================

func TestDetectAnalyser_Good_PHPStanConfig(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon"))

	typ, found := DetectAnalyser(dir)
	assert.True(t, found)
	assert.Equal(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_PHPStanDistConfig(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon.dist"))

	typ, found := DetectAnalyser(dir)
	assert.True(t, found)
	assert.Equal(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_PHPStanBinary(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpstan"))

	typ, found := DetectAnalyser(dir)
	assert.True(t, found)
	assert.Equal(t, AnalyserPHPStan, typ)
}

func TestDetectAnalyser_Good_Larastan(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "phpstan.neon"))
	mkFile(t, filepath.Join(dir, "vendor", "larastan", "larastan"))

	typ, found := DetectAnalyser(dir)
	assert.True(t, found)
	assert.Equal(t, AnalyserLarastan, typ)
}

func TestDetectAnalyser_Good_LarastanNunomaduro(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "phpstan"))
	mkFile(t, filepath.Join(dir, "vendor", "nunomaduro", "larastan"))

	typ, found := DetectAnalyser(dir)
	assert.True(t, found)
	assert.Equal(t, AnalyserLarastan, typ)
}

func TestDetectAnalyser_Bad_NoAnalyser(t *testing.T) {
	dir := t.TempDir()

	typ, found := DetectAnalyser(dir)
	assert.False(t, found)
	assert.Equal(t, AnalyserType(""), typ)
}

// =============================================================================
// DetectPsalm
// =============================================================================

func TestDetectPsalm_Good_PsalmConfig(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "psalm.xml"))

	typ, found := DetectPsalm(dir)
	assert.True(t, found)
	assert.Equal(t, PsalmStandard, typ)
}

func TestDetectPsalm_Good_PsalmDistConfig(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "psalm.xml.dist"))

	typ, found := DetectPsalm(dir)
	assert.True(t, found)
	assert.Equal(t, PsalmStandard, typ)
}

func TestDetectPsalm_Good_PsalmBinary(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "vendor", "bin", "psalm"))

	typ, found := DetectPsalm(dir)
	assert.True(t, found)
	assert.Equal(t, PsalmStandard, typ)
}

func TestDetectPsalm_Bad_NoPsalm(t *testing.T) {
	dir := t.TempDir()

	typ, found := DetectPsalm(dir)
	assert.False(t, found)
	assert.Equal(t, PsalmType(""), typ)
}

// =============================================================================
// buildPHPStanCommand
// =============================================================================

func TestBuildPHPStanCommand_Good_Defaults(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir}

	cmdName, args := buildPHPStanCommand(opts)
	assert.Equal(t, "phpstan", cmdName)
	assert.Equal(t, []string{"analyse"}, args)
}

func TestBuildPHPStanCommand_Good_VendorBinary(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin", "phpstan")
	mkFile(t, vendorBin)

	opts := AnalyseOptions{Dir: dir}
	cmdName, args := buildPHPStanCommand(opts)
	assert.Equal(t, vendorBin, cmdName)
	assert.Equal(t, []string{"analyse"}, args)
}

func TestBuildPHPStanCommand_Good_WithLevel(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Level: 5}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "--level")
	assert.Contains(t, args, "5")
}

func TestBuildPHPStanCommand_Good_WithMemory(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Memory: "2G"}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "--memory-limit")
	assert.Contains(t, args, "2G")
}

func TestBuildPHPStanCommand_Good_SARIF(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, SARIF: true}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "--error-format=sarif")
}

func TestBuildPHPStanCommand_Good_JSON(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, JSON: true}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "--error-format=json")
}

func TestBuildPHPStanCommand_Good_SARIFPrecedence(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, SARIF: true, JSON: true}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "--error-format=sarif")
	assert.NotContains(t, args, "--error-format=json")
}

func TestBuildPHPStanCommand_Good_WithPaths(t *testing.T) {
	dir := t.TempDir()
	opts := AnalyseOptions{Dir: dir, Paths: []string{"src", "app"}}

	_, args := buildPHPStanCommand(opts)
	assert.Contains(t, args, "src")
	assert.Contains(t, args, "app")
}

