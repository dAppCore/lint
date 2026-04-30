package lint

import (
	. "dappco.re/go"
	lintpkg "dappco.re/go/lint/pkg/lint"
)

func TestLint_LoadEmbeddedCatalog_Good(t *T) {
	result := LoadEmbeddedCatalog()
	AssertTrue(t, result.OK, result.Error())
	catalog := result.Value.(*lintpkg.Catalog)
	AssertNotNil(t, catalog)
	AssertTrue(t, len(catalog.Rules) > 0)
}

func TestLint_LoadEmbeddedCatalog_Bad(t *T) {
	result := LoadEmbeddedCatalog()
	AssertTrue(t, result.OK, result.Error())
	catalog := result.Value.(*lintpkg.Catalog)
	AssertNil(t, catalog.ByID("missing-rule"))
	AssertEmpty(t, catalog.ForLanguage("nope"))
}

func TestLint_LoadEmbeddedCatalog_Ugly(t *T) {
	result := LoadEmbeddedCatalog()
	AssertTrue(t, result.OK, result.Error())
	catalog := result.Value.(*lintpkg.Catalog)
	AssertNotNil(t, catalog.ByID("go-cor-003"))
	AssertFalse(t, len(catalog.AtSeverity("critical")) < 0)
}
