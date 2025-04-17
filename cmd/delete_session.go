package cmd

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func Delete() *cobra.Command {
	var p struct {
		Session boa.Optional[string] `descr:"Session id" positional:"true" name:"session"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
		Yes     boa.Required[bool]   `descr:"Auto confirm" short:"y" default:"false" name:"yes"`
	}

	return boa.Wrap{
		Use:   "delete",
		Short: "Delete a session, or the current session if no session id is provided",
		ValidArgsFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return storedSessionIDs(), cobra.ShellCompDirectiveDefault
		},
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			session.DeleteSession(p.Session.GetOrElse(""), p.Yes.Value())
		},
	}.ToCmd()
}
