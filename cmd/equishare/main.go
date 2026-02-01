package main

import (
	"os"

	"github.com/Rohianon/equishare-global-trading/cmd/equishare/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
