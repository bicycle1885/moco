package cmd

import (
	"github.com/bicycle1885/moco/internal/experiment"
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
			// Get flag values
			format, _ := cmd.Flags().GetString("format")
			sortBy, _ := cmd.Flags().GetString("sort")
			reverse, _ := cmd.Flags().GetBool("reverse")
			branch, _ := cmd.Flags().GetString("branch")
			status, _ := cmd.Flags().GetString("status")
			since, _ := cmd.Flags().GetString("since")
			command, _ := cmd.Flags().GetString("command")
			limit, _ := cmd.Flags().GetInt("limit")

			// List experiments with provided options
			return experiment.List(experiment.ListOptions{
				Format:  format,
				SortBy:  sortBy,
				Reverse: reverse,
				Branch:  branch,
				Status:  status,
				Since:   since,
				Command: command,
				Limit:   limit,
			})
		},
	}

	// Add flags
	listCmd.Flags().StringP("format", "f", "table", "Output format (table, json, csv)")
	listCmd.Flags().StringP("sort", "s", "date", "Sort by (date, branch, status, duration)")
	listCmd.Flags().BoolP("reverse", "r", false, "Reverse sort order")
	listCmd.Flags().StringP("branch", "b", "", "Filter by branch name")
	listCmd.Flags().String("status", "", "Filter by status (success, failure, running)")
	listCmd.Flags().String("since", "", "Filter by date (e.g., '7d' for last 7 days)")
	listCmd.Flags().StringP("command", "c", "", "Filter by command pattern (regex)")
	listCmd.Flags().IntP("limit", "n", 0, "Limit number of results (0 = no limit)")

	rootCmd.AddCommand(listCmd)
}
