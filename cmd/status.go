package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
	"strings"
)

func Status() *cobra.Command {
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:    "status",
		Short:  "Prints info about current session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			_, cfgInFile := config.LoadCfgFile()
			s := session.LoadSession(session.GetSessionID(p.Session.GetOrElse("")))
			provider := p.Provider.GetOrElse(cfgInFile.Provider)
			provider = strings.ReplaceAll(strings.TrimSpace(provider), "_", "-")
			fmt.Printf("current provider: %s\n", provider)
			fmt.Printf("current model: %s\n", cfgInFile.Model(provider))
			fmt.Printf("config file: %s\n", config.CfgFilePath())
			fmt.Printf("storage dir: %s\n", session.Dir())
			fmt.Printf("lookup dir: %s\n", session.LookupDir())
			fmt.Printf("current session: %s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("current session file: %s\n", s.StateFile)
		},
	}.ToCmd()
}
