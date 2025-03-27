package main

import (
	"github.com/bicycle1885/moco/cmd"
	"github.com/bicycle1885/moco/internal/config"
	"github.com/charmbracelet/log"
)

func main() {
	if err := config.Init(); err != nil {
		log.Fatalf("Failed to intialize configuration: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}
