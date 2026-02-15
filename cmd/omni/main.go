package main

import (
	"os"

	"github.com/omni-co/omni-cli/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Execute(os.Args[1:], version))
}
