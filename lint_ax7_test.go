package lint

import . "dappco.re/go"

func TestLint_LoadEmbeddedCatalog_Good(t *T) {
	catalog, err := LoadEmbeddedCatalog()
	AssertNoError(t, err)
	AssertNotNil(t, catalog)
	AssertTrue(t, len(catalog.Rules) > 0)
}

func TestLint_LoadEmbeddedCatalog_Bad(t *T) {
	catalog, err := LoadEmbeddedCatalog()
	AssertNoError(t, err)
	AssertNil(t, catalog.ByID("missing-rule"))
	AssertEmpty(t, catalog.ForLanguage("nope"))
}

func TestLint_LoadEmbeddedCatalog_Ugly(t *T) {
	catalog, err := LoadEmbeddedCatalog()
	AssertNoError(t, err)
	AssertNotNil(t, catalog.ByID("go-cor-003"))
	AssertTrue(t, len(catalog.AtSeverity("critical")) >= 0)
}
