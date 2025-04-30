package cmd

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func SetOrLoadSession(alias string) *cobra.Command {
	var p struct {
		Session boa.Required[string] `descr:"Session id" positional:"true"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Cmd{
		Use:   alias,
		Short: "Set/load an existing ai session",
		ValidArgsFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return storedSessionIDs(), cobra.ShellCompDirectiveDefault
		},
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
			session.SetSession(p.Session.Value())
		},
	}.ToCobra()
}
