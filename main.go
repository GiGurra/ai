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
	pStatus := config.CliStatusParams{}

	boa.Wrap{
		Use:   "ai",
		Short: "ai cli tool, you know, for building stuff",
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherBool,
			boa.ParamEnricherName,
			//boa.ParamEnricherShort, // conflicts with varargs positional args
		),
		Params: &p,
		Long:   `See the README.MD for more information`,
		Args:   cobra.MinimumNArgs(1),
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:   "sessions",
				Short: "List all stored sessions",
				Run: func(cmd *cobra.Command, args []string) {
					sessions := session.ListSessions()
					for _, s := range sessions {
						fmt.Printf(" - %s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
					}
				},
			}.ToCmd(),
			boa.Wrap{
				Use:    "status",
				Short:  "Prints info about current session",
				Params: &pStatus,
				Run: func(cmd *cobra.Command, args []string) {
					s := session.LoadSession(pStatus.Session.Value())
					fmt.Printf("%s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
				},
			}.ToCmd(),
			boa.Wrap{
				Use:    "config",
				Short:  "Prints the current configuration",
				Params: &pStatus,
				Run: func(cmd *cobra.Command, args []string) {
					cfgFilePath, storedCfg := config.LoadCfgFile()
					cfg := config.ValidateCfg(cfgFilePath, storedCfg, p)
					cfg = cfg.WithoutSecrets()
					fmt.Printf("--- %s ---\n%s", cfgFilePath, cfg.ToYaml())
				},
			}.ToCmd(),
		},
		Run: func(cmd *cobra.Command, args []string) {

			sb := strings.Builder{}
			for i, arg := range args {
				if i > 0 {
					sb.WriteString(" ")
				}
				sb.WriteString(arg)
			}
			question := sb.String()

			// if verbose is set, set slog to debug
			if p.Verbose.Value() {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, p)

			provider := createProvider(cfg)

			// if stdin is not empty, add it at the bottom of the first message
			stdInContents := readStdin()
			if stdInContents != "" {
				footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInContents)
				question = fmt.Sprintf("%s\n%s", question, footer)
			}

			newMessage := domain.Message{
				SourceType: domain.User,
				Content:    question,
			}

			state := session.LoadSession(p.Session.Value())

			messageHistory := func() []domain.Message {
				var messages []domain.Message
				for _, entry := range state.History {
					if entry.Type == "message" {
						messages = append(messages, entry.Message)
					}
				}
				return messages
			}()

			stream := provider.BasicAskStream(domain.Question{
				Messages: append(messageHistory, newMessage),
			})

			state.AddMessage(newMessage)

			accum := ""
			for {
				res, ok := <-stream
				if !ok {
					break // stream done
				}
				if res.Err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to receive stream response: %v", res.Err))
				}

				state.InputTokensAccum += res.Resp.GetUsage().PromptTokens
				state.OutputTokensAccum += res.Resp.GetUsage().CompletionTokens
				state.InputTokens = res.Resp.GetUsage().PromptTokens
				state.OutputTokens = res.Resp.GetUsage().CompletionTokens

				if len(res.Resp.GetChoices()) == 0 {
					continue
				}

				accum += res.Resp.GetChoices()[0].Message.Content
				fmt.Printf("%s", res.Resp.GetChoices()[0].Message.Content)

			}

			// Save the session
			state.AddMessage(domain.Message{
				SourceType: domain.Assistant,
				Content:    accum,
			})

			session.StoreSession(state)
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
