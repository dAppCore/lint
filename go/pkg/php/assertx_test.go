package php

import . "dappco.re/go"

func RequireError(t TB, err error, msg ...string) {
	t.Helper()
	AssertError(t, err, msg...)
	if !requireErrorOK(err, msg) {
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

func RequireResult[T any](t TB, result Result, msg ...string) T {
	t.Helper()
	AssertTrue(t, result.OK, append(msg, result.Error())...)
	if !result.OK {
		t.FailNow()
	}
	value, ok := result.Value.(T)
	AssertTrue(t, ok, "unexpected Result value type")
	if !ok {
		t.FailNow()
	}
	return value
}

func RequireResultOK(t TB, result Result, msg ...string) {
	t.Helper()
	AssertTrue(t, result.OK, append(msg, result.Error())...)
	if !result.OK {
		t.FailNow()
	}
}

func requireErrorOK(err error, msg []string) bool {
	if err == nil {
		return false
	}
	for _, want := range msg {
		if !Contains(err.Error(), want) {
			return false
		}
	}
	return true
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
