package detect

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
)

func TestIsGoProject_Good(t *T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	AssertTrue(t, IsGoProject(dir))
}

func TestIsGoProject_Bad(t *T) {
	dir := t.TempDir()
	AssertFalse(t, IsGoProject(dir))
}

func TestIsPHPProject_Good(t *T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
	AssertTrue(t, IsPHPProject(dir))
}

func TestIsPHPProject_Bad(t *T) {
	dir := t.TempDir()
	AssertFalse(t, IsPHPProject(dir))
}

func TestDetectAll_Good(t *T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
	types := DetectAll(dir)
	AssertContains(t, types, Go)
	AssertContains(t, types, PHP)
}

func TestDetectAll_Empty(t *T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	AssertEmpty(t, types)
}
