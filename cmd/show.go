package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/show"
	"github.com/spf13/cobra"
)

func init() {
	showCmd := &cobra.Command{
		Use:     "show [run]",
		Aliases: []string{"sh"},
		Short:   "Show a run's summary file",
		Long: `Show displays the summary file of a specified run.
  
The summary file is rendered as markdown by default and displayed in a pager.
You can specify either a directory containing the summary file or the summary file itself.
  
If a directory is provided, it will look for the summary file as defined in your configuration.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return show.Main(args[0])
		},
	}

	cfg := config.GetPointer()
	showCmd.Flags().BoolVarP(&cfg.Show.Raw, "raw", "r", false,
		"Show raw summary without rendering")

	rootCmd.AddCommand(showCmd)
}
