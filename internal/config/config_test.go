// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"hm/internal/config"
)

func TestLoad_CreatesDefaultsOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClipboardCmd != "xclip -selection clipboard" {
		t.Errorf("ClipboardCmd = %q, want default", cfg.ClipboardCmd)
	}
	if cfg.SystemPrompt == "" {
		t.Error("SystemPrompt is empty, want default")
	}
	if cfg.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", cfg.SessionID)
	}
	// File must exist after Load
	if _, err := os.Stat(filepath.Join(dir, ".config", "hm", "config.toml")); os.IsNotExist(err) {
		t.Error("config.toml was not created on first run")
	}
}

func TestLoad_ReadsExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfgDir := filepath.Join(dir, ".config", "hm")
	os.MkdirAll(cfgDir, 0755)
	content := `clipboard_cmd = "pbcopy"
system_prompt = "custom prompt"
session_id = "test-uuid-1234"
`
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0644)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClipboardCmd != "pbcopy" {
		t.Errorf("ClipboardCmd = %q, want pbcopy", cfg.ClipboardCmd)
	}
	if cfg.SessionID != "test-uuid-1234" {
		t.Errorf("SessionID = %q, want test-uuid-1234", cfg.SessionID)
	}
}

func TestSaveSessionID_PersistsAcrossLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.SaveSessionID("new-session-id"); err != nil {
		t.Fatalf("SaveSessionID() error = %v", err)
	}

	cfg2, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if cfg2.SessionID != "new-session-id" {
		t.Errorf("SessionID after save = %q, want new-session-id", cfg2.SessionID)
	}
}

func TestClearSessionID_RemovesID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.SaveSessionID("some-id"); err != nil {
		t.Fatalf("SaveSessionID() error = %v", err)
	}

	cfg2, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if err := cfg2.ClearSessionID(); err != nil {
		t.Fatalf("ClearSessionID() error = %v", err)
	}

	cfg3, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if cfg3.SessionID != "" {
		t.Errorf("SessionID after clear = %q, want empty", cfg3.SessionID)
	}
}

func TestLoad_PreservesOtherFieldsOnSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfgDir := filepath.Join(dir, ".config", "hm")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`clipboard_cmd = "pbcopy"
system_prompt = "custom"
session_id = ""
`), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.SaveSessionID("abc-123"); err != nil {
		t.Fatalf("SaveSessionID() error = %v", err)
	}

	cfg2, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if cfg2.ClipboardCmd != "pbcopy" {
		t.Errorf("ClipboardCmd changed after SaveSessionID: got %q", cfg2.ClipboardCmd)
	}
	if cfg2.SystemPrompt != "custom" {
		t.Errorf("SystemPrompt changed after SaveSessionID: got %q", cfg2.SystemPrompt)
	}
}
