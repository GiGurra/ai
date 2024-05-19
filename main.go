package main

import (
	"context"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers/openai"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"strings"
)

type CliParams struct {
	Quiet       boa.Required[bool]    `descr:"Quiet mode, requires no user input" short:"q" default:"false"`
	Verbose     boa.Required[bool]    `descr:"Verbose output" short:"v" default:"false"`
	Provider    boa.Optional[string]  `descr:"AI provider to use" short:"p"`
	Model       boa.Optional[string]  `descr:"Model to use" short:"m"`
	Temperature boa.Optional[float64] `descr:"Temperature to use" short:"t" `
}

type StoredConfig struct {
	Provider string        `yaml:"provider"`
	OpenAI   openai.Config `yaml:"openai"`
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

			// if verbose is set, set slog to debug
			if p.Verbose.Value() {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			// check that file exists
			homeDir, err := os.UserHomeDir()
			if err != nil {
				panic(fmt.Errorf("failed to get home dir: %w", err))
			}

			slog.Debug(fmt.Sprintf("Will use provider: %s", p.Provider.Value()))
			slog.Debug("Will load config from " + homeDir + "/.config/gigurra/ai/config.yaml")

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
			configFilePath := homeDir + "/.config/gigurra/ai/config.yaml"
			cfg := StoredConfig{}
			yamlBytes, err := os.ReadFile(configFilePath)
			if err != nil {
				panic(fmt.Errorf("failed to read config file: %w", err))
			}

			err = yaml.Unmarshal(yamlBytes, &cfg)
			if err != nil {
				panic(fmt.Errorf("failed to unmarshal config file: %w", err))
			}

			if p.Provider.HasValue() {
				cfg.Provider = *p.Provider.Value()
			}

			var provider domain.Provider
			switch strings.TrimSpace(cfg.Provider) {
			case "":
				slog.Error("No provider found in config file: " + configFilePath)
				os.Exit(1)
			case "openai":
				// check that we have the required config
				if p.Temperature.HasValue() {
					cfg.OpenAI.Temperature = *p.Temperature.Value()
				}
				if p.Model.HasValue() {
					cfg.OpenAI.Model = *p.Model.Value()
				}
				if cfg.OpenAI.APIKey == "" {
					slog.Error("No openai api key found in config file: " + configFilePath)
					os.Exit(1)
				}
				provider = openai.NewOpenAIProvider(cfg.OpenAI)

				printAndListModels := func(level slog.Level) []string {
					slog.Log(context.Background(), level, "Available models:")

					models, err := provider.ListModels()
					if err != nil {
						slog.Error(fmt.Sprintf("Failed to list models: %v", err))
						os.Exit(1)
					}
					for _, model := range models {
						slog.Log(context.Background(), level, fmt.Sprintf(" - %s", model))
					}

					return models
				}

				if p.Verbose.Value() {
					models := printAndListModels(slog.LevelDebug)

					if !lo.Contains(models, cfg.OpenAI.Model) {
						slog.Error(fmt.Sprintf("Model '%s' not found. (Maybe you don't have access to it?)", cfg.OpenAI.Model))
						os.Exit(1)
					}
				}

			default:
				slog.Error(fmt.Sprintf("Unknown provider: %s", cfg.Provider))
				os.Exit(1)
			}

			slog.Info("Asking question")
			res, err := provider.BasicAsk(domain.Question{
				Messages: []domain.Message{
					{
						SourceType: domain.User,
						Content:    "please say hello",
					},
				},
			})
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to ask question: %v", err))
				os.Exit(1)
			}
			slog.Info("Got response")

			slog.Info(fmt.Sprintf("Response: %v", res))
		},
	}.ToApp()
}
