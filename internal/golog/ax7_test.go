package log

import (
	stderrors "errors"

	core "dappco.re/go"
)

const (
	ax7TestRepoLoadb2f262 = "repo.load"
)

func TestLog_E_Good(t *core.T) {
	cause := stderrors.New("root")
	err := E(ax7TestRepoLoadb2f262, "failed", cause)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), ax7TestRepoLoadb2f262)
	core.AssertContains(t, err.Error(), "root")
}

func TestLog_E_Bad(t *core.T) {
	err := E("", "plain", nil)
	core.AssertError(t, err)
	core.AssertEqual(t, "plain", err.Error())
	core.AssertFalse(t, stderrors.Is(err, stderrors.New("plain")))
}

func TestLog_E_Ugly(t *core.T) {
	err := E("op", "", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "op")
	core.AssertEqual(t, "op: ", err.Error())
}

func TestLog_Wrap_Good(t *core.T) {
	cause := stderrors.New("root")
	err := Wrap(cause, ax7TestRepoLoadb2f262, "failed")
	core.AssertError(t, err)
	core.AssertTrue(t, stderrors.Is(err, cause))
	core.AssertContains(t, err.Error(), "failed")
}

func TestLog_Wrap_Bad(t *core.T) {
	err := Wrap(nil, ax7TestRepoLoadb2f262, "failed")
	core.AssertNil(t, err)
	core.AssertFalse(t, stderrors.Is(err, stderrors.New("missing")))
}

func TestLog_Wrap_Ugly(t *core.T) {
	cause := E("inner", "failed", nil)
	err := Wrap(cause, "", "outer")
	core.AssertError(t, err)
	core.AssertTrue(t, stderrors.Is(err, cause))
	core.AssertContains(t, err.Error(), "outer")
}

func TestLog_Err_Error_Good(t *core.T) {
	err := &Err{Operation: "op", Message: "failed", Cause: stderrors.New("root")}
	got := err.Error()
	core.AssertContains(t, got, "op")
	core.AssertContains(t, got, "root")
}

func TestLog_Err_Error_Bad(t *core.T) {
	err := &Err{Message: "failed"}
	got := err.Error()
	core.AssertEqual(t, "failed", got)
	core.AssertNotContains(t, got, ":")
}

func TestLog_Err_Error_Ugly(t *core.T) {
	err := &Err{}
	got := err.Error()
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, stderrors.Is(err, stderrors.New("")))
}

func TestLog_Err_Unwrap_Good(t *core.T) {
	cause := stderrors.New("root")
	err := &Err{Cause: cause}
	got := err.Unwrap()
	core.AssertEqual(t, cause, got)
	core.AssertTrue(t, stderrors.Is(err, cause))
}

func TestLog_Err_Unwrap_Bad(t *core.T) {
	err := &Err{Message: "failed"}
	got := err.Unwrap()
	core.AssertNil(t, got)
	core.AssertFalse(t, stderrors.Is(err, stderrors.New("failed")))
}

func TestLog_Err_Unwrap_Ugly(t *core.T) {
	cause := &Err{Message: "inner"}
	err := &Err{Message: "outer", Cause: cause}
	got := err.Unwrap()
	core.AssertEqual(t, cause, got)
	core.AssertContains(t, err.Error(), "inner")
}
