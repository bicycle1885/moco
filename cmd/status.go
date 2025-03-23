package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/status"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Show the current status of the project",
		Long: `Show the current status of the project including:

- Git repository status (branch, changes)
- Currently running experiments
- Recent experiment history
- Project statistics (success/failure rate, disk usage)

The level of detail and output format can be customized.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show project status
			return status.Show()
		},
	}

	// Add flags
	cfg := config.GetPointer()
	statusCmd.Flags().StringVarP(&cfg.Status.Level, "level", "l", "normal", "Level of detail (minimal, normal, full)")

	rootCmd.AddCommand(statusCmd)
}
