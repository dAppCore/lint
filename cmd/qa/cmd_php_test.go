package qa

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"forge.lthn.ai/core/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPHPStanJSONOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "phpstan"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"phpstan\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPStanFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPStanCommand(parent)
	command := findSubcommand(t, parent, "stan")
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"tool\":\"phpstan\",\"status\":\"ok\"}\n", output)
	assert.NotContains(t, output, "Static analysis passed")
	assert.NotContains(t, output, "PHP Static Analysis")
}

func TestPHPPsalmJSONOutput_DoesNotAppendSuccessBanner(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "composer.json"), "{}")
	writeExecutable(t, filepath.Join(dir, "vendor", "bin", "psalm"), "#!/bin/sh\nprintf '%s\\n' '{\"tool\":\"psalm\",\"status\":\"ok\"}'\n")

	restoreWorkingDir(t, dir)
	resetPHPPsalmFlags(t)

	parent := &cli.Command{Use: "qa"}
	addPHPPsalmCommand(parent)
	command := findSubcommand(t, parent, "psalm")
	require.NoError(t, command.Flags().Set("json", "true"))

	output := captureStdout(t, func() {
		require.NoError(t, command.RunE(command, nil))
	})

	assert.Equal(t, "{\"tool\":\"psalm\",\"status\":\"ok\"}\n", output)
	assert.NotContains(t, output, "Psalm analysis passed")
	assert.NotContains(t, output, "PHP Psalm Analysis")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func restoreWorkingDir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})
}

func resetPHPStanFlags(t *testing.T) {
	t.Helper()
	oldLevel := phpStanLevel
	oldMemory := phpStanMemory
	oldJSON := phpStanJSON
	oldSARIF := phpStanSARIF
	phpStanLevel = 0
	phpStanMemory = ""
	phpStanJSON = false
	phpStanSARIF = false
	t.Cleanup(func() {
		phpStanLevel = oldLevel
		phpStanMemory = oldMemory
		phpStanJSON = oldJSON
		phpStanSARIF = oldSARIF
	})
}

func resetPHPPsalmFlags(t *testing.T) {
	t.Helper()
	oldLevel := phpPsalmLevel
	oldFix := phpPsalmFix
	oldBaseline := phpPsalmBaseline
	oldShowInfo := phpPsalmShowInfo
	oldJSON := phpPsalmJSON
	oldSARIF := phpPsalmSARIF
	phpPsalmLevel = 0
	phpPsalmFix = false
	phpPsalmBaseline = false
	phpPsalmShowInfo = false
	phpPsalmJSON = false
	phpPsalmSARIF = false
	t.Cleanup(func() {
		phpPsalmLevel = oldLevel
		phpPsalmFix = oldFix
		phpPsalmBaseline = oldBaseline
		phpPsalmShowInfo = oldShowInfo
		phpPsalmJSON = oldJSON
		phpPsalmSARIF = oldSARIF
	})
}

func findSubcommand(t *testing.T, parent *cli.Command, name string) *cli.Command {
	t.Helper()
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()
	defer func() {
		require.NoError(t, reader.Close())
	}()

	fn()

	require.NoError(t, writer.Close())

	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(output)
}
