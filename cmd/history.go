package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
	"log/slog"
)

func History() *cobra.Command {

	type HistoryCmdParams struct {
		config.CliSubcParams
		Format boa.Required[string] `name:"format" descr:"Output format. Valid options: pretty or yaml" default:"pretty"`
	}

	p := HistoryCmdParams{}
	return boa.Cmd{
		Use:   "history",
		Short: "Prints the conversation history of the current session",
		ValidArgsFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return storedSessionIDs(), cobra.ShellCompDirectiveDefault
		},
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
			state := session.LoadSession(session.GetSessionID(p.Session.GetOrElse("")))
			oneMsgPrinted := false
			for _, entry := range state.History {
				if entry.Type == "message" {
					if p.Format.Value() == "pretty" {
						fmt.Printf("\n----------------------\n")
						fmt.Printf("|  %s\n", entry.Message.SourceType)
						fmt.Printf("-------------\n")
						fmt.Printf("%s\n", entry.Message.Content)
					} else if p.Format.Value() == "yaml" {
						if oneMsgPrinted {
							fmt.Printf("---\n")
						}
						fmt.Printf("%s", entry.Message.ToYaml())
					} else {
						common.FailAndExit(1, fmt.Sprintf("Unsupported format: %s", p.Format.Value()))
					}
					oneMsgPrinted = true
				} else {
					slog.Warn(fmt.Sprintf("Unsupported entry type: %s", entry.Type))
				}
			}
		},
	}.ToCobra()
}
