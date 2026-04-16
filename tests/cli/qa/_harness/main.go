package main

import (
	"forge.lthn.ai/core/cli/pkg/cli"
	_ "dappco.re/go/core/lint/cmd/qa"
)

func main() {
	cli.WithAppName("core")
	cli.Main()
}
