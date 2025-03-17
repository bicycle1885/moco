package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/experiment"
	"github.com/spf13/cobra"
)

func init() {
	runCmd := &cobra.Command{
		Use:          "run [command]",
		Aliases:      []string{"r"},
		SilenceUsage: true,
		Short:        "Run a command in an experiment directory with metadata tracking",
		Long: `Run a command with full reproducibility tracking.

This command will:
1. Check the git repository status
2. Create a unique experiment directory
3. Record git metadata and system information
4. Execute the specified command
5. Capture stdout and stderr
6. Generate a comprehensive summary

Each experiment is stored in a directory with a timestamp, branch name,
and git commit hash to ensure traceability.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --dev flag combination
			if devMode, _ := cmd.Flags().GetBool("dev"); devMode {
				_ = cmd.Flags().Set("force", "true")
				_ = cmd.Flags().Set("cleanup-on-fail", "true")
			}

			// Get flag values
			force, _ := cmd.Flags().GetBool("force")
			baseDir, _ := cmd.Flags().GetString("dir")
			noPushd, _ := cmd.Flags().GetBool("no-pushd")
			cleanupOnFail, _ := cmd.Flags().GetBool("cleanup-on-fail")

			// Execute the command with experiment tracking
			return experiment.Run(experiment.RunOptions{
				Command:       args,
				Force:         force,
				BaseDir:       baseDir,
				NoPushd:       noPushd,
				CleanupOnFail: cleanupOnFail,
			})
		},
	}

	// Add flags with defaults from config
	cfg := config.GetConfig()
	runCmd.Flags().BoolP("force", "f", cfg.Run.DefaultForce,
		"Allow experiments to run with uncommitted changes")
	runCmd.Flags().StringP("dir", "d", cfg.Paths.BaseDir,
		"Base directory for experiment output")
	runCmd.Flags().BoolP("no-pushd", "n", cfg.Run.DefaultNoPushd,
		"Execute command in current directory (don't cd to experiment dir)")
	runCmd.Flags().BoolP("cleanup-on-fail", "c", cfg.Run.DefaultCleanupOnFail,
		"Remove experiment directory if command fails")
	runCmd.Flags().Bool("dev", false,
		"Development mode: enables both --force and --cleanup-on-fail")

	rootCmd.AddCommand(runCmd)
}
