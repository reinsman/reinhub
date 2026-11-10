package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf.toml")
	content := []byte(`
listen = ":9010"
key = "secret"

[database]
db = "example"
user = "app"
password = "password"
endpoint = "localhost:3306"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":9010" {
		t.Fatalf("Listen = %q, want %q", cfg.Listen, ":9010")
	}
	if cfg.LogLv != "info" {
		t.Fatalf("LogLv = %q, want %q", cfg.LogLv, "info")
	}
	if cfg.Directory != "~/.reinhub" {
		t.Fatalf("Directory = %q, want ~/.reinhub", cfg.Directory)
	}
	if cfg.Db.Endpoint != "localhost:3306" {
		t.Fatalf("Db.Endpoint = %q, want %q", cfg.Db.Endpoint, "localhost:3306")
	}
}

func TestLoadRejectsRemovedModeConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf.toml")
	if err := os.WriteFile(path, []byte("[agent_development]\nenabled=true\ndirectory=\"old-data\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "move directory to the top level") {
		t.Fatalf("old configuration must not silently select a different data directory: %v", err)
	}
}

func TestLoadDoesNotShareState(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.toml")
	secondPath := filepath.Join(dir, "second.toml")
	if err := os.WriteFile(firstPath, []byte(`listen = ":9010"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte(`listen = ":9020"`), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := Load(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(secondPath)
	if err != nil {
		t.Fatal(err)
	}

	if first.Listen != ":9010" || second.Listen != ":9020" {
		t.Fatalf("loads share state: first=%q second=%q", first.Listen, second.Listen)
	}
}
