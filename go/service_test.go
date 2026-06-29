// SPDX-License-Identifier: EUPL-1.2

package lint

import (
	"testing"

	core "dappco.re/go"
)

// TestNewService_Defaults — root NewService re-exports pkg/lint.NewServiceFor
// per the canonical naming. Defaults should register cleanly under "lint".
func TestNewService_Defaults(t *testing.T) {
	c := core.New(core.WithService(NewService(ServiceOptions{})))
	if !c.Service("lint").OK {
		t.Fatal("lint not registered via root NewService alias")
	}
}

// TestRegister_Imperative — root Register re-exports pkg/lint.Register.
func TestRegister_Imperative(t *testing.T) {
	c := core.New(core.WithService(Register))
	if !c.Service("lint").OK {
		t.Fatal("lint not registered via root Register alias")
	}
}
