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
	t.Run("fails without vault_path or vaults", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{}`), 0o600)
		_, err := config.Load(path)
		if err == nil {
			t.Fatal("expected error for missing vaults")
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

func TestLoadMultiVault(t *testing.T) {
	t.Run("loads multi-vault config", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		raw := `{"vaults":[{"path":"/opt/Vaults/Work","name":"Work","theme":"dark"},{"path":"/opt/Vaults/Personal","name":"Personal","theme":"forest"}]}`
		_ = os.WriteFile(path, []byte(raw), 0o600)
		c, err := config.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(c.Vaults) != 2 {
			t.Fatalf("expected 2 vaults, got %d", len(c.Vaults))
		}
		if c.Vaults[0].Path != "/opt/Vaults/Work" {
			t.Errorf("vault 0 path: got %s", c.Vaults[0].Path)
		}
		if c.Vaults[0].Name != "Work" {
			t.Errorf("vault 0 name: got %s", c.Vaults[0].Name)
		}
		if c.Vaults[0].Theme != "dark" {
			t.Errorf("vault 0 theme: got %s", c.Vaults[0].Theme)
		}
		if c.Vaults[1].Path != "/opt/Vaults/Personal" {
			t.Errorf("vault 1 path: got %s", c.Vaults[1].Path)
		}
		if c.Vaults[1].Theme != "forest" {
			t.Errorf("vault 1 theme: got %s", c.Vaults[1].Theme)
		}
	})
}

func TestLegacyVaultPathMigration(t *testing.T) {
	t.Run("migrates vault_path to Vaults[0]", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{"vault_path":"/legacy/vault"}`), 0o600)
		c, err := config.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(c.Vaults) != 1 {
			t.Fatalf("expected 1 vault after migration, got %d", len(c.Vaults))
		}
		if c.Vaults[0].Path != "/legacy/vault" {
			t.Errorf("migrated path: got %s", c.Vaults[0].Path)
		}
		if c.Vaults[0].Name != "Vault" {
			t.Errorf("migrated name: got %s", c.Vaults[0].Name)
		}
		if c.Vaults[0].Theme != "dark" {
			t.Errorf("migrated theme: got %s", c.Vaults[0].Theme)
		}
	})
}

func TestEmptyVaultsRejected(t *testing.T) {
	t.Run("fails with empty vaults array", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		_ = os.WriteFile(path, []byte(`{"vaults":[]}`), 0o600)
		_, err := config.Load(path)
		if err == nil {
			t.Fatal("expected error for empty vaults list")
		}
	})
}

func TestVaultEmptyThemeDefaultsDark(t *testing.T) {
	t.Run("vault with empty theme gets dark default", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "obsidianoid.json")
		raw := `{"vaults":[{"path":"/some/vault","name":"Mine","theme":""}]}`
		_ = os.WriteFile(path, []byte(raw), 0o600)
		c, err := config.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Vaults[0].Theme != "dark" {
			t.Errorf("expected default theme dark, got %s", c.Vaults[0].Theme)
		}
	})
}
