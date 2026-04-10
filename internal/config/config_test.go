package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"obsidianoid/internal/config"
)

func TestLoadValid(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{"vault_path":"/my/vault"}`), 0o600)
		c, err := config.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.VaultPath != "/my/vault" {
			t.Errorf("expected /my/vault, got %s", c.VaultPath)
		}
		if c.Port != 8989 {
			t.Errorf("expected default port 8989, got %d", c.Port)
		}
	})
}

func TestLoadMissingVaultPath(t *testing.T) {
	t.Run("fails without vault_path", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{}`), 0o600)
		_, err := config.Load(path)
		if err == nil {
			t.Fatal("expected error for missing vault_path")
		}
	})
}

func TestLoadFileNotFound(t *testing.T) {
	t.Run("fails on missing file", func(t *testing.T) {
		_, err := config.Load("/nonexistent/path.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestSaveAndReload(t *testing.T) {
	t.Run("save and reload roundtrip", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		orig := &config.Config{VaultPath: "/test/vault", Port: 8989}
		if err := config.Save(path, orig); err != nil {
			t.Fatalf("save failed: %v", err)
		}
		loaded, err := config.Load(path)
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		if loaded.VaultPath != orig.VaultPath {
			t.Errorf("vault_path mismatch: %s vs %s", loaded.VaultPath, orig.VaultPath)
		}
	})
}

func TestCustomPort(t *testing.T) {
	t.Run("respects custom port", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{"vault_path":"/v","port":9000}`), 0o600)
		c, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if c.Port != 9000 {
			t.Errorf("expected port 9000, got %d", c.Port)
		}
	})
}
