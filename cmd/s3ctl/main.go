// Package main provides the s3ctl executable entrypoint.
package main

import (
	"os"

	"github.com/netspeedy/s3ctl/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
