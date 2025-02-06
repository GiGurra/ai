package common

import (
	"fmt"
	"log/slog"
	"os"
)

func FailAndExit(code int, msg string) {
	slog.Error(fmt.Sprintf("Exiting with code %d: %s", code, msg))
	os.Exit(code)
}

func AppDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		FailAndExit(1, fmt.Sprintf("failed to get home dir: %v", err))
	}
	dir := homeDir + "/.config/gigurra/ai"
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		FailAndExit(1, fmt.Sprintf("failed to create config dir: %v", err))
	}

	return dir
}

func CfgOrDefaultF(cfgVal float64, defaultVal float64) float64 {
	if cfgVal > 0 {
		return cfgVal
	}
	return defaultVal
}

func CfgOrDefaultI(cfgVal int, defaultVal int) int {
	if cfgVal != 0 {
		return cfgVal
	}
	return defaultVal
}
