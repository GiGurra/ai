package cmd

import (
	"context"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
)

func Pull() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Cmd{
		Use:    "pull",
		Short:  "Pull the latest conversation sessions/session updates from git remote",
		Params: &p,
		Args:   cobra.MinimumNArgs(0),
		RunFunc: func(cmd *cobra.Command, args []string) {

			sessionsDir := session.Dir()
			fmt.Printf("Pulling latest sessions from git remote -> %s\n", sessionsDir)

			// check that this is a git dir, but using git status
			// if it is not a git dir, we will not be able to pull
			res := cmder.New("git", "status").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git status on sessions dir:\n- %v", res.Combined))
			}

			res = cmder.New("git", "pull").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git pull on sessions dir:\n- %v", res.Combined))
			}

			fmt.Printf("- %s", res.Combined)
		},
	}.ToCobra()
}
