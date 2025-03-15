// cmd/root.go
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "moco",
	Short: "Moco - Research experiment manager",
	Long: `Moco is a tool for managing reproducible research experiments.

It ensures reproducibility by tracking git repository state, 
capturing command output, and documenting execution details.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
