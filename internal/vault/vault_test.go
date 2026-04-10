package vault_test

import (
	"os"
	"path/filepath"
	"testing"

	"obsidianoid/internal/vault"
)

func createVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"Note One.md":          "# Note One\nHello",
		"subdir/Note Two.md":   "# Note Two\nWorld",
		"subdir/Note Three.md": "# Note Three\nFoo",
		".hidden/secret.md":    "should be ignored",
		"image.png":            "binary",
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(abs), 0o755)
		_ = os.WriteFile(abs, []byte(content), 0o644)
	}
	return root
}

func TestListMarkdownFiles(t *testing.T) {
	t.Run("lists only .md files excluding hidden dirs", func(t *testing.T) {
		root := createVault(t)
		notes, err := vault.List(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 3 {
			t.Errorf("expected 3 notes, got %d: %v", len(notes), notes)
		}
	})
}

func TestTreeStructure(t *testing.T) {
	t.Run("tree has correct root and children", func(t *testing.T) {
		root := createVault(t)
		tree, err := vault.Tree(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tree == nil {
			t.Fatal("tree is nil")
		}
		if !tree.IsDir {
			t.Error("root should be a directory")
		}
		if len(tree.Children) == 0 {
			t.Error("root should have children")
		}
	})
}

func TestReadNote(t *testing.T) {
	t.Run("reads note content", func(t *testing.T) {
		root := createVault(t)
		content, err := vault.ReadNote(root, "Note One.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(content) != "# Note One\nHello" {
			t.Errorf("unexpected content: %s", content)
		}
	})
}

func TestReadNotePathTraversal(t *testing.T) {
	t.Run("rejects path traversal", func(t *testing.T) {
		root := createVault(t)
		_, err := vault.ReadNote(root, "../../etc/passwd")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})
}

func TestWriteNote(t *testing.T) {
	t.Run("writes and reads back note", func(t *testing.T) {
		root := createVault(t)
		err := vault.WriteNote(root, "New Note.md", []byte("# New\ncontent"))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		content, err := vault.ReadNote(root, "New Note.md")
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(content) != "# New\ncontent" {
			t.Errorf("unexpected content: %s", content)
		}
	})
}

func TestWriteNotePathTraversal(t *testing.T) {
	t.Run("rejects path traversal on write", func(t *testing.T) {
		root := createVault(t)
		err := vault.WriteNote(root, "../outside.md", []byte("bad"))
		if err == nil {
			t.Error("expected error for path traversal on write")
		}
	})
}
