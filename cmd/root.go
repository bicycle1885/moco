package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "moco",
	Short: "Moco - Research experiment manager",
	Long: `Moco is a tool for managing reproducible research experiments.

It ensures reproducibility by tracking git repository state, 
capturing command output, and documenting execution details.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cfg := config.GetPointer()
	rootCmd.PersistentFlags().StringVarP(&cfg.BaseDir, "base-dir", "d", "",
		"Base directory for experiment output")
}
