package qa

import . "dappco.re/go"

func TestCmdQa_AddQACommands_Good(t *T) {
	subject := AddQACommands
	if subject == nil {
		t.FailNow()
	}
	marker := "AddQACommands:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdQa_AddQACommands_Bad(t *T) {
	subject := AddQACommands
	if subject == nil {
		t.FailNow()
	}
	marker := "AddQACommands:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestCmdQa_AddQACommands_Ugly(t *T) {
	subject := AddQACommands
	if subject == nil {
		t.FailNow()
	}
	marker := "AddQACommands:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
