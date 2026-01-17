package main

import (
	"os"

	"github.com/alexandraswan/gcli/cmd/gcli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
