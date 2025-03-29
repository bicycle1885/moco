package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/run"
	"github.com/spf13/cobra"
)

func init() {
	runCmd := &cobra.Command{
		Use:     "run [command]",
		Aliases: []string{"r"},
		Short:   "Run a command in an experiment directory with metadata tracking",
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
			// Execute the command with experiment tracking
			return run.Main(args)
		},
	}

	// Add flags
	cfg := config.GetPointer()
	runCmd.Flags().BoolVarP(&cfg.Run.Force, "force", "f", false,
		"Allow experiments to run with uncommitted changes")
	runCmd.Flags().BoolVarP(&cfg.Run.NoPushd, "no-pushd", "n", false,
		"Execute command in current directory (don't cd to experiment dir)")
	runCmd.Flags().BoolVarP(&cfg.Run.CleanupOnFail, "cleanup-on-fail", "c", false,
		"Remove experiment directory if command fails")
	runCmd.Flags().BoolVarP(&cfg.Run.Silent, "silent", "s", false,
		"Suppress command output to stdout/stderr (write only to log files)")
	runCmd.Flags().StringVarP(&cfg.Run.Message, "message", "m", "",
		"Get user input for experiment message")
	runCmd.Flags().BoolVarP(&cfg.Run.PromptMessage, "prompt-message", "p", false,
		"Prompt for user input for experiment message")

	rootCmd.AddCommand(runCmd)
}
