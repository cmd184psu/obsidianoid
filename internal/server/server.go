package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"obsidianoid/internal/config"
	"obsidianoid/internal/git"
	"obsidianoid/internal/threads"
	"obsidianoid/internal/vault"
	"github.com/russross/blackfriday/v2"
)

func New(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	broker := newEventBroker()
	startVaultWatcher(cfg.VaultPath, broker)
	mux.HandleFunc("/api/events", broker.serveSSE)


	mux.HandleFunc("/api/tree", func(w http.ResponseWriter, r *http.Request) {
		tree, err := vault.Tree(cfg.VaultPath)
		if err != nil {
			http.Error(w, "failed to list vault", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tree)
	})

	mux.HandleFunc("/api/note", func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("path")
		if rel == "" {
			http.Error(w, "path required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			content, err := vault.ReadNote(cfg.VaultPath, rel)
			if err != nil {
				if os.IsNotExist(err) || os.IsPermission(err) {
					http.Error(w, "note not found", http.StatusNotFound)
				} else {
					http.Error(w, "read error", http.StatusInternalServerError)
				}
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(content)

		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read body failed", http.StatusBadRequest)
				return
			}
			if err := vault.WriteNote(cfg.VaultPath, rel, body); err != nil {
				if os.IsPermission(err) {
					http.Error(w, "forbidden", http.StatusForbidden)
				} else {
					http.Error(w, "write error", http.StatusInternalServerError)
				}
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/render", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		flags := blackfriday.CommonExtensions |
			blackfriday.AutoHeadingIDs |
			blackfriday.Tables |
			blackfriday.FencedCode |
			blackfriday.Strikethrough
		renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
			Flags: blackfriday.CommonHTMLFlags,
		})
		html := blackfriday.Run(body, blackfriday.WithExtensions(flags), blackfriday.WithRenderer(renderer))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	})

	mux.HandleFunc("/api/threads", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ts, err := threads.ReadAll(cfg.VaultPath, cfg)
			if err != nil {
				http.Error(w, "failed to read threads", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ts)

		case http.MethodPut:
			var incoming []threads.Thread
			if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if len(incoming) != cfg.ThreadCount {
				http.Error(w, "wrong thread count", http.StatusBadRequest)
				return
			}
			if err := threads.WriteAll(cfg.VaultPath, cfg.ThreadsFolder, incoming); err != nil {
				http.Error(w, "write error", http.StatusInternalServerError)
				return
			}
			for i, t := range incoming {
				cfg.ThreadStates[i].Disabled = t.Disabled
			}
			if cfg.ConfigPath != "" {
				_ = config.Save(cfg.ConfigPath, cfg)
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/git/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"available":%v}`, git.IsAvailable(cfg.VaultPath))
	})

	mux.HandleFunc("/api/git/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !git.IsAvailable(cfg.VaultPath) {
			http.Error(w, "git not available", http.StatusNotFound)
			return
		}
		var body struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if body.Message == "" {
			body.Message = "obsidianoid sync"
		}
		type result struct {
			OK     bool   `json:"ok"`
			Output string `json:"output"`
		}
		output, err := git.Sync(cfg.VaultPath, body.Message)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(result{OK: false, Output: output})
			return
		}
		_ = json.NewEncoder(w).Encode(result{OK: true, Output: output})
	})

	mux.HandleFunc("/api/custom-css", func(w http.ResponseWriter, r *http.Request) {
		if cfg.CustomCSS == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		data, err := os.ReadFile(cfg.CustomCSS)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write(data)
	})

	staticFS := http.FileServer(http.Dir(staticDir()))
	mux.Handle("/", staticFS)

	return mux
}

func staticDir() string {
	if d := os.Getenv("OBSIDIANOID_STATIC"); d != "" {
		return d
	}
	return "/opt/obsidianoid/static"
}

func Run(cfg *config.Config, handler http.Handler) error {
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("\U0001f7e2 obsidianoid listening on https://localhost%s\n", addr)
	return http.ListenAndServeTLS(addr, cfg.CertFile, cfg.KeyFile, handler)
}

func RunInsecure(cfg *config.Config, handler http.Handler) error {
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("\U0001f7e1 obsidianoid (insecure) listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, handler)
}

func CertDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".obsidianoid")
}
