package common

import (
	"fmt"
	"log/slog"
	"os"
)

func FailAndExit(code int, msg string) {
	slog.Error(msg)
	os.Exit(code)
}

func AppDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get home dir: %w", err))
	}
	dir := homeDir + "/.config/gigurra/ai"
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		panic(fmt.Errorf("failed to create config dir: %w", err))
	}

	return dir
}
