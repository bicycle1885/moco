package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds the application configuration
type Config struct {
	Paths struct {
		BaseDir     string `toml:"base_dir"`
		SummaryFile string `toml:"summary_file"`
		StdoutFile  string `toml:"stdout_file"`
		StderrFile  string `toml:"stderr_file"`
	} `toml:"paths"`

	Run struct {
		Force         bool `toml:"force"`
		CleanupOnFail bool `toml:"cleanup_on_fail"`
		NoPushd       bool `toml:"no_pushd"`
	} `toml:"run"`

	Archive struct {
		Format      string `toml:"format"`
		Destination string `toml:"destination"`
	} `toml:"archive"`
}

var globalConfig Config

const defaultConfig = `
[paths]
base_dir = "runs"
summary_file = "summary.md"
stdout_file = "stdout.log"
stderr_file = "stderr.log"

[run]
force = false
cleanup_on_fail = false
no_pushd = false

[archive]
format = "tar.gz"
older_than = "30d"
`

// InitConfig loads configuration from files
func InitConfig() error {
	// Set defaults
	loadConfigData([]byte(defaultConfig))

	// Check for user-level config
	configDir, err := os.UserConfigDir()
	if err == nil {
		userConfig := filepath.Join(configDir, "moco", "config.toml")
		if _, err := os.Stat(userConfig); err == nil {
			if err := loadConfigFile(userConfig); err != nil {
				return err
			}
		}
	}

	// Check for project-level config
	if _, err := os.Stat(".moco.toml"); err == nil {
		if err := loadConfigFile(".moco.toml"); err != nil {
			return err
		}
	}

	return nil
}

// GetConfig returns the current configuration
func GetConfig() Config {
	return globalConfig
}

// loadConfigFile reads a TOML file and parses it into the global configuration
func loadConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return loadConfigData(data)
}

// loadConfigData parses TOML data into the global configuration
func loadConfigData(data []byte) error {
	return toml.Unmarshal(data, &globalConfig)
}
