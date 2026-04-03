package main

import (
	"forge.lthn.ai/core/cli/pkg/cli"
	_ "forge.lthn.ai/core/lint/cmd/qa"
)

func main() {
	cli.WithAppName("core")
	cli.Main()
}
