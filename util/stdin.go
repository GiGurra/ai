package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func ReadAllStdIn() (string, error) {
	stdInContents := ""
	stat, err := os.Stdin.Stat()
	if err == nil && stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var sb strings.Builder
		for {
			input, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return "", fmt.Errorf("failed to read from stdin: %v", err)
			}
			sb.WriteString(input)
			if err == io.EOF {
				break
			}
		}
		stdInContents = strings.TrimSpace(sb.String())
	}
	return stdInContents, nil
}
