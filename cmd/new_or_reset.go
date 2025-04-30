package cmd

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func NewOrResetCmd(name string) *cobra.Command {
	var p struct {
		Session boa.Optional[string] `descr:"Session id" positional:"true" name:"session"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Cmd{
		Use:    name,
		Short:  "Create a new session",
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
			session.NewSession(p.Session.GetOrElse(""))
		},
	}.ToCobra()
}
