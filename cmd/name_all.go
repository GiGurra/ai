package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers"
	"github.com/gigurra/ai/session"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

func NameAll() *cobra.Command {
	var p struct {
		Verbose  boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
		Yes      boa.Required[bool]   `descr:"Verbose output" short:"y" default:"false" name:"yes"`
		Provider boa.Optional[string] `descr:"AI provider to use" name:"provider" env:"AI_PROVIDER" short:"p"`
	}
	return boa.Cmd{
		Use:    "name-all",
		Short:  "generate names to replace UUID session IDs",
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
			sessions := session.ListSessions()

			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, &config.CliParams{Provider: p.Provider})
			provider := providers.CreateProvider(cfg)

			sessionsToRename := lo.Filter(sessions, func(s session.Header, _ int) bool {
				return isUUID(s.SessionID)
			})

			if len(sessionsToRename) == 0 {
				fmt.Printf("No UUID sessions to rename\n")
				return
			}

			for _, s := range sessionsToRename {

				fmt.Printf("Processing session %s\n", s.SessionID)

				if !p.Yes.Value() {
					newQuestionTokens := s.InputTokens + s.OutputTokens // the last inputTokenCount and outputTokenCount is what we will send as input to the next question
					res := askYesNo("  The total input tokens that will be used to generate the name: " + strconv.Itoa(newQuestionTokens) +
						"\n  Do you wish to auto-assign a name to it?")
					if !res {
						fmt.Printf("  Skipping session %s\n", s.SessionID)
						continue
					}
				}

				sessionData := session.LoadSession(s.SessionID)

				newQuestionMsgs := append(sessionData.MessageHistory(), domain.Message{
					SourceType: domain.User,
					Content:    "Please summarize this conversation in 3 words, concatenated with _ (underscores)",
				})

				resp, err := provider.BasicAsk(domain.Question{
					Messages: newQuestionMsgs,
				})
				if err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to ask question: %v", err))
				}

				if len(resp.GetChoices()) == 0 {
					common.FailAndExit(1, "No response from provider")
				}

				newName := strings.ToLower(resp.GetChoices()[0].Message.Content)
				newName = string(lo.Filter([]rune(newName), func(r rune, _ int) bool {
					return session.IsAllowedNameChar(r)
				}))
				newName = strings.TrimSpace(newName)

				if newName == "" {
					common.FailAndExit(1, "No name returned from provider")
				}

				fmt.Printf("  ==>>> %s\n", newName)

				if !p.Yes.Value() {
					res := askYesNo("  Do you wish to assign this name to the session?")
					if !res {
						fmt.Printf("  Skipping session %s\n", s.SessionID)
						continue
					}
				}

				session.RenameSession(s.SessionID, newName)
			}

		},
	}.ToCobra()
}
