package cmd

import (
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
			// Get flag values
			detail, _ := cmd.Flags().GetString("detail")
			format, _ := cmd.Flags().GetString("format")

			// Show project status
			return status.Show(status.StatusOptions{
				DetailLevel: detail,
				Format:      format,
			})
		},
	}

	// Add flags
	statusCmd.Flags().StringP("detail", "d", "normal", "Level of detail (minimal, normal, full)")
	statusCmd.Flags().StringP("format", "f", "text", "Output format (text, json, markdown)")

	rootCmd.AddCommand(statusCmd)
}
