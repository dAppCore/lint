// SPDX-License-Identifier: EUPL-1.2

// Service registration for the root lint package — exposes the canonical
// `NewService(opts)` + `Register(c)` shape per Mantis #1336 by
// re-exporting the pkg/lint canonical entries from the root import path.
//
// The full Service implementation (Service struct, LintConfigServiceOptions,
// adapters, RunInput, ToolInfo, Report, etc.) lives in
// dappco.re/go/lint/pkg/lint and is already canonical-shape. This file
// is the root-package surface so consumers calling `lint.NewService(...)`
// from `import "dappco.re/go/lint"` get the canon without having to
// reach into the pkg/lint subpackage.
//
//	c, _ := core.New(
//	    core.WithService(lint.NewService(lint.ServiceOptions{})),
//	)
//	svc := core.MustServiceFor[*lintpkg.Service](c, "lint")
//
// Note: Service struct is exposed as `lintpkg.Service` (the pkg/lint
// type) — root re-exports the constructors but the Service type itself
// stays in pkg/lint to avoid duplicating the surface.

package lint

import (
	core "dappco.re/go"
	lintpkg "dappco.re/go/lint/pkg/lint"
)

// ServiceOptions is the canonical-shape configuration alias for the root lint
// package, re-exporting pkg/lint.LintConfigServiceOptions so consumers can
// type `lint.ServiceOptions{...}` from the root import path.
type ServiceOptions = lintpkg.LintConfigOptions

// NewService returns a factory that wires the lint service into a Core
// with the supplied options. Use through core.WithService:
//
//	core.WithService(lint.NewService(lint.ServiceOptions{}))
//
// Re-exports pkg/lint.NewServiceFor under the canonical name. The
// resulting *Service registers under "lint" and exposes Run, Tools,
// and the rest of the lint API.
func NewService(opts ServiceOptions) func(*core.Core) core.Result {
	return lintpkg.NewServiceFor(opts)
}

// Register wires the lint service into the Core with default options —
// the imperative-style alternative to NewService.
//
//	c := core.New()
//	if r := lint.Register(c); !r.OK { return r }
//
// Re-exports pkg/lint.Register under the canonical root surface.
func Register(c *core.Core) core.Result {
	return lintpkg.Register(c)
}
