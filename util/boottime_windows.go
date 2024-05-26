//go:build windows

package util

import (
	"golang.org/x/sys/windows"
	"syscall"
	"time"
)

func BootTimeWindowsStr() string {
	// Get the number of 100-nanosecond intervals since January 1, 1601 (UTC)
	var now windows.Filetime
	windows.GetSystemTimeAsFileTime(&now)

	// Get the system uptime in milliseconds
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getTickCount64 := kernel32.NewProc("GetTickCount64")
	ret, _, _ := getTickCount64.Call()
	uptime := time.Duration(ret) * time.Millisecond

	// Calculate the boot time
	nowTime := time.Unix(0, now.Nanoseconds())
	bootTime := nowTime.Add(-uptime)

	return bootTime.Format(time.RFC1123)
}
