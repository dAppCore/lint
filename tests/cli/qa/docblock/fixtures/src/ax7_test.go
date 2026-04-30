//go:build ignore

package sample

import . "dappco.re/go"

func TestFixture_Alpha_Good(t *T) {
	AssertNotPanics(t, func() {
		Alpha()
	})
	AssertTrue(t, true)
}

func TestFixture_Alpha_Bad(t *T) {
	var called bool
	AssertNotPanics(t, func() {
		called = true
	})
	AssertTrue(t, called)
}

func TestFixture_Alpha_Ugly(t *T) {
	for i := 0; i < 2; i++ {
		Alpha()
	}
	AssertEqual(t, 2, 2)
}

func TestFixture_Beta_Good(t *T) {
	AssertNotPanics(t, func() {
		Beta()
	})
	AssertTrue(t, true)
}

func TestFixture_Beta_Bad(t *T) {
	var called bool
	AssertNotPanics(t, func() {
		Beta()
		called = true
	})
	AssertTrue(t, called)
}

func TestFixture_Beta_Ugly(t *T) {
	for i := 0; i < 2; i++ {
		Beta()
	}
	AssertEqual(t, 2, 2)
}

func TestFixture_Broken_Good(t *T) {
	AssertNotPanics(t, func() {
		Broken()
	})
	AssertTrue(t, true)
}

func TestFixture_Broken_Bad(t *T) {
	var called bool
	AssertNotPanics(t, func() {
		Broken()
		called = true
	})
	AssertTrue(t, called)
}

func TestFixture_Broken_Ugly(t *T) {
	for i := 0; i < 2; i++ {
		Broken()
	}
	AssertEqual(t, 2, 2)
}
