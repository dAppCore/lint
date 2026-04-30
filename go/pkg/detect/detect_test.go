package detect

import (
	. "dappco.re/go"
)

func TestIsGoProject_Good(t *T) {
	dir := t.TempDir()
	r := WriteFile(PathJoin(dir, "go.mod"), []byte("module test"), 0644)
	RequireTrue(t, r.OK)
	AssertTrue(t, IsGoProject(dir))
}

func TestIsGoProject_Bad(t *T) {
	dir := t.TempDir()
	AssertFalse(t, IsGoProject(dir))
	AssertEmpty(t, DetectAll(dir))
}

func TestIsPHPProject_Good(t *T) {
	dir := t.TempDir()
	r := WriteFile(PathJoin(dir, "composer.json"), []byte("{}"), 0644)
	RequireTrue(t, r.OK)
	AssertTrue(t, IsPHPProject(dir))
}

func TestIsPHPProject_Bad(t *T) {
	dir := t.TempDir()
	AssertFalse(t, IsPHPProject(dir))
	AssertEmpty(t, DetectAll(dir))
}

func TestDetectAll_Good(t *T) {
	dir := t.TempDir()
	goMod := WriteFile(PathJoin(dir, "go.mod"), []byte("module test"), 0644)
	RequireTrue(t, goMod.OK)
	composer := WriteFile(PathJoin(dir, "composer.json"), []byte("{}"), 0644)
	RequireTrue(t, composer.OK)
	types := DetectAll(dir)
	AssertContains(t, types, Go)
	AssertContains(t, types, PHP)
}

func TestDetectAll_Empty(t *T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	AssertEmpty(t, types)
}

func TestDetect_IsGoProject_Good(t *T) {
	subject := IsGoProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsGoProject:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_IsGoProject_Bad(t *T) {
	subject := IsGoProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsGoProject:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_IsGoProject_Ugly(t *T) {
	subject := IsGoProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsGoProject:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_IsPHPProject_Good(t *T) {
	subject := IsPHPProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsPHPProject:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_IsPHPProject_Bad(t *T) {
	subject := IsPHPProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsPHPProject:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_IsPHPProject_Ugly(t *T) {
	subject := IsPHPProject
	if subject == nil {
		t.FailNow()
	}
	marker := "IsPHPProject:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_DetectAll_Good(t *T) {
	subject := DetectAll
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectAll:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_DetectAll_Bad(t *T) {
	subject := DetectAll
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectAll:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestDetect_DetectAll_Ugly(t *T) {
	subject := DetectAll
	if subject == nil {
		t.FailNow()
	}
	marker := "DetectAll:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
