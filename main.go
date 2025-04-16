package main

import (
	"context"
	"fmt"
	"github.com/GiGurra/cmder"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers"
	"github.com/gigurra/ai/session"
	"github.com/gigurra/ai/util"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func main() {

	cliParams := config.CliParams{}

	boa.Wrap{
		Use:         "ai",
		Short:       "ai/llm conversation tool, every terminal is a conversation",
		ParamEnrich: config.CliParamEnricher,
		Params:      &cliParams,
		Args:        cobra.MinimumNArgs(0),
		SubCommands: []*cobra.Command{
			sessionsCmd(),
			sessionCmd(),
			statusCmd(),
			configCmd(),
			historyCmd(),
			newOrResetCmd("new"),
			newOrResetCmd("reset"),
			setSessionCmd(),
			deleteSessionCmd(),
			renameCmd(),
			copyCmd(),
			nameAll(),
			prepCmd(),
			pullCmd(),
			pushCmd(),
		},
		Run: func(cmd *cobra.Command, args []string) {

			question := strings.Join(args, " ")

			if cliParams.Verbose.Value() {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, cliParams)

			provider := providers.CreateProvider(cfg)

			stdInAttachment, err := util.ReadAllStdIn()
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("Failed to read attachment from stdin: %v", err))
			}
			if stdInAttachment != "" {
				if question != "" {
					footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInAttachment)
					question = fmt.Sprintf("%s\n%s", question, footer)
				} else {
					question = stdInAttachment
				}
			}

			if question == "" {
				common.FailAndExit(1, "No data provided")
			}

			state := session.LoadSession(session.GetSessionID(cliParams.Session.GetOrElse("")))
			messageHistory := state.MessageHistory()

			newMessage := domain.Message{
				SourceType: domain.User,
				Content:    question,
			}

			stream := provider.BasicAskStream(domain.Question{
				Messages: append(messageHistory, newMessage),
			})

			inputTokens := 0
			outputTokens := 0
			accum := strings.Builder{}
			for {
				res, hasMore := <-stream
				if !hasMore {
					if len(accum.String()) == 0 {
						common.FailAndExit(1, "No response handled from ai provider")
					}
					break // stream done
				}
				if res.Err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to receive stream response: %v", res.Err))
				}

				inputTokens += res.Resp.GetUsage().PromptTokens
				outputTokens += res.Resp.GetUsage().CompletionTokens

				if len(res.Resp.GetChoices()) == 0 {
					continue
				}

				accum.WriteString(res.Resp.GetChoices()[0].Message.Content)
				fmt.Printf("%s", res.Resp.GetChoices()[0].Message.Content)

			}

			fmt.Printf("\n")

			state.InputTokensAccum += inputTokens
			state.InputTokens = inputTokens
			state.OutputTokensAccum += outputTokens
			state.OutputTokens = outputTokens
			state.AddMessage(newMessage)
			state.AddMessage(domain.Message{
				SourceType: domain.Assistant,
				Content:    accum.String(),
			})

			session.StoreSession(state)
		},
	}.ToApp()
}

func sessionsCmd() *cobra.Command {
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

func sessionCmd() *cobra.Command {
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:    "session",
		Short:  "Print id of current session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			sessionId := session.GetSessionID(p.Session.GetOrElse(""))
			if p.Verbose.Value() {
				s := session.LoadSession(sessionId)
				fmt.Printf("%s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("%s\n", sessionId)
			}
		},
	}.ToCmd()
}

func statusCmd() *cobra.Command {
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

func configCmd() *cobra.Command {
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:    "config",
		Short:  "Prints the current configuration",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, p.ToCliParams())
			cfg = cfg.WithoutSecrets()
			fmt.Printf("--- %s ---\n%s", cfgFilePath, cfg.ToYaml())
		},
	}.ToCmd()
}

func historyCmd() *cobra.Command {

	type HistoryCmdParams struct {
		config.CliSubcParams
		Format boa.Required[string] `name:"format" descr:"Output format. Valid options: pretty or yaml" default:"pretty"`
	}

	p := HistoryCmdParams{}
	return boa.Wrap{
		Use:   "history",
		Short: "Prints the conversation history of the current session",
		ValidArgsFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return storedSessionIDs(), cobra.ShellCompDirectiveDefault
		},
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
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
	}.ToCmd()
}

func newOrResetCmd(name string) *cobra.Command {
	var p struct {
		Session boa.Optional[string] `descr:"Session id" positional:"true" name:"session"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    name,
		Short:  "Create a new session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			session.NewSession(p.Session.GetOrElse(""))
		},
	}.ToCmd()
}

func setSessionCmd() *cobra.Command {
	var p struct {
		Session boa.Required[string] `descr:"Session id" positional:"true"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:   "set",
		Short: "Set the ai session",
		ValidArgsFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return storedSessionIDs(), cobra.ShellCompDirectiveDefault
		},
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			session.SetSession(p.Session.Value())
		},
	}.ToCmd()
}

func renameCmd() *cobra.Command {
	var p struct {
		Arg1    boa.Required[string] `descr:"arg1 (new name if 1 arg, from name if 2 args)" positional:"true"`
		Arg2    boa.Optional[string] `descr:"arg2 (to name if 2 args)" positional:"true"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "rename",
		Short:  "Rename a session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			if p.Arg2.HasValue() {
				session.RenameSession(p.Arg1.Value(), *p.Arg2.Value())
			} else {
				session.RenameSession("", p.Arg1.Value())
			}
		},
	}.ToCmd()
}

func copyCmd() *cobra.Command {
	var p struct {
		Arg1    boa.Required[string] `descr:"arg1 (new name if 1 arg, from name if 2 args)" positional:"true"`
		Arg2    boa.Optional[string] `descr:"arg2 (to name if 2 args)" positional:"true"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "copy",
		Short:  "Copy a session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			if p.Arg2.HasValue() {
				session.CopySession(p.Arg1.Value(), *p.Arg2.Value())
			} else {
				session.CopySession("", p.Arg1.Value())
			}
		},
	}.ToCmd()
}

func nameAll() *cobra.Command {
	var p struct {
		Verbose  boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
		Yes      boa.Required[bool]   `descr:"Verbose output" short:"y" default:"false" name:"yes"`
		Provider boa.Optional[string] `descr:"AI provider to use" name:"provider" env:"AI_PROVIDER" short:"p"`
	}
	return boa.Wrap{
		Use:    "name-all",
		Short:  "generate names to replace UUID session IDs",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			sessions := session.ListSessions()

			cfgFilePath, storedCfg := config.LoadCfgFile()
			cfg := config.ValidateCfg(cfgFilePath, storedCfg, config.CliParams{Provider: p.Provider})
			provider := providers.CreateProvider(cfg)

			sessionsToRename := lo.Filter(sessions, func(s session.Header, _ int) bool {
				return isUUID(s.SessionID)
			})

			if len(sessionsToRename) == 0 {
				fmt.Printf("No UUID sessions to rename\n")
				return
			}

			for _, s := range sessionsToRename {

				fmt.Printf("Processing session %s\n", s.SessionID)

				if !p.Yes.Value() {
					newQuestionTokens := s.InputTokens + s.OutputTokens // the last inputTokenCount and outputTokenCount is what we will send as input to the next question
					res := askYesNo("  The total input tokens that will be used to generate the name: " + strconv.Itoa(newQuestionTokens) +
						"\n  Do you wish to auto-assign a name to it?")
					if !res {
						fmt.Printf("  Skipping session %s\n", s.SessionID)
						continue
					}
				}

				sessionData := session.LoadSession(s.SessionID)

				newQuestionMsgs := append(sessionData.MessageHistory(), domain.Message{
					SourceType: domain.User,
					Content:    "Please summarize this conversation in 3 words, concatenated with _ (underscores)",
				})

				resp, err := provider.BasicAsk(domain.Question{
					Messages: newQuestionMsgs,
				})
				if err != nil {
					common.FailAndExit(1, fmt.Sprintf("Failed to ask question: %v", err))
				}

				if len(resp.GetChoices()) == 0 {
					common.FailAndExit(1, "No response from provider")
				}

				newName := strings.ToLower(resp.GetChoices()[0].Message.Content)
				newName = string(lo.Filter([]rune(newName), func(r rune, _ int) bool {
					return session.IsAllowedNameChar(r)
				}))
				newName = strings.TrimSpace(newName)

				if newName == "" {
					common.FailAndExit(1, "No name returned from provider")
				}

				fmt.Printf("  ==>>> %s\n", newName)

				if !p.Yes.Value() {
					res := askYesNo("  Do you wish to assign this name to the session?")
					if !res {
						fmt.Printf("  Skipping session %s\n", s.SessionID)
						continue
					}
				}

				session.RenameSession(s.SessionID, newName)
			}

		},
	}.ToCmd()
}

func deleteSessionCmd() *cobra.Command {
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

func prepCmd() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "prep",
		Short:  "Add a user message to the current session without sending a question",
		Params: &p,
		Args:   cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			question := strings.Join(args, " ")

			stdInAttachment, err := util.ReadAllStdIn()
			if err != nil {
				common.FailAndExit(1, fmt.Sprintf("Failed to read attachment from stdin: %v", err))
			}
			if stdInAttachment != "" {
				if question != "" {
					footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInAttachment)
					question = fmt.Sprintf("%s\n%s", question, footer)
				} else {
					question = stdInAttachment
				}
			}

			if question == "" {
				common.FailAndExit(1, "No data provided")
			}

			state := session.LoadSession(session.GetSessionID(""))
			newMessage := domain.Message{
				SourceType: domain.User,
				Content:    question,
			}

			state.AddMessage(newMessage)
			session.StoreSession(state)

			if p.Verbose.Value() {
				fmt.Printf("Added message to session %s: %s\n", state.SessionID, question)
			}
		},
	}.ToCmd()
}

func pullCmd() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "pull",
		Short:  "Pull the latest conversation sessions/session updates from git remote",
		Params: &p,
		Args:   cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

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
	}.ToCmd()
}

func pushCmd() *cobra.Command {
	var p struct {
		Verbose boa.Required[bool] `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
	}
	return boa.Wrap{
		Use:    "pull",
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

func storedSessionIDs() []string {
	availableSessions := session.ListSessions()
	return lo.Map(availableSessions, func(s session.Header, _ int) string {
		return s.SessionID
	})
}

func askYesNo(question string) bool {
	fmt.Printf("%s (y/n) ", question)
	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to read answer: %v", err))
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y")
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
