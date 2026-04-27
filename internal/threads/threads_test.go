package threads_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"obsidianoid/internal/config"
	"obsidianoid/internal/threads"
)

func threadFile(i int) string { return fmt.Sprintf("Thread%02d.md", i) }

func makeThreadFiles(t *testing.T, root string, contents []string) {
	t.Helper()
	dir := filepath.Join(root, "Threads")
	_ = os.MkdirAll(dir, 0o755)
	for i, c := range contents {
		_ = os.WriteFile(filepath.Join(dir, threadFile(i+1)), []byte(c), 0o644)
	}
}

func baseConfig(root string) *config.Config {
	return &config.Config{
		VaultPath:     root,
		ThreadsFolder: "Threads",
		ThreadCount:   4,
		ThreadStates:  []config.ThreadState{{}, {}, {Disabled: true}, {}},
	}
}

func TestReadAll(t *testing.T) {
	t.Run("reads content and merges disabled state", func(t *testing.T) {
		root := t.TempDir()
		makeThreadFiles(t, root, []string{"alpha", "beta", "gamma", "delta"})
		cfg := baseConfig(root)

		ts, err := threads.ReadAll(root, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ts) != 4 {
			t.Fatalf("expected 4 threads, got %d", len(ts))
		}
		if ts[0].Content != "alpha" {
			t.Errorf("thread 1 content: got %q, want %q", ts[0].Content, "alpha")
		}
		if ts[2].Disabled != true {
			t.Error("thread 3 should be disabled per config")
		}
		if ts[0].Disabled {
			t.Error("thread 1 should not be disabled")
		}
	})

	t.Run("missing file returns empty content without error", func(t *testing.T) {
		root := t.TempDir()
		_ = os.MkdirAll(filepath.Join(root, "Threads"), 0o755)
		cfg := &config.Config{
			VaultPath:     root,
			ThreadsFolder: "Threads",
			ThreadCount:   4,
			ThreadStates:  []config.ThreadState{{}, {}, {}, {}},
		}
		ts, err := threads.ReadAll(root, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, th := range ts {
			if th.Content != "" {
				t.Errorf("thread %d: expected empty content, got %q", i+1, th.Content)
			}
		}
	})
}

func TestWriteAll(t *testing.T) {
	t.Run("writes content to correctly named files", func(t *testing.T) {
		root := t.TempDir()
		input := []threads.Thread{
			{Content: "one"},
			{Content: "two"},
			{Content: "three"},
			{Content: "four"},
		}
		if err := threads.WriteAll(root, "Threads", input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, want := range []string{"one", "two", "three", "four"} {
			path := filepath.Join(root, "Threads", threadFile(i+1))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("thread %d file missing: %v", i+1, err)
			}
			if string(data) != want {
				t.Errorf("thread %d: got %q, want %q", i+1, string(data), want)
			}
		}
	})

	t.Run("creates folder if absent", func(t *testing.T) {
		root := t.TempDir()
		input := []threads.Thread{{Content: "a"}, {Content: "b"}, {Content: "c"}, {Content: "d"}}
		if err := threads.WriteAll(root, "Threads", input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "Threads")); os.IsNotExist(err) {
			t.Error("Threads directory was not created")
		}
	})
}
