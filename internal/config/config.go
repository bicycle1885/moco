package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds the application configuration
type Config struct {
	BaseDir     string `toml:"base_dir"`
	SummaryFile string `toml:"summary_file"`

	Run struct {
		Force         bool   `toml:"force"`
		CleanupOnFail bool   `toml:"cleanup_on_fail"`
		NoPushd       bool   `toml:"no_pushd"`
		StdoutFile    string `toml:"stdout_file"`
		StderrFile    string `toml:"stderr_file"`
		Silent        bool   `toml:"silent"`
		Message       string `toml:"message"`
		PromptMessage bool   `toml:"prompt_message"`
	} `toml:"run"`

	List struct {
		Format  string `toml:"format"`
		SortBy  string `toml:"sort_by"`
		Reverse bool   `toml:"reverse"`
		Branch  string `toml:"branch"`
		Status  string `toml:"status"`
		Since   string `toml:"since"`
		Command string `toml:"command"`
		Limit   int    `toml:"limit"`
	} `toml:"list"`

	Status struct {
		Level string `toml:"level"`
	} `toml:"status"`

	Config struct {
		Default bool `toml:"default"`
	} `toml:"config"`

	Archive struct {
		Format    string `toml:"format"`
		To        string `toml:"to"`
		OlderThan string `toml:"older_than"`
		Status    string `toml:"status"`
		Delete    bool   `toml:"delete"`
		DryRun    bool   `toml:"dry_run"`
	} `toml:"archive"`
}

// temprary struct for toml unmarshal to check if the value is nil
type config struct {
	BaseDir     *string `toml:"base_dir"`
	SummaryFile *string `toml:"summary_file"`

	Run *struct {
		Force         *bool   `toml:"force"`
		CleanupOnFail *bool   `toml:"cleanup_on_fail"`
		NoPushd       *bool   `toml:"no_pushd"`
		StdoutFile    *string `toml:"stdout_file"`
		StderrFile    *string `toml:"stderr_file"`
		Silent        *bool   `toml:"silent"`
		Message       *string `toml:"message"`
		PromptMessage *bool   `toml:"prompt_message"`
	} `toml:"run"`

	List *struct {
		Format  *string `toml:"format"`
		SortBy  *string `toml:"sort_by"`
		Reverse *bool   `toml:"reverse"`
		Branch  *string `toml:"branch"`
		Status  *string `toml:"status"`
		Since   *string `toml:"since"`
		Command *string `toml:"command"`
		Limit   *int    `toml:"limit"`
	} `toml:"list"`

	Status *struct {
		Level *string `toml:"level"`
	} `toml:"status"`

	Config *struct {
		Default *bool `toml:"default"`
	} `toml:"config"`

	Archive *struct {
		Format    *string `toml:"format"`
		To        *string `toml:"to"`
		OlderThan *string `toml:"older_than"`
		Status    *string `toml:"status"`
		Delete    *bool   `toml:"delete"`
		DryRun    *bool   `toml:"dry_run"`
	} `toml:"archive"`
}

const defaultConfig = `
# default configuration
base_dir = "runs"
summary_file = "summary.md"

[run]
force = false
cleanup_on_fail = false
no_pushd = false
stdout_file = "stdout.log"
stderr_file = "stderr.log"
silent = false
message = ""
prompt_message = false

[list]
format = "table"
sort_by = "date"
reverse = false
branch = ""
status = ""
since = ""
command = ""
limit = 0

[status]
level = "normal"

[config]
default = false

[archive]
format = "tar.gz"
to = "archives"
older_than = ""
status = ""
delete = false
dry_run = false
`

var globalConfig Config

func Main(config Config) error {
	b, _ := toml.Marshal(config)
	fmt.Print(string(b))
	return nil
}

// Init loads configuration from files
func Init() error {
	// Set defaults
	config, _ := loadData([]byte(defaultConfig))
	merge(&globalConfig, config)

	// Check for user-level config
	configDir, err := os.UserConfigDir()
	if err == nil {
		userConfig := filepath.Join(configDir, "moco", "config.toml")
		if _, err := os.Stat(userConfig); err == nil {
			config, err := loadFile(userConfig)
			if err != nil {
				return err
			}
			merge(&globalConfig, config)
		}
	}

	// Check for project-level config
	if _, err := os.Stat(".moco.toml"); err == nil {
		config, err := loadFile(".moco.toml")
		if err != nil {
			return err
		}
		merge(&globalConfig, config)
	}

	return nil
}

// Get returns the current configuration
func Get() Config {
	return globalConfig
}

func GetPointer() *Config {
	return &globalConfig
}

// GetDefault returns the default configuration
func GetDefault() Config {
	config, _ := loadData([]byte(defaultConfig))
	result := Config{}
	merge(&result, config)
	return result
}

// merge merges the src configuration into the dst configuration
func merge(dst *Config, src config) {
	if src.BaseDir != nil {
		dst.BaseDir = *src.BaseDir
	}
	if src.SummaryFile != nil {
		dst.SummaryFile = *src.SummaryFile
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
		if src.Run.StdoutFile != nil {
			dst.Run.StdoutFile = *src.Run.StdoutFile
		}
		if src.Run.StderrFile != nil {
			dst.Run.StderrFile = *src.Run.StderrFile
		}
		if src.Run.Silent != nil {
			dst.Run.Silent = *src.Run.Silent
		}
		if src.Run.Message != nil {
			dst.Run.Message = *src.Run.Message
		}
		if src.Run.PromptMessage != nil {
			dst.Run.PromptMessage = *src.Run.PromptMessage
		}
	}

	if src.List != nil {
		if src.List.Format != nil {
			dst.List.Format = *src.List.Format
		}
		if src.List.SortBy != nil {
			dst.List.SortBy = *src.List.SortBy
		}
		if src.List.Reverse != nil {
			dst.List.Reverse = *src.List.Reverse
		}
		if src.List.Branch != nil {
			dst.List.Branch = *src.List.Branch
		}
		if src.List.Status != nil {
			dst.List.Status = *src.List.Status
		}
		if src.List.Since != nil {
			dst.List.Since = *src.List.Since
		}
		if src.List.Command != nil {
			dst.List.Command = *src.List.Command
		}
		if src.List.Limit != nil {
			dst.List.Limit = *src.List.Limit
		}
	}

	if src.Status != nil {
		if src.Status.Level != nil {
			dst.Status.Level = *src.Status.Level
		}
	}

	if src.Config != nil {
		if src.Config.Default != nil {
			dst.Config.Default = *src.Config.Default
		}
	}

	if src.Archive != nil {
		if src.Archive.Format != nil {
			dst.Archive.Format = *src.Archive.Format
		}
		if src.Archive.To != nil {
			dst.Archive.To = *src.Archive.To
		}
		if src.Archive.OlderThan != nil {
			dst.Archive.OlderThan = *src.Archive.OlderThan
		}
		if src.Archive.Status != nil {
			dst.Archive.Status = *src.Archive.Status
		}
		if src.Archive.Delete != nil {
			dst.Archive.Delete = *src.Archive.Delete
		}
		if src.Archive.DryRun != nil {
			dst.Archive.DryRun = *src.Archive.DryRun
		}
	}
}

func loadFile(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	return loadData(data)
}

func loadData(data []byte) (config, error) {
	config := config{}
	err := toml.Unmarshal(data, &config)
	return config, err
}
