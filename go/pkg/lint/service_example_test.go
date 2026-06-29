package lint

func ExampleNewService() {
	_ = NewService
}

func ExampleService_Run() {
	_ = (*Service).Run
}

func ExampleService_Tools() {
	_ = (*Service).Tools
}

func ExampleService_WriteDefaultConfig() {
	_ = (*Service).WriteDefaultConfig
}

func ExampleService_InstallHook() {
	_ = (*Service).InstallHook
}

func ExampleService_RemoveHook() {
	_ = (*Service).RemoveHook
}

func ExampleNewServiceFor() {
	_ = NewServiceFor
}

func ExampleRegister() {
	_ = Register
}

func ExampleService_OnStartup() {
	_ = (*Service).OnStartup
}

func ExampleService_OnShutdown() {
	_ = (*Service).OnShutdown
}
