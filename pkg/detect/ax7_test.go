package detect

import (
	"os"
	"path/filepath"

	. "dappco.re/go"
)

func TestDetect_IsGoProject_Good(t *T) {
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test\n"), 0o644))
	got := IsGoProject(dir)
	AssertTrue(t, got)
	AssertEqual(t, []ProjectType{Go}, DetectAll(dir))
}

func TestDetect_IsGoProject_Bad(t *T) {
	dir := t.TempDir()
	got := IsGoProject(dir)
	AssertFalse(t, got)
	AssertEmpty(t, DetectAll(dir))
}

func TestDetect_IsGoProject_Ugly(t *T) {
	file := filepath.Join(t.TempDir(), "go.mod")
	RequireNoError(t, os.WriteFile(file, []byte("module example.test\n"), 0o644))
	got := IsGoProject(file)
	AssertTrue(t, got)
	AssertTrue(t, filepath.IsAbs(file))
}

func TestDetect_IsPHPProject_Good(t *T) {
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0o644))
	got := IsPHPProject(dir)
	AssertTrue(t, got)
	AssertEqual(t, []ProjectType{PHP}, DetectAll(dir))
}

func TestDetect_IsPHPProject_Bad(t *T) {
	dir := t.TempDir()
	got := IsPHPProject(dir)
	AssertFalse(t, got)
	AssertEmpty(t, DetectAll(dir))
}

func TestDetect_IsPHPProject_Ugly(t *T) {
	file := filepath.Join(t.TempDir(), "composer.json")
	RequireNoError(t, os.WriteFile(file, []byte(`{}`), 0o644))
	got := IsPHPProject(file)
	AssertTrue(t, got)
	AssertTrue(t, filepath.IsAbs(file))
}

func TestDetect_DetectAll_Good(t *T) {
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test\n"), 0o644))
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0o644))
	got := DetectAll(dir)
	AssertEqual(t, []ProjectType{Go, PHP}, got)
	AssertLen(t, got, 2)
}

func TestDetect_DetectAll_Bad(t *T) {
	dir := t.TempDir()
	got := DetectAll(dir)
	AssertEmpty(t, got)
	AssertFalse(t, IsGoProject(dir))
}

func TestDetect_DetectAll_Ugly(t *T) {
	dir := t.TempDir()
	RequireNoError(t, os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0o644))
	got := DetectAll(dir)
	AssertEqual(t, []ProjectType{PHP}, got)
	AssertFalse(t, IsGoProject(dir))
}
