package config

import (
	"errors"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/providers/openai_provider"
	"gopkg.in/yaml.v3"
	"io/fs"
	"log/slog"
	"os"
	"strings"
)

type CliParams struct {
	Question    boa.Required[[]string] `descr:"Question to ask" positional:"true"` // not used, but needed to produce help text
	Quiet       boa.Required[bool]     `descr:"Quiet mode, requires no user input" default:"false"`
	Verbose     boa.Required[bool]     `descr:"Verbose output" default:"false"`
	Session     boa.Optional[string]   `descr:"Session id (deprecated)" positional:"false" env:"CURRENT_AI_SESSION"`
	Provider    boa.Optional[string]   `descr:"AI provider to use"`
	Model       boa.Optional[string]   `descr:"Model to use"`
	Temperature boa.Optional[float64]  `descr:"Temperature to use"`
}

type CliStatusParams struct {
	Session boa.Optional[string] `descr:"Session id (deprecated)" positional:"false" env:"CURRENT_AI_SESSION"`
	Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false"`
}

type CliSetSession struct {
	Session boa.Required[string] `descr:"Session id" positional:"true"`
	Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false"`
}

func (c CliStatusParams) ToCliParams() CliParams {
	return CliParams{
		Session: c.Session,
		Verbose: c.Verbose,
	}
}

type StoredConfig struct {
	Provider string                 `yaml:"provider"`
	OpenAI   openai_provider.Config `yaml:"openai"`
}

type Config struct {
	StoredConfig
	Verbose bool
}

func (c Config) WithoutSecrets() Config {
	c.OpenAI.APIKey = "*****"
	return c
}

func (c Config) ToYaml() string {
	yamlBytes, err := yaml.Marshal(c.StoredConfig)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to marshal config: %v", err))
	}
	return string(yamlBytes)
}

func cfgFilePath() string {
	appDir := common.AppDir()
	return appDir + "/config.yaml"
}

func LoadCfgFile() (string, Config) {

	filePath := cfgFilePath()

	_, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Info("No config file found, will create one")
			yamlBytes, err := yaml.Marshal(StoredConfig{
				Provider: "openai",
				OpenAI: openai_provider.Config{
					Model:       "gpt-4o",
					Temperature: 0.1,
				},
			})
			if err != nil {
				panic(fmt.Errorf("failed to marshal default config: %w", err))
			}
			err = os.WriteFile(filePath, yamlBytes, 0644)
			if err != nil {
				panic(fmt.Errorf("failed to write default config file: %w", err))
			}
		} else {
			panic(fmt.Errorf("failed to stat config file: %w", err))
		}
	}

	cfg := StoredConfig{}
	yamlBytes, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to read config file: %s: %w", filePath, err))
	}

	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config file: %s: %w", filePath, err))
	}

	return filePath, Config{
		StoredConfig: cfg,
	}
}

func ValidateCfg(
	configFilePath string,
	cfg Config,
	p CliParams,
) Config {

	if p.Provider.HasValue() {
		cfg.Provider = *p.Provider.Value()
	}

	if p.Verbose.Value() {
		cfg.Verbose = true
	}

	switch strings.TrimSpace(cfg.Provider) {
	case "":
		common.FailAndExit(1, "No provider found in config file: "+configFilePath)
	case "openai":
		if p.Temperature.HasValue() {
			cfg.OpenAI.Temperature = *p.Temperature.Value()
		}
		if p.Model.HasValue() {
			cfg.OpenAI.Model = *p.Model.Value()
		}
		if cfg.OpenAI.APIKey == "" {
			common.FailAndExit(1, "No openai api key found in config file: "+configFilePath)
		}
	default:
		common.FailAndExit(1, fmt.Sprintf("Unsupported provider: %s", cfg.Provider))
	}

	return cfg
}
