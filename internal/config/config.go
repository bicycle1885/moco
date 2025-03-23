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

// temprary struct for toml unmarshal to check if the value is nil
type config struct {
	Paths *struct {
		BaseDir     *string `toml:"base_dir"`
		SummaryFile *string `toml:"summary_file"`
		StdoutFile  *string `toml:"stdout_file"`
		StderrFile  *string `toml:"stderr_file"`
	} `toml:"paths"`

	Run *struct {
		Force         *bool `toml:"force"`
		CleanupOnFail *bool `toml:"cleanup_on_fail"`
		NoPushd       *bool `toml:"no_pushd"`
	} `toml:"run"`

	Archive *struct {
		Format      *string `toml:"format"`
		Destination *string `toml:"destination"`
	} `toml:"archive"`
}

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
destination = "archives"
`

var globalConfig Config

// InitConfig loads configuration from files
func InitConfig() error {
	// Set defaults
	config, _ := loadConfigData([]byte(defaultConfig))
	mergeConfig(&globalConfig, config)

	// Check for user-level config
	configDir, err := os.UserConfigDir()
	if err == nil {
		userConfig := filepath.Join(configDir, "moco", "config.toml")
		if _, err := os.Stat(userConfig); err == nil {
			config, err := loadConfigFile(userConfig)
			if err != nil {
				return err
			}
			mergeConfig(&globalConfig, config)
		}
	}

	// Check for project-level config
	if _, err := os.Stat(".moco.toml"); err == nil {
		config, err := loadConfigFile(".moco.toml")
		if err != nil {
			return err
		}
		mergeConfig(&globalConfig, config)
	}

	return nil
}

// GetConfig returns the current configuration
func GetConfig() Config {
	return globalConfig
}

func GetConfigPointer() *Config {
	return &globalConfig
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() Config {
	config, _ := loadConfigData([]byte(defaultConfig))
	result := Config{}
	mergeConfig(&result, config)
	return result
}

// mergeConfig merges the src configuration into the dst configuration
func mergeConfig(dst *Config, src config) {
	if src.Paths != nil {
		if src.Paths.BaseDir != nil {
			dst.Paths.BaseDir = *src.Paths.BaseDir
		}
		if src.Paths.SummaryFile != nil {
			dst.Paths.SummaryFile = *src.Paths.SummaryFile
		}
		if src.Paths.StdoutFile != nil {
			dst.Paths.StdoutFile = *src.Paths.StdoutFile
		}
		if src.Paths.StderrFile != nil {
			dst.Paths.StderrFile = *src.Paths.StderrFile
		}
	}

	if src.Run != nil {
		if src.Run.Force != nil {
			dst.Run.Force = *src.Run.Force
		}
		if src.Run.CleanupOnFail != nil {
			dst.Run.CleanupOnFail = *src.Run.CleanupOnFail
		}
		if src.Run.NoPushd != nil {
			dst.Run.NoPushd = *src.Run.NoPushd
		}
	}

	if src.Archive != nil {
		if src.Archive.Format != nil {
			dst.Archive.Format = *src.Archive.Format
		}
		if src.Archive.Destination != nil {
			dst.Archive.Destination = *src.Archive.Destination
		}
	}
}

func loadConfigFile(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	return loadConfigData(data)
}

func loadConfigData(data []byte) (config, error) {
	config := config{}
	err := toml.Unmarshal(data, &config)
	return config, err
}
