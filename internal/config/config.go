// internal/config/config.go
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
		DefaultForce         bool `toml:"default_force"`
		DefaultCleanupOnFail bool `toml:"default_cleanup_on_fail"`
		DefaultNoPushd       bool `toml:"default_no_pushd"`
	} `toml:"run"`

	Git struct {
		RequireClean bool `toml:"require_clean"`
	} `toml:"git"`

	Archive struct {
		Format    string `toml:"format"`
		OlderThan string `toml:"older_than"`
	} `toml:"archive"`
}

var globalConfig Config

// InitConfig loads configuration from files and environment
func InitConfig() error {
	// Set defaults
	setDefaults()

	// Check for project-level config
	if _, err := os.Stat(".moco.toml"); err == nil {
		if err := loadConfigFile(".moco.toml"); err != nil {
			return err
		}
	}

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

	// Apply environment variable overrides
	applyEnvOverrides()

	return nil
}

// GetConfig returns the current configuration
func GetConfig() Config {
	return globalConfig
}

// setDefaults initializes the configuration with default values
func setDefaults() {
	// Paths
	globalConfig.Paths.BaseDir = "runs"
	globalConfig.Paths.SummaryFile = "summary.md"
	globalConfig.Paths.StdoutFile = "stdout.log"
	globalConfig.Paths.StderrFile = "stderr.log"

	// Run
	globalConfig.Run.DefaultForce = false
	globalConfig.Run.DefaultCleanupOnFail = false
	globalConfig.Run.DefaultNoPushd = false

	// Git
	globalConfig.Git.RequireClean = true

	// Archive
	globalConfig.Archive.Format = "tar.gz"
	globalConfig.Archive.OlderThan = "30d"
}

// loadConfigFile reads and parses a TOML configuration file
func loadConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return toml.Unmarshal(data, &globalConfig)
}

// applyEnvOverrides applies environment variable overrides to configuration
func applyEnvOverrides() {
	// Override base directory
	if dir := os.Getenv("MOCO_PATHS_BASE_DIR"); dir != "" {
		globalConfig.Paths.BaseDir = dir
	}

	// Other environment overrides would go here
}
