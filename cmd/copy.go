package cmd

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func Copy() *cobra.Command {
	var p struct {
		Arg1    boa.Required[string] `descr:"arg1 (new name if 1 arg, from name if 2 args)" positional:"true"`
		Arg2    boa.Optional[string] `descr:"arg2 (to name if 2 args)" positional:"true"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Cmd{
		Use:    "copy",
		Short:  "Copy a session",
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
			if p.Arg2.HasValue() {
				session.CopySession(p.Arg1.Value(), *p.Arg2.Value())
			} else {
				session.CopySession("", p.Arg1.Value())
			}
		},
	}.ToCobra()
}
