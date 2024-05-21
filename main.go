package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers"
	"github.com/gigurra/ai/session"
	"github.com/gigurra/ai/util"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"strings"
)

func main() {

	cliParams := config.CliParams{}

	boa.Wrap{
		Use:         "ai",
		Short:       "ai cli tool, you know, for building stuff",
		ParamEnrich: config.CliParamEnricher,
		Params:      &cliParams,
		Long:        `See the README.MD for more information`,
		Args:        cobra.MinimumNArgs(1),
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
				footer := fmt.Sprintf("\n Attached additional info/data: \n %s", stdInAttachment)
				question = fmt.Sprintf("%s\n%s", question, footer)
			}

			state := session.LoadSession(session.GetSessionID(cliParams.Session.GetOrElse("")))

			messageHistory := lo.Map(lo.Filter(state.History, func(item session.HistoryEntry, index int) bool {
				return item.Type == "message"
			}), func(entry session.HistoryEntry, _ int) domain.Message {
				return entry.Message
			})

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
				res, ok := <-stream
				if !ok {
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
			if p.Verbose.Value() {
				for _, s := range sessions {
					fmt.Printf("%s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
				}
			} else {
				for _, s := range sessions {
					fmt.Printf("%s\n", s.SessionID)
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
			s := session.LoadSession(session.GetSessionID(p.Session.GetOrElse("")))
			fmt.Printf("config file: %s\n", config.CfgFilePath())
			fmt.Printf("storage dir: %s\n", session.Dir())
			fmt.Printf("lookup dir: %s\n", session.SessionLkupDir())
			fmt.Printf("current session: %s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
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
	p := config.CliSubcParams{}
	return boa.Wrap{
		Use:    "history",
		Short:  "Prints the conversation history of the current session",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			state := session.LoadSession(session.GetSessionID(p.Session.GetOrElse("")))
			oneMsgPrinted := false
			for _, entry := range state.History {
				if entry.Type == "message" {
					if oneMsgPrinted {
						fmt.Printf("---\n")
					}
					fmt.Printf("%s", entry.Message.ToYaml())
					oneMsgPrinted = true
				} else {
					slog.Warn("Unsupported entry type: %s", entry.Type)
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
		Use:    "set",
		Short:  "Set the ai session",
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

func deleteSessionCmd() *cobra.Command {
	var p struct {
		Session boa.Optional[string] `descr:"Session id" positional:"true" name:"session"`
		Verbose boa.Required[bool]   `descr:"Verbose output" short:"v" default:"false" name:"verbose"`
		Yes     boa.Required[bool]   `descr:"Auto confirm" short:"y" default:"false" name:"yes"`
	}
	return boa.Wrap{
		Use:    "delete",
		Short:  "Delete a session, or the current session if no session id is provided",
		Params: &p,
		Run: func(cmd *cobra.Command, args []string) {
			session.DeleteSession(p.Session.GetOrElse(""), p.Yes.Value())
		},
	}.ToCmd()
}
