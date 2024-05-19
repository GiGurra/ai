package main

import (
	"bufio"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers/openai_provider"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
	"os"
	"strings"
)

func main() {

	p := config.CliParams{}

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

			// Get the parent shell id
			sessionId := p.Session.Value()
			slog.Info(fmt.Sprintf("session id: %s", sessionId))

			state := session.LoadSession(sessionId)
			slog.Info(fmt.Sprintf("state: %+v", state))

			os.Exit(0)

			// if verbose is set, set slog to debug
			if p.Verbose.Value() {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, p)

			provider := createProvider(cfg)

			question := p.Question.Value()

			// if stdin is not empty, add it at the bottom of the first message
			stdInContents := readStdin()
			if stdInContents != "" {
				footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInContents)
				question = fmt.Sprintf("%s\n%s", question, footer)
			}

			stream := provider.BasicAskStream(domain.Question{
				Messages: []domain.Message{
					{
						SourceType: domain.User,
						Content:    question,
					},
				},
			})

			for {
				res, ok := <-stream
				if !ok {
					break // stream done
				}
				if res.Err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to receive stream response: %v", res.Err))
				}
				if len(res.Resp.GetChoices()) == 0 {
					continue
				}

				fmt.Printf("%s", res.Resp.GetChoices()[0].Message.Content)
			}
		},
	}.ToApp()
}

func readStdin() string {
	stdInContents := ""
	stat, err := os.Stdin.Stat()
	if err == nil && stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var sb strings.Builder
		for {
			input, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				common.FailAndExit(1, fmt.Sprintf("Failed to read from stdin: %v", err))
			}
			sb.WriteString(input)
			if err == io.EOF {
				break
			}
		}
		stdInContents = strings.TrimSpace(sb.String())
	}
	return stdInContents
}

func createProvider(cfg config.Config) domain.Provider {
	switch cfg.Provider {
	case "openai":
		return openai_provider.NewOpenAIProvider(cfg.OpenAI, cfg.Verbose)
	default:
		common.FailAndExit(1, fmt.Sprintf("Unsupported provider: %s", cfg.Provider))
		return nil // needed to compile :S
	}
}
