//go:build !windows

package util

func BootTimeWindowsStr() string {
	panic("called BootTimeWindowsStr on non-windows platform!")
}
