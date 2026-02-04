package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir, err := os.MkdirTemp("", "codexbox-registry-")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "registry.json")
	created := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)
	used := time.Date(2026, 2, 5, 8, 30, 0, 0, time.UTC)
	reg := Registry{
		Entries: map[string]Entry{
			"abc": {
				ID:        "abc",
				Path:      "/tmp/project",
				ImageTag:  "codexbox:latest",
				CreatedAt: created,
				LastUsed:  used,
			},
		},
	}

	if err := Save(path, reg); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	entry, ok := loaded.Entries["abc"]
	if !ok {
		t.Fatalf("missing entry")
	}
	if entry.Path != "/tmp/project" {
		t.Fatalf("path mismatch: %s", entry.Path)
	}
	if entry.CreatedAt.UTC() != created {
		t.Fatalf("created mismatch: %v", entry.CreatedAt)
	}
	if entry.LastUsed.UTC() != used {
		t.Fatalf("last used mismatch: %v", entry.LastUsed)
	}
}
