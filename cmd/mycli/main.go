package main

import (
	"os"

	"github.com/yourorg/mycli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
