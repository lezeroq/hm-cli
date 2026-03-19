// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	defaultClipboardCmd = "xclip -selection clipboard"
	defaultSystemPrompt = "You are a shell command assistant. Return only the raw shell command, no explanation, no markdown, no code fences."
)

// Config holds all hm configuration. Fields map to config.toml keys.
type Config struct {
	ClipboardCmd string `toml:"clipboard_cmd"`
	SystemPrompt string `toml:"system_prompt"`
	SessionID    string `toml:"session_id"`
	LastCommand  string `toml:"last_command"`
	path         string `toml:"-"`
}

// Load reads ~/.config/hm/config.toml, creating it with defaults if absent.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		ClipboardCmd: defaultClipboardCmd,
		SystemPrompt: defaultSystemPrompt,
		path:         path,
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, cfg.write()
	}
	if err != nil {
		return nil, err
	}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveSessionID persists a new session ID to config.toml.
func (c *Config) SaveSessionID(id string) error {
	c.SessionID = id
	return c.write()
}

// ClearSessionID removes the stored session ID from config.toml.
func (c *Config) ClearSessionID() error {
	c.SessionID = ""
	return c.write()
}

// SaveLastCommand persists the last generated command to config.toml.
func (c *Config) SaveLastCommand(cmd string) error {
	c.LastCommand = cmd
	return c.write()
}

func (c *Config) write() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Write to a temp file then rename for an atomic update — prevents
	// corruption if the process is interrupted mid-write.
	tmp, err := os.CreateTemp(dir, "config-*.toml.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if encErr := toml.NewEncoder(tmp).Encode(c); encErr != nil {
		tmp.Close()
		os.Remove(tmpName)
		return encErr
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, c.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("committing config: %w", err)
	}
	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hm", "config.toml"), nil
}
