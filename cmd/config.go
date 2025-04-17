package cmd

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/ai/config"
	"github.com/spf13/cobra"
)

func Config() *cobra.Command {
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
