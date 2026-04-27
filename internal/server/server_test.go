package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"obsidianoid/internal/config"
	"obsidianoid/internal/server"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "Hello.md"), []byte("# Hello\nWorld"), 0o644)
	sub := filepath.Join(root, "subdir")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(sub, "Sub.md"), []byte("# Sub"), 0o644)

	threadsDir := filepath.Join(root, "Threads")
	_ = os.MkdirAll(threadsDir, 0o755)
	for i := 1; i <= 4; i++ {
		name := filepath.Join(threadsDir, fmt.Sprintf("Thread%02d.md", i))
		_ = os.WriteFile(name, []byte(fmt.Sprintf("thread %d content", i)), 0o644)
	}

	cfg := &config.Config{
		VaultPath:     root,
		Port:          8989,
		ThreadsFolder: "Threads",
		ThreadCount:   4,
		ThreadStates:  []config.ThreadState{{}, {}, {}, {}},
	}
	os.Setenv("OBSIDIANOID_STATIC", filepath.Join("..", "..", "static"))
	h := server.New(cfg)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, root
}

func TestAPITree(t *testing.T) {
	t.Run("GET /api/tree returns JSON tree", func(t *testing.T) {
		ts, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/tree")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var node map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if node["is_dir"] != true {
			t.Error("root should be a directory")
		}
	})
}

func TestAPIGetNote(t *testing.T) {
	t.Run("GET /api/note returns note content", func(t *testing.T) {
		ts, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/note?path=Hello.md")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestAPIGetNoteMissing(t *testing.T) {
	t.Run("GET /api/note 404 for missing note", func(t *testing.T) {
		ts, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/note?path=nope.md")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})
}

func TestAPIPutNote(t *testing.T) {
	t.Run("PUT /api/note saves content", func(t *testing.T) {
		ts, root := newTestServer(t)
		body := []byte("# Updated\nnew content")
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/note?path=Hello.md", bytes.NewReader(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
		saved, _ := os.ReadFile(filepath.Join(root, "Hello.md"))
		if string(saved) != string(body) {
			t.Errorf("saved content mismatch")
		}
	})
}

func TestAPIRender(t *testing.T) {
	t.Run("POST /api/render returns HTML", func(t *testing.T) {
		ts, _ := newTestServer(t)
		body := bytes.NewBufferString("# Hello\n\n**bold** text")
		resp, err := http.Post(ts.URL+"/api/render", "text/plain", body)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if ct == "" {
			t.Error("expected Content-Type header")
		}
	})
}

func TestAPINoteMissingPath(t *testing.T) {
	t.Run("GET /api/note without path returns 400", func(t *testing.T) {
		ts, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/note")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

func TestAPIGetThreads(t *testing.T) {
	t.Run("GET /api/threads returns JSON array", func(t *testing.T) {
		ts, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/threads")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var result []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(result) != 4 {
			t.Errorf("expected 4 threads, got %d", len(result))
		}
		if result[0]["content"] != "thread 1 content" {
			t.Errorf("unexpected content: %v", result[0]["content"])
		}
	})
}

func TestAPIPutThreads(t *testing.T) {
	t.Run("PUT /api/threads writes content and returns 204", func(t *testing.T) {
		ts, root := newTestServer(t)
		payload := `[
			{"content":"updated 1","disabled":false},
			{"content":"updated 2","disabled":true},
			{"content":"updated 3","disabled":false},
			{"content":"updated 4","disabled":false}
		]`
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/threads", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
		data, _ := os.ReadFile(filepath.Join(root, "Threads", "Thread01.md"))
		if string(data) != "updated 1" {
			t.Errorf("Thread01.md: got %q", string(data))
		}
	})

	t.Run("PUT /api/threads with wrong count returns 400", func(t *testing.T) {
		ts, _ := newTestServer(t)
		payload := `[{"content":"only one","disabled":false}]`
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/threads", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

func TestAPIEvents(t *testing.T) {
	t.Run("GET /api/events returns text/event-stream", func(t *testing.T) {
		ts, _ := newTestServer(t)
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
		resp, err := http.DefaultClient.Do(req)
		// A context-deadline error is expected — the connection stays open until we cancel.
		if err != nil && ctx.Err() == nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "text/event-stream") {
				t.Errorf("expected text/event-stream, got %q", ct)
			}
		}
	})
}
