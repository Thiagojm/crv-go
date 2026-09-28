package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type InstanceLock struct {
	DataDir  string
	LockPath string
	MetaPath string
	file     *os.File
}

type instanceMeta struct {
	PID  int    `json:"pid"`
	Port int    `json:"port"`
	URL  string `json:"url"`
}

func DefaultDataDir() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "CRV-Go"), nil
}

func AcquireInstance(dataDir string) (*InstanceLock, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dataDir, "instance.lock")
	metaPath := filepath.Join(dataDir, "instance.json")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		meta, _ := readMeta(metaPath)
		if meta != nil && meta.URL != "" {
			return nil, fmt.Errorf("outra instância já está usando este diretório de dados (pid %d): %s", meta.PID, meta.URL)
		}
		return nil, errors.New("outra instância já está usando este diretório de dados")
	}
	return &InstanceLock{DataDir: dataDir, LockPath: lockPath, MetaPath: metaPath, file: f}, nil
}

func (l *InstanceLock) WriteMeta(port int, url string) error {
	meta := instanceMeta{PID: os.Getpid(), Port: port, URL: url}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.MetaPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.MetaPath)
}

func (l *InstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = unlockFile(l.file)
	err := l.file.Close()
	l.file = nil
	_ = os.Remove(l.MetaPath)
	return err
}

func readMeta(path string) (*instanceMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m instanceMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func Platform() string { return runtime.GOOS }
