//go:build ignore

package sample

import . "dappco.re/go"

func TestFixture_Run_Good(t *T) {
	AssertNotPanics(t, func() {
		Run()
	})
	AssertTrue(t, true)
}

func TestFixture_Run_Bad(t *T) {
	var called bool
	AssertNotPanics(t, func() {
		called = true
	})
	AssertTrue(t, called)
}

func TestFixture_Run_Ugly(t *T) {
	for i := 0; i < 2; i++ {
		Run()
	}
	AssertEqual(t, 2, 2)
}
