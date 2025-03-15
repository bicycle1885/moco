package main

import (
	"fmt"
	"os"

	"github.com/bicycle1885/moco/cmd"
	"github.com/bicycle1885/moco/internal/config"
)

func main() {
	// Initialize configuration
	if err := config.InitConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing configuration: %v\n", err)
		os.Exit(1)
	}

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
