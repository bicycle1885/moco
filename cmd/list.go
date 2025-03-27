package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/list"
	"github.com/spf13/cobra"
)

func init() {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all experiments in the current project",
		Long: `List and filter experiments in the current project.

This command allows you to browse, search, and filter past experiments with
various criteria such as branch name, status, date, and command pattern.
Results can be sorted and formatted in different ways for easy analysis.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// List experiments
			return list.Main()
		},
	}

	// Add flags
	cfg := config.GetPointer()
	listCmd.Flags().StringVarP(&cfg.List.Format, "format", "f", "", "Output format (table, json, csv, plain)")
	listCmd.Flags().StringVarP(&cfg.List.SortBy, "sort", "s", "", "Sort by (date, branch, status, duration)")
	listCmd.Flags().BoolVarP(&cfg.List.Reverse, "reverse", "r", false, "Reverse sort order")
	listCmd.Flags().StringVarP(&cfg.List.Branch, "branch", "b", "", "Filter by branch name")
	listCmd.Flags().StringVar(&cfg.List.Status, "status", "", "Filter by status (success, failure, running)")
	listCmd.Flags().StringVar(&cfg.List.Since, "since", "", "Filter by date (e.g., '7d' for last 7 days)")
	listCmd.Flags().StringVarP(&cfg.List.Command, "command", "c", "", "Filter by command pattern (regex)")
	listCmd.Flags().IntVarP(&cfg.List.Limit, "limit", "n", 0, "Limit number of results (0 = no limit)")

	rootCmd.AddCommand(listCmd)
}
