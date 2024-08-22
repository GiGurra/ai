package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode"
)

type HistoryEntry struct {
	Type    string         `json:"type"`
	Message domain.Message `json:"message"`
}

type State struct {
	Header
	History   []HistoryEntry `json:"history"`
	StateFile string         `json:"-"`
}

type Header struct {
	SessionID         string    `json:"session_id"`
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

func StoredSessionExists(sessionID string) bool {
	sessionDir := Dir() + "/" + sessionID
	exists, err := util.FileExists(sessionDir)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to check session dir: %v", err))
	}
	return exists
}

func LoadSession(sessionID string) State {
	sessionDir := Dir() + "/" + sessionID
	_, err := util.ReadFaileAsJson[Header](sessionDir + "/header.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return State{
				Header: Header{
					SessionID: sessionID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
		} else {
			common.FailAndExit(1, fmt.Sprintf("Failed to read session header: %v", err))
		}
	}

	stateFile := sessionDir + "/state.json"
	state, err := util.ReadFaileAsJson[State](stateFile)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to read session state: %v", err))
	}

	state.StateFile = stateFile

	return state
}

func StoreSession(state State) {
	state.UpdatedAt = time.Now()
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

func cliCommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func BootID() string {
	if util.Must(util.FileExists("/proc/sys/kernel/random/boot_id")) {
		bytes, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to read boot_id: %v", err))
		}
		return strings.TrimSpace(string(bytes))
	} else if cliCommandExists("sysctl") {
		// assume macos, try journalctl: sysctl -n kern.boottime
		cmd := exec.Command("sysctl", "-n", "kern.boottime")
		out, err := cmd.Output()
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to get boottime: %v", err))
		}
		// hash the output
		return HashString(strings.TrimSpace(string(out)))
	} else if util.IsWindows() {
		return HashString(util.BootTimeWindowsStr())
	} else {
		common.FailAndExit(1, "Failed to find boot_id. Could not find /proc/sys/kernel/random/boot_id, sysctl or systeminfo.")
		return ""
	}
}

func HashString(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

func TerminalId() string {
	// First look for WT_SESSION
	// If that isn't found, look for ITERM_SESSION_ID
	// If that isn't found, use parent process PID

	terminalId := os.Getenv("WT_SESSION")
	if terminalId != "" {
		return strings.TrimSpace(terminalId)
	}

	terminalId = os.Getenv("TERM_SESSION_ID")
	if terminalId != "" {
		return strings.TrimSpace(terminalId)
	}

	terminalId = os.Getenv("ITERM_SESSION_ID")
	if terminalId != "" {
		return strings.TrimSpace(terminalId)
	}

	return fmt.Sprintf("%d", os.Getppid())
}

func TerminalSessionID() string {
	return TerminalId() + "." + BootID()
}

func GetSessionID(sessionOverride string) string {
	if sessionOverride != "" {
		return sessionOverride
	}

	terminalSessionId := TerminalSessionID()
	lkupDir := LookupDir()
	mappingFile := lkupDir + "/" + terminalSessionId
	exists, err := util.FileExists(mappingFile)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to check session mapping file: %v", err))
	}

	if exists {
		sessionID, err := os.ReadFile(mappingFile)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to read session mapping file: %v", err))
		}
		return string(sessionID)
	} else {
		return NewSession("").SessionID
	}
}

func DeleteSession(sessionID string, yes bool) {
	currentSessionID := GetSessionID("")
	if sessionID == "" {
		sessionID = currentSessionID
	}
	if sessionID == "" {
		fmt.Printf("No current session or session provided to delete\n")
		return
	}

	if !StoredSessionExists(sessionID) {
		fmt.Printf("Session empty or not found: %s, nothing to delete\n", sessionID)
		return
	}

	s := LoadSession(sessionID)
	if !yes {
		fmt.Printf("session: %s (i=%d/%d, o=%d/%d, created %v)\n", s.SessionID, s.InputTokens, s.InputTokensAccum, s.OutputTokens, s.OutputTokensAccum, s.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Are you sure you want to delete session: %s? (y/n): ", sessionID)

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to read response: %v", err))
		}

		if !strings.HasPrefix(strings.ToLower(response), "y") {
			fmt.Printf("Aborted\n")
			return
		}
	}

	sessionDir := Dir() + "/" + sessionID
	if util.Must(util.FileExists(sessionDir)) {
		err := os.RemoveAll(sessionDir)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to delete session dir: %v", err))
		}
	} else {
		fmt.Printf("Session not found: %s, nothing to delete\n", sessionID)
	}

	if sessionID == currentSessionID {
		QuitSession("")
	}
}

func SetSession(sessionId string) {
	lkupDir := LookupDir()
	terminalSessionId := TerminalSessionID()
	mappingFile := lkupDir + "/" + terminalSessionId

	alreadyExists, err := util.FileExists(mappingFile)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to check session mapping file: %v", err))
	}
	if alreadyExists {
		err = os.Remove(mappingFile)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to remove existing session mapping file: %v", err))
		}
	}

	err = os.WriteFile(mappingFile, []byte(sessionId), 0644)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to write session mapping file: %v", err))
	}
}

func RenameSession(sessionID string, newSessionID string) {
	old, _ := CopySession(sessionID, newSessionID)
	if StoredSessionExists(old) {
		DeleteSession(old, true)
	}
}

func IsAllowedNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ' ' || r == '.'
}

func IsValidSessionName(name string) bool {
	for _, r := range name {
		if !IsAllowedNameChar(r) {
			return false
		}
	}
	return true
}

func CopySession(sessionID string, newSessionID string) (string, string) {
	if !IsValidSessionName(newSessionID) {
		common.FailAndExit(1, fmt.Sprintf("Invalid session name: %s", newSessionID))
	}

	curSessionID := GetSessionID("")
	if sessionID == "" {
		sessionID = curSessionID
	}

	if sessionID == "" {
		common.FailAndExit(1, "No session to rename")
	}

	if newSessionID == "" {
		common.FailAndExit(1, "No new session id provided")
	}

	if StoredSessionExists(newSessionID) {
		common.FailAndExit(1, fmt.Sprintf("Session already exists: %s", newSessionID))
	}

	if !StoredSessionExists(sessionID) {
		if sessionID == curSessionID {
			SetSession(newSessionID)
			return sessionID, newSessionID
		}
		common.FailAndExit(1, fmt.Sprintf("Session not found: %s", sessionID))
	}

	s := LoadSession(sessionID)
	s.SessionID = newSessionID

	StoreSession(s)
	if sessionID == curSessionID {
		SetSession(newSessionID)
	}

	return sessionID, newSessionID
}

func QuitSession(sessionOverride string) {
	if sessionOverride != "" {
		common.FailAndExit(1, "Cannot quit session with external session override")
	}

	lkupDir := LookupDir()
	terminalSessionId := TerminalSessionID()
	mappingFile := lkupDir + "/" + terminalSessionId
	exists, err := util.FileExists(mappingFile)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to check session mapping file: %v", err))
	}

	if exists {
		err = os.Remove(mappingFile)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("Failed to remove session mapping file: %v", err))
		}
	}
}

func NewSession(newSessionName string) State {
	if !IsValidSessionName(newSessionName) {
		common.FailAndExit(1, fmt.Sprintf("Invalid session name: %s", newSessionName))
	}

	sessionID := func() string {
		if newSessionName != "" {
			return newSessionName
		} else {
			return uuid.NewString()
		}
	}()

	if StoredSessionExists(sessionID) {
		common.FailAndExit(1, fmt.Sprintf("Session already exists: %s", sessionID))
	}

	QuitSession("")

	SetSession(sessionID)

	return State{
		Header: Header{
			SessionID: sessionID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

func LookupDir() string {
	tmpDir := os.TempDir()
	bootId := BootID()
	tmpDir = tmpDir + "/gigurra/ai-session-lkup/" + bootId
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to create session lookup dir: %v", err))
	}
	return tmpDir
}

func Dir() string {
	appDir := common.AppDir()
	res := appDir + "/sessions"
	err := os.MkdirAll(res, 0755)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to create session storage dir: %v", err))
	}
	return res
}

func (s *State) AddMessage(message domain.Message) {
	s.History = append(s.History, HistoryEntry{
		Type:    "message",
		Message: message,
	})
}

func (s *State) MessageHistory() []domain.Message {
	return lo.Map(lo.Filter(s.History, func(item HistoryEntry, _ int) bool {
		return item.Type == "message"
	}), func(entry HistoryEntry, _ int) domain.Message {
		return entry.Message
	})
}
