package lint

func ExampleNewToolkit() {
	_ = NewToolkit
}

func ExampleToolkit_Run() {
	_ = (*Toolkit).Run
}

func ExampleToolkit_FindTrackedComments() {
	_ = (*Toolkit).FindTrackedComments
}

func ExampleToolkit_FindTODOs() {
	_ = (*Toolkit).FindTODOs
}

func ExampleToolkit_AuditDeps() {
	_ = (*Toolkit).AuditDeps
}

func ExampleToolkit_DiffStat() {
	_ = (*Toolkit).DiffStat
}

func ExampleToolkit_UncommittedFiles() {
	_ = (*Toolkit).UncommittedFiles
}

func ExampleToolkit_Lint() {
	_ = (*Toolkit).Lint
}

func ExampleToolkit_ScanSecrets() {
	_ = (*Toolkit).ScanSecrets
}

func ExampleToolkit_ModTidy() {
	_ = (*Toolkit).ModTidy
}

func ExampleToolkit_Build() {
	_ = (*Toolkit).Build
}

func ExampleToolkit_TestCount() {
	_ = (*Toolkit).TestCount
}

func ExampleToolkit_Coverage() {
	_ = (*Toolkit).Coverage
}

func ExampleToolkit_RaceDetect() {
	_ = (*Toolkit).RaceDetect
}

func ExampleToolkit_GocycloComplexity() {
	_ = (*Toolkit).GocycloComplexity
}

func ExampleToolkit_DepGraph() {
	_ = (*Toolkit).DepGraph
}

func ExampleToolkit_GitLog() {
	_ = (*Toolkit).GitLog
}

func ExampleToolkit_CheckPerms() {
	_ = (*Toolkit).CheckPerms
}
