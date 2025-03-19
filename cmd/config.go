package cmd

import (
	"fmt"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

func init() {
	configCmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"co"},
		Short:   "Show configuration settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			default_, _ := cmd.Flags().GetBool("default")
			var cfg config.Config
			if default_ {
				cfg = config.GetDefaultConfig()
			} else {
				cfg = config.GetConfig()
			}
			b, _ := toml.Marshal(cfg)
			fmt.Print(string(b))
			return nil
		},
	}

	configCmd.Flags().BoolP("default", "d", false, "Show the default configuration")
	rootCmd.AddCommand(configCmd)
}
