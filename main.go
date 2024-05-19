package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"log/slog"
)

type Params struct {
	Quiet    boa.Required[bool]   `descr:"Quiet mode, requires no user input" short:"q" default:"false"`
	Provider boa.Required[string] `descr:"AI provider to use" short:"p" default:"openai"`
	Model    boa.Required[string] `descr:"Model to use" short:"m" default:"gpt-4o"`
}

func main() {

	p := Params{}

	boa.Wrap{
		Use:   "ai",
		Short: "ai cli tool, you know, for building stuff",
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherBool,
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &p,
		Long:   `See the README.MD for more information`,
		Run: func(cmd *cobra.Command, args []string) {

			slog.Info("Starting ai")
			slog.Info(fmt.Sprintf("Will use provider: %s", p.Provider.Value()))
		},
	}.ToApp()
}
