package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func Session() *cobra.Command {
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:    "session",
		Short:  "Print id of current session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			sessionId := session.GetSessionID(p.Session.GetOrElse(""))
			if p.Verbose.Value() {
				s := session.LoadSession(sessionId)
				fmt.Printf("%s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("%s\n", sessionId)
			}
		},
	}.ToCmd()
}
