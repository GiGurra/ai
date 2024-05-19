package common

import (
	"log/slog"
	"os"
)

func FailAndExit(code int, msg string) {
	slog.Error(msg)
	os.Exit(code)
}
