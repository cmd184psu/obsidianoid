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

// newTestServer creates a test server with two vaults. The first vault
// contains Hello.md + subdir/Sub.md + Threads/. The second vault contains
// only AltNote.md. Returns the server and both vault root paths.
func newTestServer(t *testing.T) (*httptest.Server, string, string) {
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

	root2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(root2, "AltNote.md"), []byte("# Alt\nVault Two"), 0o644)

	cfg := &config.Config{
		Vaults: []config.VaultConfig{
			{Path: root, Name: "Work", Theme: "dark"},
			{Path: root2, Name: "Personal", Theme: "forest"},
		},
		Port:          8989,
		ThreadsFolder: "Threads",
		ThreadCount:   4,
		ThreadStates:  []config.ThreadState{{}, {}, {}, {}},
	}
	os.Setenv("OBSIDIANOID_STATIC", filepath.Join("..", "..", "static"))
	h := server.New(cfg)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, root, root2
}

func TestAPITree(t *testing.T) {
	t.Run("GET /api/tree returns JSON tree for vault 0", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
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

	t.Run("GET /api/tree?vault=1 returns tree for vault 1", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/tree?vault=1")
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
		children, _ := node["children"].([]interface{})
		found := false
		for _, child := range children {
			m, _ := child.(map[string]interface{})
			if m["name"] == "AltNote" {
				found = true
			}
		}
		if !found {
			t.Error("vault 1 tree should contain AltNote")
		}
	})

	t.Run("GET /api/tree?vault=99 falls back to vault 0", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/tree?vault=99")
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
		children, _ := node["children"].([]interface{})
		found := false
		for _, child := range children {
			m, _ := child.(map[string]interface{})
			if m["name"] == "Hello" {
				found = true
			}
		}
		if !found {
			t.Error("vault 0 fallback tree should contain Hello")
		}
	})
}

func TestAPIVaults(t *testing.T) {
	t.Run("GET /api/vaults returns vault names and themes", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/vaults")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var vaults []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&vaults); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(vaults) != 2 {
			t.Fatalf("expected 2 vaults, got %d", len(vaults))
		}
		if vaults[0]["name"] != "Work" {
			t.Errorf("vault 0 name: got %v", vaults[0]["name"])
		}
		if vaults[0]["theme"] != "dark" {
			t.Errorf("vault 0 theme: got %v", vaults[0]["theme"])
		}
		if vaults[1]["name"] != "Personal" {
			t.Errorf("vault 1 name: got %v", vaults[1]["name"])
		}
		if vaults[1]["theme"] != "forest" {
			t.Errorf("vault 1 theme: got %v", vaults[1]["theme"])
		}
		// Paths must not be exposed.
		if _, hasPath := vaults[0]["path"]; hasPath {
			t.Error("vault response must not include path")
		}
	})
}

func TestAPIGetNote(t *testing.T) {
	t.Run("GET /api/note returns note content from vault 0", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/note?path=Hello.md")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GET /api/note returns note from vault 1", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/note?vault=1&path=AltNote.md")
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
		ts, _, _ := newTestServer(t)
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
	t.Run("PUT /api/note saves content to vault 0", func(t *testing.T) {
		ts, root, _ := newTestServer(t)
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

	t.Run("PUT /api/note saves content to vault 1", func(t *testing.T) {
		ts, _, root2 := newTestServer(t)
		body := []byte("# Alt Updated")
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/note?vault=1&path=AltNote.md", bytes.NewReader(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
		saved, _ := os.ReadFile(filepath.Join(root2, "AltNote.md"))
		if string(saved) != string(body) {
			t.Errorf("saved content mismatch: got %q", string(saved))
		}
	})
}

func TestAPIRender(t *testing.T) {
	t.Run("POST /api/render returns HTML", func(t *testing.T) {
		ts, _, _ := newTestServer(t)
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
		ts, _, _ := newTestServer(t)
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
		ts, _, _ := newTestServer(t)
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
		ts, root, _ := newTestServer(t)
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
		ts, _, _ := newTestServer(t)
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
		ts, _, _ := newTestServer(t)
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
