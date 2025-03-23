package cmd

import (
	"github.com/bicycle1885/moco/internal/archive"
	"github.com/bicycle1885/moco/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	archiveCmd := &cobra.Command{
		Use:   "archive [run_directories...]",
		Short: "Archive and compress experiment directories",
		Long: `Archive and compress experiment directories to save disk space.

This command helps manage disk space by archiving older or completed
experiments into compressed archives (tar.gz or zip). Experiments can be
filtered by age, status, and other criteria before archiving.

You can specify one or more run directories to archive specific experiments,
or use the filtering options to archive experiments based on criteria.

An archive index is maintained for easy reference to archived experiments.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Archive experiments using config values
			return archive.Main(args)
		},
	}

	// Add flags
	cfg := config.GetPointer()
	archiveCmd.Flags().StringVarP(&cfg.Archive.OlderThan, "older-than", "o", "",
		"Archive experiments older than duration (e.g., '30d')")
	archiveCmd.Flags().StringVarP(&cfg.Archive.Status, "status", "s", "",
		"Archive by status (success, failure, running, all)")
	archiveCmd.Flags().StringVarP(&cfg.Archive.Format, "format", "f", "",
		"Archive format (zip, tar.gz)")
	archiveCmd.Flags().StringVarP(&cfg.Archive.To, "to", "t", "",
		"Archive destination directory")
	archiveCmd.Flags().BoolVar(&cfg.Archive.Delete, "delete", false,
		"Remove original directories after archiving")
	archiveCmd.Flags().BoolVar(&cfg.Archive.DryRun, "dry-run", false,
		"Show what would be archived without executing")

	rootCmd.AddCommand(archiveCmd)
}
