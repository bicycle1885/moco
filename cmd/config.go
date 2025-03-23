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
			cfg := config.GetConfig()
			if cfg.Config.Default {
				cfg = config.GetDefaultConfig()
			}
			config.ShowConfig(cfg)
			return nil
		},
	}

	cfg := config.GetConfigPointer()
	configCmd.Flags().BoolVarP(&cfg.Config.Default, "default", "", false, "Show the default configuration")
	rootCmd.AddCommand(configCmd)
}
