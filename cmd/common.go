package cmd

import (
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/session"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"strings"
)

// Helper functions moved from main.go

func storedSessionIDs() []string {
	availableSessions := session.ListSessions()
	return lo.Map(availableSessions, func(s session.Header, _ int) string {
		return s.SessionID
	})
}

func askYesNo(question string) bool {
	fmt.Printf("%s (y/n) ", question)
	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to read answer: %v", err))
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y")
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
