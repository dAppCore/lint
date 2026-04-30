//go:build ignore

package sample

import . "dappco.re/go"

func TestA_Alpha_Good(t *T) {
	subject := Alpha
	if subject == nil {
		t.FailNow()
	}
	marker := "Alpha:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestA_Alpha_Bad(t *T) {
	subject := Alpha
	if subject == nil {
		t.FailNow()
	}
	marker := "Alpha:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestA_Alpha_Ugly(t *T) {
	subject := Alpha
	if subject == nil {
		t.FailNow()
	}
	marker := "Alpha:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
