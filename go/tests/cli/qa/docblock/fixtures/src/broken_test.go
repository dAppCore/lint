//go:build ignore

package sample

import . "dappco.re/go"

func TestBroken_Broken_Good(t *T) {
	subject := Broken
	if subject == nil {
		t.FailNow()
	}
	marker := "Broken:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestBroken_Broken_Bad(t *T) {
	subject := Broken
	if subject == nil {
		t.FailNow()
	}
	marker := "Broken:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestBroken_Broken_Ugly(t *T) {
	subject := Broken
	if subject == nil {
		t.FailNow()
	}
	marker := "Broken:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
