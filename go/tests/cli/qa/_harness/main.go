package main

import (
	"github.com/dappcore/cli/go/pkg/cli"
	_ "github.com/dappcore/lint/go/cmd/qa"
)

func main() {
	cli.WithAppName("core")
	cli.Main()
}
