// Command p30check scans a local repository for Stellar Protocol 30
// compatibility risks. See README.md for usage.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/Protocol-30/stellar-p30-check/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, os.Args[1:], cli.Env{Stdout: os.Stdout, Stderr: os.Stderr})
	stop()
	os.Exit(code)
}
