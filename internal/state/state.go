package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	sessionFileName = "session.json"
)

type Session struct {
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	Dir        string    `json:"dir"`
	HTTPAddr   string    `json:"http_addr"`
	HTTPPort   int       `json:"http_port"`
	SocketPath string    `json:"socket_path"`
	Token      string    `json:"token"`
	Headless   bool      `json:"headless"`
	BrowserPID int       `json:"browser_pid,omitempty"`
	BrowserBin string    `json:"browser_bin,omitempty"`
}

func Dir(stateDir string) string {
	return stateDir
}

func SessionPath(stateDir string) string {
	return filepath.Join(stateDir, sessionFileName)
}

func Load(stateDir string) (Session, error) {
	path := SessionPath(stateDir)
	b, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func Save(stateDir string, s Session) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}
	path := SessionPath(stateDir)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Remove(stateDir string) error {
	path := SessionPath(stateDir)
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
