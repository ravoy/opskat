package main

import (
	"os"

	"github.com/octays/opskat/cmd/opsctl/command"
)

func main() {
	os.Exit(command.Execute())
}
