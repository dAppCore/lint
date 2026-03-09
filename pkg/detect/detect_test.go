package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGoProject_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	assert.True(t, IsGoProject(dir))
}

func TestIsGoProject_Bad(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, IsGoProject(dir))
}

func TestIsPHPProject_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
	assert.True(t, IsPHPProject(dir))
}

func TestIsPHPProject_Bad(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, IsPHPProject(dir))
}

func TestDetectAll_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
	types := DetectAll(dir)
	assert.Contains(t, types, Go)
	assert.Contains(t, types, PHP)
}

func TestDetectAll_Empty(t *testing.T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	assert.Empty(t, types)
}
