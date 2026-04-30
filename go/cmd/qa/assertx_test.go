package qa

import . "dappco.re/go"

func RequireNotEqual(t TB, want, got any, msg ...string) {
	t.Helper()
	AssertNotEqual(t, want, got, msg...)
	if DeepEqual(want, got) {
		t.FailNow()
	}
}

func RequireError(t TB, err error, msg ...string) {
	t.Helper()
	AssertError(t, err, msg...)
	if !requireErrorOK(err, msg) {
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

func RequireResultError(t TB, result Result, msg ...string) {
	t.Helper()
	AssertFalse(t, result.OK, append(msg, result.Error())...)
	if result.OK {
		t.FailNow()
	}
	for _, want := range msg {
		if !Contains(result.Error(), want) {
			t.FailNow()
		}
	}
}

func RequireLen(t TB, v any, want int, msg ...string) {
	t.Helper()
	AssertLen(t, v, want, msg...)
	if !requireLenOK(v, want) {
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

func testStringIndex(haystack string, needle string) int {
	if needle == "" {
		return 0
	}
	max := len(haystack) - len(needle)
	for i := 0; i <= max; i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
