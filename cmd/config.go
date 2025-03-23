package cmd

import (
	"github.com/bicycle1885/moco/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	configCmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"co"},
		Short:   "Show configuration settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()
			if cfg.Config.Default {
				cfg = config.GetDefault()
			}
			config.Show(cfg)
			return nil
		},
	}

	cfg := config.GetPointer()
	configCmd.Flags().BoolVarP(&cfg.Config.Default, "default", "", false, "Show the default configuration")
	rootCmd.AddCommand(configCmd)
}
