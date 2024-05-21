package config

import (
	"errors"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/providers/openai_provider"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

type CliParams struct {
	Question       boa.Required[[]string] `descr:"Question to ask" positional:"true"` // not used, but needed to produce help text
	Quiet          boa.Required[bool]     `descr:"Quiet mode, requires no user input" default:"false"`
	Verbose        boa.Required[bool]     `descr:"Verbose output" default:"false"`
	Session        boa.Optional[string]   `descr:"Session id (deprecated)" positional:"false" env:"CURRENT_AI_SESSION"`
	Provider       boa.Optional[string]   `descr:"AI provider to use"`
	Model          boa.Optional[string]   `descr:"Model to use"`
	Temperature    boa.Optional[float64]  `descr:"Temperature to use"`
	ProviderApiKey boa.Optional[string]   `descr:"API key for provider" env:"PROVIDER_API_KEY"`
}

type CliStatusParams struct {
	Session boa.Optional[string] `descr:"Session id (deprecated)" positional:"false" env:"CURRENT_AI_SESSION"`
	Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false"`
}

type CliSetSession struct {
	Session boa.Required[string] `descr:"Session id" positional:"true"`
	Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false"`
}

type CliDeleteSession struct {
	Session boa.Optional[string] `descr:"Session id" pos:"true"`
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
			fmt.Printf("No config file found, will create one at: %s\n", filePath)
			fmt.Printf("Do you wish to enter an OpenAI API key now? (y/n) ")

			var input string
			_, err := fmt.Scanln(&input)
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("Failed to read input: %v", err))
			}

			openaiApiKey := ""
			if strings.HasPrefix(strings.ToLower(input), "y") {
				fmt.Printf("Please enter your OpenAI API key (first time only): ")
				bytePassword, err := terminal.ReadPassword(syscall.Stdin)
				if err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to read input: %v", err))
				}
				openaiApiKey = string(bytePassword)
				fmt.Printf("*****\n")
			}

			yamlBytes, err := yaml.Marshal(StoredConfig{
				Provider: "openai",
				OpenAI: openai_provider.Config{
					APIKey:      openaiApiKey,
					Model:       "gpt-4o",
					Temperature: 0.1,
				},
			})
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("failed to marshal default config: %v", err))
			}
			err = os.WriteFile(filePath, yamlBytes, 0644)
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("failed to write default config file: %v", err))
			}
		} else {
			common.FailAndExit(1, fmt.Sprintf("failed to stat config file: %v", err))
		}
	}

	cfg := StoredConfig{}
	yamlBytes, err := os.ReadFile(filePath)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to read config file: %s: %v", filePath, err))
	}

	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to unmarshal config file: %s: %v", filePath, err))
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
		if p.ProviderApiKey.HasValue() {
			cfg.OpenAI.APIKey = *p.ProviderApiKey.Value()
		}
		if cfg.OpenAI.APIKey == "" {
			common.FailAndExit(1, "No openai api key found in config file: "+configFilePath)
		}
	default:
		common.FailAndExit(1, fmt.Sprintf("Unsupported provider: %s", cfg.Provider))
	}

	return cfg
}
