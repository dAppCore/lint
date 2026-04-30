package php

func ExampleNewQARunner() {
	_ = NewQARunner
}

func ExampleQARunner_BuildSpecs() {
	_ = (*QARunner).BuildSpecs
}

func ExampleQACheckRunResult_GetIssueMessage() {
	_ = (*QACheckRunResult).GetIssueMessage
}
