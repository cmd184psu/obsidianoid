package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ThreadState struct {
	Disabled bool `json:"disabled"`
}

// VaultConfig holds the path, display name, and color theme for one vault.
type VaultConfig struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Theme string `json:"theme"`
}

// Config holds obsidianoid runtime configuration.
type Config struct {
	Vaults        []VaultConfig `json:"vaults,omitempty"`
	VaultPath     string        `json:"vault_path,omitempty"` // legacy: migrated to Vaults on load
	CustomCSS     string        `json:"custom_css,omitempty"`
	CertFile      string        `json:"cert_file,omitempty"`
	KeyFile       string        `json:"key_file,omitempty"`
	Port          int           `json:"port,omitempty"`
	ThreadsFolder string        `json:"threads_folder,omitempty"`
	ThreadCount   int           `json:"thread_count,omitempty"`
	ThreadStates  []ThreadState `json:"thread_states,omitempty"`
	ConfigPath    string        `json:"-"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".obsidianoid.json")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if len(c.Vaults) == 0 && c.VaultPath != "" {
		c.Vaults = []VaultConfig{{Path: c.VaultPath, Name: "Vault", Theme: "dark"}}
	}
	if len(c.Vaults) == 0 {
		return nil, errors.New("vaults list is required in config")
	}
	for i := range c.Vaults {
		if c.Vaults[i].Theme == "" {
			c.Vaults[i].Theme = "dark"
		}
	}
	if c.Port == 0 {
		c.Port = 8989
	}
	if c.CertFile == "" {
		home, _ := os.UserHomeDir()
		c.CertFile = filepath.Join(home, ".obsidianoid", "server.crt")
	}
	if c.KeyFile == "" {
		home, _ := os.UserHomeDir()
		c.KeyFile = filepath.Join(home, ".obsidianoid", "server.key")
	}
	if c.ThreadsFolder == "" {
		c.ThreadsFolder = "Threads"
	}
	if c.ThreadCount == 0 {
		c.ThreadCount = 4
	}
	for len(c.ThreadStates) < c.ThreadCount {
		c.ThreadStates = append(c.ThreadStates, ThreadState{})
	}
	c.ThreadStates = c.ThreadStates[:c.ThreadCount]
	c.ConfigPath = path
	return &c, nil
}

func Save(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
