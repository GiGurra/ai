package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"io/fs"
	"log/slog"
	"os"
	"slices"
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
	SessionID         string    `json:"session_id"`
	Name              string    `json:"name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	InputTokens       int       `json:"input_tokens"`
	OutputTokens      int       `json:"output_tokens"`
	InputTokensAccum  int       `json:"input_tokens_accum"`
	OutputTokensAccum int       `json:"output_tokens_accum"`
}

func ListSessions() []Header {
	// list dirs inside session dir
	// for each dir, read header file
	sessionDir := Dir()

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

	// sort by created_at desc
	slices.SortFunc(headers, func(a, b Header) int {
		if a == b {
			return 0
		} else if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		} else {
			return -1
		}
	})

	return headers
}

func LoadSession(sessionID string) State {
	sessionDir := Dir() + "/" + sessionID
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

func StoreSession(state State) {
	sessionDir := Dir() + "/" + state.SessionID
	err := os.MkdirAll(sessionDir, 0755)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to create session dir: %v", err))
	}

	headerBytes, err := json.Marshal(state.Header)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to marshal session header: %v", err))
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to marshal session state: %v", err))
	}
	err = os.WriteFile(sessionDir+"/header.json", headerBytes, 0644)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to write session header: %v", err))
	}
	err = os.WriteFile(sessionDir+"/state.json", stateBytes, 0644)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to write session state: %v", err))
	}
}

func Dir() string {
	appDir := common.AppDir()
	return appDir + "/sessions"
}

func (s *State) AddMessage(message domain.Message) {
	s.History = append(s.History, HistoryEntry{
		Type:    "message",
		Message: message,
	})
}
