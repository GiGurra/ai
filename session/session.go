package session

import (
	"errors"
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"io/fs"
	"log/slog"
	"os"
	"time"
)

type HistoryEntry struct {
	Type    string         `json:"type"`
	Message domain.Message `json:"message"`
}

type State struct {
	Header
	History []HistoryEntry `json:"history"`
}

type Header struct {
	SessionID string    `json:"session_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ListSessions() []Header {
	// list dirs inside session dir
	// for each dir, read header file
	sessionDir := SessionDir()

	// list dirs inside session dir
	dirEntries, err := os.ReadDir(sessionDir)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to list session dirs: %v", err))
	}

	var headers []Header
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		header, err := util.ReadFaileAsJson[Header](sessionDir + "/" + dirEntry.Name() + "/header.json")
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to read header file: %s, %v", dirEntry.Name(), err))
			continue
		}

		headers = append(headers, header)
	}

	return headers
}

func LoadSession(sessionID string) State {
	sessionDir := SessionDir() + "/" + sessionID
	_, err := util.ReadFaileAsJson[Header](sessionDir + "/header.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return State{
				Header: Header{
					SessionID: sessionID,
					Name:      sessionID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
		} else {
			common.FailAndExit(1, fmt.Sprintf("Failed to read session header: %v", err))
		}
	}

	state, err := util.ReadFaileAsJson[State](sessionDir + "/state.json")
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to read session state: %v", err))
	}

	return state
}

func SessionDir() string {
	appDir := common.AppDir()
	return appDir + "/sessions"
}
