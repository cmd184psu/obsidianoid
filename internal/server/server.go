package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"obsidianoid/internal/config"
	"obsidianoid/internal/vault"
	"github.com/russross/blackfriday/v2"
)

func New(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

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
