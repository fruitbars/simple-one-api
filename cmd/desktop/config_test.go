package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDesktopConfigCreatesPrivateDefault(t *testing.T) {
	path, err := resolveDesktopConfig([]string{"desktop"}, t.TempDir())
	if err != nil {
		t.Fatalf("resolve desktop config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read desktop config: %v", err)
	}
	if string(data) != defaultDesktopConfig {
		t.Fatal("unexpected default desktop config")
	}
	if filepath.Base(path) != "config.json" {
		t.Fatalf("unexpected config path: %s", path)
	}
}

func TestResolveDesktopConfigUsesExplicitPath(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "custom.json")
	path, err := resolveDesktopConfig([]string{"desktop", explicit}, "")
	if err != nil {
		t.Fatalf("resolve explicit config: %v", err)
	}
	if path != explicit {
		t.Fatalf("got %s, want %s", path, explicit)
	}
}
