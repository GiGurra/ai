package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func Sessions() *cobra.Command {
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:         "sessions",
		Short:       "List all stored sessions",
		Params:      &p,
		ParamEnrich: config.CliParamEnricher,
		Run: func(cmd *cobra.Command, args []string) {
			sessions := session.ListSessions()
			currentSession := session.GetSessionID(p.Session.GetOrElse(""))
			if p.Verbose.Value() {
				for _, s := range sessions {
					currentSessionSuffix := ""
					if s.SessionID == currentSession {
						currentSessionSuffix = " [ *current* ]"
					}
					fmt.Printf("%s (i=%d/%d, o=%d/%d, created %v)%s\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"), currentSessionSuffix)
				}
			} else {
				for _, s := range sessions {
					currentSessionSuffix := ""
					if s.SessionID == currentSession {
						currentSessionSuffix = " [ *current* ]"
					}
					fmt.Printf("%s%s\n", s.SessionID, currentSessionSuffix)
				}
			}

		},
	}.ToCmd()
}
