package config

import (
	"errors"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/providers/google_ai_studio_provider"
	"github.com/gigurra/ai/providers/google_cloud_provider"
	"github.com/gigurra/ai/providers/openai_provider"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"strings"
)

var CliParamEnricher = boa.ParamEnricherCombine(
	boa.ParamEnricherBool,
	boa.ParamEnricherName,
)

type CliParams struct {
	Question       boa.Required[[]string] `descr:"Question to ask" positional:"true"` // not used, but needed to produce help text
	Verbose        boa.Required[bool]     `descr:"Verbose output" default:"false" name:"verbose"`
	Session        boa.Optional[string]   `descr:"Session id (deprecated)" positional:"false" env:"CURRENT_AI_SESSION" name:"session"`
	Provider       boa.Optional[string]   `descr:"AI provider to use" name:"provider" env:"AI_PROVIDER" short:"p"`
	Model          boa.Optional[string]   `descr:"Model to use" name:"model"`
	Temperature    boa.Optional[float64]  `descr:"Temperature to use" name:"temperature"`
	ProviderApiKey boa.Optional[string]   `descr:"API key for provider" env:"PROVIDER_API_KEY"`
}

type CliSubcParams struct {
	Session boa.Optional[string] `descr:"Session id" positional:"true" env:"CURRENT_AI_SESSION" name:"session"`
	Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
}

func (c CliSubcParams) ToCliParams() CliParams {
	return CliParams{
		Session: c.Session,
		Verbose: c.Verbose,
	}
}

type StoredConfig struct {
	Provider       string                           `yaml:"provider"`
	OpenAI         openai_provider.Config           `yaml:"openai"`
	GoogleCloud    google_cloud_provider.Config     `yaml:"google_cloud"`
	GoogleAiStudio google_ai_studio_provider.Config `yaml:"google_ai_studio"`
}

type Config struct {
	StoredConfig
	Verbose bool
}

func (c Config) WithoutSecrets() Config {
	c.OpenAI.APIKey = "*****"
	c.GoogleAiStudio.APIKey = "*****"
	return c
}

func (c Config) ToYaml() string {
	yamlBytes, err := yaml.Marshal(c.StoredConfig)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to marshal config: %v", err))
	}
	return string(yamlBytes)
}

func CfgFilePath() string {
	appDir := common.AppDir()
	return appDir + "/config.yaml"
}

func LoadCfgFile() (string, Config) {

	filePath := CfgFilePath()

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
				bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
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

	if p.Verbose.HasValue() && p.Verbose.Value() {
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
	case "google-cloud":
		if p.Model.HasValue() {
			cfg.GoogleCloud.ModelId = *p.Model.Value()
		}
		if cfg.GoogleCloud.ProjectID == "" {
			common.FailAndExit(1, "No google cloud project id found in config file: "+configFilePath)
		}
		if cfg.GoogleCloud.LocationID == "" {
			common.FailAndExit(1, "No google cloud location id found in config file: "+configFilePath)
		}
		if cfg.GoogleCloud.ModelId == "" {
			common.FailAndExit(1, "No google cloud model id found in config file: "+configFilePath)
		}
	case "google-ai-studio":
		if p.Model.HasValue() {
			cfg.GoogleAiStudio.ModelId = *p.Model.Value()
		}
		if cfg.GoogleAiStudio.APIKey == "" {
			common.FailAndExit(1, "No google ai studio api_key found in config file: "+configFilePath)
		}
		if cfg.GoogleAiStudio.ModelId == "" {
			common.FailAndExit(1, "No google ai studio model id found in config file: "+configFilePath)
		}
	default:
		common.FailAndExit(1, fmt.Sprintf("Unsupported provider: %s", cfg.Provider))
	}

	return cfg
}
