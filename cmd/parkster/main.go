package main

import (
	"os"

	"github.com/petter-b/parkster-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
