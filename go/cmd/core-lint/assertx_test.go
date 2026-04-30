package main

import . "dappco.re/go"

func RequireEqual(t TB, want, got any, msg ...string) {
	t.Helper()
	AssertEqual(t, want, got, msg...)
	if !DeepEqual(want, got) {
		t.FailNow()
	}
}

func RequireLen(t TB, v any, want int, msg ...string) {
	t.Helper()
	AssertLen(t, v, want, msg...)
	if !requireLenOK(v, want) {
		t.FailNow()
	}
}

func RequireResultOK(t TB, result Result, msg ...string) {
	t.Helper()
	AssertTrue(t, result.OK, append(msg, result.Error())...)
	if !result.OK {
		t.FailNow()
	}
}

func requireLenOK(v any, want int) bool {
	rv := ValueOf(v)
	switch rv.Kind() {
	case KindString, KindSlice, KindArray, KindMap, KindChan:
		return rv.Len() == want
	default:
		return false
	}
}
