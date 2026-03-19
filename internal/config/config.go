// internal/config/config.go
package config

import (
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
	cfg.path = path
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
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hm", "config.toml"), nil
}
