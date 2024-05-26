package util

import (
	"fmt"
	"log/slog"
	"testing"
)

func TestBootTimeWindowsStr(t *testing.T) {
	if !IsWindows() {
		slog.Warn("TestBootTimeWindowsStr skipped, not running on Windows")
		return
	}

	bootTimeStr := BootTimeWindowsStr()
	slog.Info(fmt.Sprintf("Boot time: %v", bootTimeStr))
}
