//go:build ignore

package sample

import . "dappco.re/go"

func TestInput_Run_Good(t *T) {
	subject := Run
	if subject == nil {
		t.FailNow()
	}
	marker := "Run:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestInput_Run_Bad(t *T) {
	subject := Run
	if subject == nil {
		t.FailNow()
	}
	marker := "Run:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestInput_Run_Ugly(t *T) {
	subject := Run
	if subject == nil {
		t.FailNow()
	}
	marker := "Run:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
