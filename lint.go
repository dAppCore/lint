// Package lint provides an embedded pattern catalog for code quality checks.
package lint

import (
	"embed"

	lintpkg "dappco.re/go/core/lint/pkg/lint"
)

//go:embed catalog/*.yaml
var catalogFS embed.FS

// LoadEmbeddedCatalog returns a Catalog loaded from the embedded YAML files.
func LoadEmbeddedCatalog() (*lintpkg.Catalog, error) {
	return lintpkg.LoadFS(catalogFS, "catalog")
}
