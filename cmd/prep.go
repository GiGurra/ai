package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/session"
	"github.com/gigurra/ai/util"
	"github.com/spf13/cobra"
	"strings"
)

func Prep() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "prep",
		Short:  "Add a user message to the current session without sending a question",
		Params: &p,
		Args:   cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			question := strings.Join(args, " ")

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

			state := session.LoadSession(session.GetSessionID(""))
			newMessage := domain.Message{
				SourceType: domain.User,
				Content:    question,
			}

			state.AddMessage(newMessage)
			session.StoreSession(state)

			if p.Verbose.Value() {
				fmt.Printf("Added message to session %s: %s\n", state.SessionID, question)
			}
		},
	}.ToCmd()
}
