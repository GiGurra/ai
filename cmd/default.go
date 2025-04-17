package cmd

import (
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers"
	"github.com/gigurra/ai/session"
	"github.com/gigurra/ai/util"
	"github.com/spf13/cobra"
	"log/slog"
	"strings"
)

func Default(cliParams *config.CliParams) func(cmd *cobra.Command, args []string) {

	return func(cmd *cobra.Command, args []string) {

		question := strings.Join(args, " ")

		if cliParams.Verbose.Value() {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		}

		cfgFilePath, storedCfg := config.LoadCfgFile()
		cfg := config.ValidateCfg(cfgFilePath, storedCfg, cliParams)

		provider := providers.CreateProvider(cfg)

		stdInAttachment, err := util.ReadAllStdIn()
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to read attachment from stdin: %v", err))
		}
		if stdInAttachment != "" {
			if question != "" {
				footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInAttachment)
				question = fmt.Sprintf("%s\n%s", question, footer)
			} else {
				question = stdInAttachment
			}
		}

		if question == "" {
			common.FailAndExit(1, "No data provided")
		}

		state := session.LoadSession(session.GetSessionID(cliParams.Session.GetOrElse("")))
		messageHistory := state.MessageHistory()

		newMessage := domain.Message{
			SourceType: domain.User,
			Content:    question,
		}

		stream := provider.BasicAskStream(domain.Question{
			Messages: append(messageHistory, newMessage),
		})

		inputTokens := 0
		outputTokens := 0
		accum := strings.Builder{}
		for {
			res, hasMore := <-stream
			if !hasMore {
				if len(accum.String()) == 0 {
					common.FailAndExit(1, "No response handled from ai provider")
				}
				break // stream done
			}
			if res.Err != nil {
				common.FailAndExit(1, fmt.Sprintf("Failed to receive stream response: %v", res.Err))
			}

			inputTokens += res.Resp.GetUsage().PromptTokens
			outputTokens += res.Resp.GetUsage().CompletionTokens

			if len(res.Resp.GetChoices()) == 0 {
				continue
			}

			accum.WriteString(res.Resp.GetChoices()[0].Message.Content)
			fmt.Printf("%s", res.Resp.GetChoices()[0].Message.Content)

		}

		fmt.Printf("\n")

		state.InputTokensAccum += inputTokens
		state.InputTokens = inputTokens
		state.OutputTokensAccum += outputTokens
		state.OutputTokens = outputTokens
		state.AddMessage(newMessage)
		state.AddMessage(domain.Message{
			SourceType: domain.Assistant,
			Content:    accum.String(),
		})

		session.StoreSession(state)
	}
}
