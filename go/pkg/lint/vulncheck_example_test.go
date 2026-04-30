package lint

func ExampleToolkit_VulnCheck() {
	_ = (*Toolkit).VulnCheck
}

func ExampleParseVulnCheckJSON() {
	_ = ParseVulnCheckJSON
}
