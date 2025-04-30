package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/cmd"
	"github.com/gigurra/ai/config"
	"github.com/spf13/cobra"
)

func main() {

	cliParams := &config.CliParams{}

	boa.Cmd{
		Use:         "ai",
		Short:       "ai/llm conversation tool, every terminal is a conversation",
		ParamEnrich: config.CliParamEnricher,
		Params:      cliParams,
		Args:        cobra.MinimumNArgs(0),
		SubCommands: []*cobra.Command{
			cmd.Sessions(),
			cmd.Session(),
			cmd.Status(),
			cmd.Config(),
			cmd.History(),
			cmd.NewOrResetCmd("new"),
			cmd.NewOrResetCmd("reset"),
			cmd.SetOrLoadSession("set"),
			cmd.SetOrLoadSession("load"),
			cmd.Delete(),
			cmd.Rename(),
			cmd.Copy(),
			cmd.NameAll(),
			cmd.Prep(),
			cmd.Pull(),
			cmd.Push(),
			cmd.Sync(),
		},
		RunFunc: cmd.Default(cliParams),
	}.Run()
}
