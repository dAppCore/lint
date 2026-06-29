package gocmd

import (
	"testing"

	core "dappco.re/go"
)

// AddGoCommands registers the full `core go` surface without error (every
// c.Command call succeeds, so go / go/qa / go/qa/full / go/test are wired).
func TestGocmd_AddGoCommands_Good(t *testing.T) {
	if r := AddGoCommands(core.New()); !r.OK {
		t.Fatalf("AddGoCommands: %v", r.Error())
	}
}

// indentLines prefixes every line so multi-line failure detail stays readable.
func TestGocmd_indentLines_Good(t *testing.T) {
	if got := indentLines("a\nb", "  "); got != "  a\n  b" {
		t.Fatalf("indentLines = %q, want %q", got, "  a\n  b")
	}
}

// The outcome constructors carry the right status and detail.
func TestGocmd_stepOutcome_Good(t *testing.T) {
	if okOutcome().status != "ok" {
		t.Fatalf("okOutcome status = %q", okOutcome().status)
	}
	if o := failOutcome("boom"); o.status != "fail" || o.detail != "boom" {
		t.Fatalf("failOutcome = %+v", o)
	}
	if o := skipOutcome("absent"); o.status != "skip" || o.detail != "absent" {
		t.Fatalf("skipOutcome = %+v", o)
	}
}
