package main

import (
	"os"

	"github.com/rudrasantadip/ransumgo/cli"
)

func main() {
	defer os.Exit(0)

	cmd := cli.CommandLine{}
	cmd.Run()
}
