package lint

func ExampleLoadDir() {
	_ = LoadDir
}

func ExampleLoadFS() {
	_ = LoadFS
}

func ExampleCatalog_ForLanguage() {
	_ = (*Catalog).ForLanguage
}

func ExampleCatalog_AtSeverity() {
	_ = (*Catalog).AtSeverity
}

func ExampleCatalog_ByID() {
	_ = (*Catalog).ByID
}
