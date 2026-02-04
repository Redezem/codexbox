package project

import (
	"os"
	"testing"
)

func TestDetectDirScopeStable(t *testing.T) {
	dir, err := os.MkdirTemp("", "codexbox-project-")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(dir)

	info1, err := Detect(ScopeDir, dir)
	if err != nil {
		t.Fatalf("detect 1: %v", err)
	}
	info2, err := Detect(ScopeDir, dir)
	if err != nil {
		t.Fatalf("detect 2: %v", err)
	}

	if info1.Root != dir {
		t.Fatalf("expected root %s, got %s", dir, info1.Root)
	}
	if info1.ID == "" {
		t.Fatalf("expected non-empty id")
	}
	if info1.ID != info2.ID {
		t.Fatalf("expected stable id, got %s vs %s", info1.ID, info2.ID)
	}
}
