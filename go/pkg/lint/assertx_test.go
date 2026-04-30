package lint

import (
	core "dappco.re/go"
	"testing"
)

type M = testing.M

func RequireEqual(t core.TB, want, got any, msg ...string) {
	t.Helper()
	core.AssertEqual(t, want, got, msg...)
	if !core.DeepEqual(want, got) {
		t.FailNow()
	}
}

func RequireError(t core.TB, err error, msg ...string) {
	t.Helper()
	core.AssertError(t, err, msg...)
	if !requireErrorOK(err, msg) {
		t.FailNow()
	}
}

func RequireErrorIs(t core.TB, err, target error, msg ...string) {
	t.Helper()
	core.AssertErrorIs(t, err, target, msg...)
	if !core.Is(err, target) {
		t.FailNow()
	}
}

func RequireLen(t core.TB, v any, want int, msg ...string) {
	t.Helper()
	core.AssertLen(t, v, want, msg...)
	if !requireLenOK(v, want) {
		t.FailNow()
	}
}

func RequireNotNil(t core.TB, v any, msg ...string) {
	t.Helper()
	core.AssertNotNil(t, v, msg...)
	if !requireNotNilOK(v) {
		t.FailNow()
	}
}

func RequireResult[T any](t core.TB, result core.Result, msg ...string) T {
	t.Helper()
	core.AssertTrue(t, result.OK, append(msg, result.Error())...)
	if !result.OK {
		t.FailNow()
	}
	value, ok := result.Value.(T)
	core.AssertTrue(t, ok, "unexpected Result value type")
	if !ok {
		t.FailNow()
	}
	return value
}

func RequireResultOK(t core.TB, result core.Result, msg ...string) {
	t.Helper()
	core.AssertTrue(t, result.OK, append(msg, result.Error())...)
	if !result.OK {
		t.FailNow()
	}
}

func requireErrorOK(err error, msg []string) bool {
	if err == nil {
		return false
	}
	for _, want := range msg {
		if !core.Contains(err.Error(), want) {
			return false
		}
	}
	return true
}

func requireLenOK(v any, want int) bool {
	rv := core.ValueOf(v)
	switch rv.Kind() {
	case core.KindString, core.KindSlice, core.KindArray, core.KindMap, core.KindChan:
		return rv.Len() == want
	default:
		return false
	}
}

func requireNotNilOK(v any) bool {
	if v == nil {
		return false
	}
	rv := core.ValueOf(v)
	switch rv.Kind() {
	case core.KindChan, core.KindFunc, core.KindInterface, core.KindMap, core.KindPointer, core.KindSlice:
		return !rv.IsNil()
	default:
		return true
	}
}
