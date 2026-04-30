// Package lint provides an embedded pattern catalog for code quality checks.
package lint

import (
	"embed"

	core "dappco.re/go"
	lintpkg "dappco.re/go/lint/pkg/lint"
)

//go:embed catalog/*.yaml
var catalogFS embed.FS

// LoadEmbeddedCatalog returns a Catalog loaded from the embedded YAML files.
func LoadEmbeddedCatalog() core.Result {
	catalog := lintpkg.LoadFS(catalogFS, "catalog")
	if !catalog.OK {
		return catalog
	}
	return core.Ok(catalog.Value)
}
