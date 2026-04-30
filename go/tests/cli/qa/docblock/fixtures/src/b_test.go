//go:build ignore

package sample

import . "dappco.re/go"

func TestB_Beta_Good(t *T) {
	subject := Beta
	if subject == nil {
		t.FailNow()
	}
	marker := "Beta:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestB_Beta_Bad(t *T) {
	subject := Beta
	if subject == nil {
		t.FailNow()
	}
	marker := "Beta:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestB_Beta_Ugly(t *T) {
	subject := Beta
	if subject == nil {
		t.FailNow()
	}
	marker := "Beta:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
