package cmd

import (
	"context"
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/session"
	"github.com/spf13/cobra"
	"os"
)

func Push() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "push",
		Short:  "Push the latest conversation sessions/session updates to git remote",
		Params: &p,
		Args:   cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			sessionsDir := session.Dir()
			fmt.Printf("Pushing latest sessions %s -> git remote\n", sessionsDir)

			// check that this is a git dir, but using git status
			// if it is not a git dir, we will not be able to pull
			res := cmder.New("git", "status").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git status on sessions dir:\n- %v", res.Combined))
			}

			// Check if we have any changes to commit, with git porcelain
			res = cmder.New("git", "status", "--porcelain").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git status on sessions dir:\n- %v", res.Combined))
			}
			if len(res.Combined) == 0 {
				fmt.Printf("No changes to commit, exiting\n")
				return
			}

			fmt.Printf("Will first pull latest from git remote -> %s\n", sessionsDir)
			res = cmder.New("git", "pull").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git pull on sessions dir:\n- %v", res.Combined))
			}

			res = cmder.New("git", "add", "-A").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git add on sessions dir:\n- %v", res.Combined))
			}

			hostName, err := os.Hostname()
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("Failed to get hostname: %v", err))
			}

			res = cmder.New("git", "commit", "-m", "latest session updates from host: "+hostName).WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git commit on sessions dir:\n- %v", res.Combined))
			}

			res = cmder.New("git", "push").WithWorkingDirectory(sessionsDir).Run(context.Background())
			if res.Err != nil {
				common.FailAndExit(res.ExitCode, fmt.Sprintf("Failed to run git push on sessions dir:\n- %v", res.Combined))
			}

			fmt.Printf("- %s", res.Combined)
		},
	}.ToCmd()
}
