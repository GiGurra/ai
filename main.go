package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/providers/openai"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

type CliParams struct {
	Quiet    boa.Required[bool]   `descr:"Quiet mode, requires no user input" short:"q" default:"false"`
	Provider boa.Required[string] `descr:"AI provider to use" short:"p" default:"openai"`
	Model    boa.Required[string] `descr:"Model to use" short:"m" default:"gpt-4o"`
}

type StoredConfig struct {
	OpenAI openai.OpenAIConfig `yaml:"openai"`
}

var defaultConfig = StoredConfig{}

func main() {

	p := CliParams{}

	boa.Wrap{
		Use:   "ai",
		Short: "ai cli tool, you know, for building stuff",
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherBool,
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &p,
		Long:   `See the README.MD for more information`,
		Run: func(cmd *cobra.Command, args []string) {

			// check that file exists
			homeDir, err := os.UserHomeDir()
			if err != nil {
				panic(fmt.Errorf("failed to get home dir: %w", err))
			}

			slog.Info(fmt.Sprintf("Will use provider: %s", p.Provider.Value()))
			slog.Info("Will load config from " + homeDir + "/.config/gigurra/ai/config.yaml")

			_, err = os.Stat(homeDir + "/.config/gigurra/ai/config.yaml")
			if err != nil {
				if os.IsNotExist(err) {
					slog.Info("No config file found, will create one")
					// create file
					yamlBytes, err := yaml.Marshal(defaultConfig)
					if err != nil {
						panic(fmt.Errorf("failed to marshal default config: %w", err))
					}
					err = os.MkdirAll(homeDir+"/.config/gigurra/ai", 0755)
					if err != nil {
						panic(fmt.Errorf("failed to create config dir: %w", err))
					}
					err = os.WriteFile(homeDir+"/.config/gigurra/ai/config.yaml", yamlBytes, 0644)
					if err != nil {
						panic(fmt.Errorf("failed to write default config file: %w", err))
					}
				} else {
					panic(fmt.Errorf("failed to stat config file: %w", err))
				}
			}

			// load config
			cfg := StoredConfig{}
			yamlBytes, err := os.ReadFile(homeDir + "/.config/gigurra/ai/config.yaml")
			if err != nil {
				panic(fmt.Errorf("failed to read config file: %w", err))
			}

			err = yaml.Unmarshal(yamlBytes, &cfg)
			if err != nil {
				panic(fmt.Errorf("failed to unmarshal config file: %w", err))
			}

			switch p.Provider.Value() {
			case "openai":
				// check that we have the required config
				if cfg.OpenAI.APIKey == "" {
					slog.Error("No openai api key found in config file")
					os.Exit(1)
				}
				models, err := openai.ListModels(cfg.OpenAI)
				if err != nil {
					slog.Error(fmt.Sprintf("Failed to list models: %v", err))
					os.Exit(1)
				}
				slog.Info(fmt.Sprintf("Found %d models", len(models.Data)))
				for _, model := range models.Data {
					slog.Info(fmt.Sprintf(" - %s", model.ID))
				}
			default:
				slog.Error(fmt.Sprintf("Unknown provider: %s", p.Provider.Value()))
			}

			// load config from <home>/.config/gigurra/ai/config.yaml
		},
	}.ToApp()
}
