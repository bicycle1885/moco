package cmd

import (
	"github.com/bicycle1885/moco/internal/archive"
	"github.com/spf13/cobra"
)

func init() {
	archiveCmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive and compress experiment directories",
		Long: `Archive and compress experiment directories to save disk space.

This command helps manage disk space by archiving older or completed
experiments into compressed archives (tar.gz or zip). Experiments can be
filtered by age, status, and other criteria before archiving.

An archive index is maintained for easy reference to archived experiments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values
			olderThan, _ := cmd.Flags().GetString("older-than")
			status, _ := cmd.Flags().GetString("status")
			format, _ := cmd.Flags().GetString("format")
			destination, _ := cmd.Flags().GetString("destination")
			delete, _ := cmd.Flags().GetBool("delete")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Archive experiments with provided options
			return archive.Run(archive.ArchiveOptions{
				OlderThan:   olderThan,
				Status:      status,
				Format:      format,
				Destination: destination,
				Delete:      delete,
				DryRun:      dryRun,
			})
		},
	}

	// Add flags
	archiveCmd.Flags().StringP("older-than", "o", "", "Archive experiments older than duration (e.g., '30d')")
	archiveCmd.Flags().StringP("status", "s", "", "Archive by status (success, failure, running, all)")
	archiveCmd.Flags().StringP("format", "f", "", "Archive format (zip, tar.gz)")
	archiveCmd.Flags().StringP("destination", "d", "archives", "Archive destination directory")
	archiveCmd.Flags().Bool("delete", false, "Remove original directories after archiving")
	archiveCmd.Flags().Bool("dry-run", false, "Show what would be archived without executing")

	rootCmd.AddCommand(archiveCmd)
}
