package main

import (
	"os"

	"github.com/geekjourneyx/findo/internal/cli"
)

var version = "1.2.1"

func main() {
	os.Exit(cli.Run(os.Args[1:], version, os.Stdout, os.Stderr))
}
