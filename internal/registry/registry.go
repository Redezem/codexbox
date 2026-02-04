package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Entry struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	ImageTag  string    `json:"image_tag"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

type Registry struct {
	Entries map[string]Entry `json:"entries"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "codexbox", "registry.json"), nil
	default:
		return filepath.Join(home, ".local", "share", "codexbox", "registry.json"), nil
	}
}

func Load(path string) (Registry, error) {
	reg := Registry{Entries: map[string]Entry{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return reg, nil
		}
		return reg, err
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return reg, err
	}
	if reg.Entries == nil {
		reg.Entries = map[string]Entry{}
	}
	return reg, nil
}

func Save(path string, reg Registry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
