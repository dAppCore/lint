package main

import (
	"dappco.re/go/core/cli/pkg/cli"
	_ "dappco.re/go/lint/cmd/qa"
)

func main() {
	cli.WithAppName("core")
	cli.Main()
}
