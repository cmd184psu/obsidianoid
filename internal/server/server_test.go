package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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

	cfg := &config.Config{VaultPath: root, Port: 8989}
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
