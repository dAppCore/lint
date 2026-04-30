package lint

func ExampleDetectLanguage() {
	_ = DetectLanguage
}

func ExampleNewScanner() {
	_ = NewScanner
}

func ExampleScanner_ScanDir() {
	_ = (*Scanner).ScanDir
}

func ExampleScanner_ScanFile() {
	_ = (*Scanner).ScanFile
}

func ExampleIsExcludedDir() {
	_ = IsExcludedDir
}
