package lint

func ExampleNewCoverageStore() {
	_ = NewCoverageStore
}

func ExampleCoverageStore_Append() {
	_ = (*CoverageStore).Append
}

func ExampleCoverageStore_Load() {
	_ = (*CoverageStore).Load
}

func ExampleCoverageStore_Latest() {
	_ = (*CoverageStore).Latest
}

func ExampleParseCoverProfile() {
	_ = ParseCoverProfile
}

func ExampleParseCoverOutput() {
	_ = ParseCoverOutput
}

func ExampleCompareCoverage() {
	_ = CompareCoverage
}
