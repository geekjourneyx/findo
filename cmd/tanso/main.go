package main

import (
	"os"

	"github.com/geekjourneyx/tanso/internal/cli"
)

var version = "2.0.2"

func main() {
	os.Exit(cli.Run(os.Args[1:], version, os.Stdout, os.Stderr))
}
