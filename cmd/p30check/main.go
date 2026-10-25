package main

import (
	"os"

	"github.com/Protocol-30/stellar-p30-check/internal/cli"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, cli.Options{Version: version}))
}
