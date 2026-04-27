package threads

import (
	"fmt"
	"os"
	"path/filepath"

	"obsidianoid/internal/config"
)

type Thread struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

func fileName(index int) string {
	return fmt.Sprintf("Thread%02d.md", index+1)
}

// ReadAll reads all thread files from the vault and merges disabled state from config.
func ReadAll(vaultPath string, cfg *config.Config) ([]Thread, error) {
	result := make([]Thread, cfg.ThreadCount)
	for i := 0; i < cfg.ThreadCount; i++ {
		path := filepath.Join(vaultPath, cfg.ThreadsFolder, fileName(i))
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("thread %d: %w", i+1, err)
		}
		result[i] = Thread{
			Content:  string(data),
			Disabled: cfg.ThreadStates[i].Disabled,
		}
	}
	return result, nil
}

// WriteAll writes thread content to vault files. Disabled state is not stored in files.
func WriteAll(vaultPath, folder string, threads []Thread) error {
	dir := filepath.Join(vaultPath, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i, t := range threads {
		path := filepath.Join(dir, fileName(i))
		if err := os.WriteFile(path, []byte(t.Content), 0o644); err != nil {
			return fmt.Errorf("thread %d: %w", i+1, err)
		}
	}
	return nil
}
