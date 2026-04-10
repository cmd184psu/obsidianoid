package vault

import (
	"os"
	"path/filepath"
	"strings"
)

type Note struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	AbsPath string `json:"-"`
}

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path,omitempty"`
	IsDir    bool        `json:"is_dir"`
	Children []*TreeNode `json:"children,omitempty"`
}

func List(root string) ([]Note, error) {
	var notes []Note
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		notes = append(notes, Note{
			Path:    rel,
			Name:    strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
			AbsPath: path,
		})
		return nil
	})
	return notes, err
}

func Tree(root string) (*TreeNode, error) {
	notes, err := List(root)
	if err != nil {
		return nil, err
	}
	rootNode := &TreeNode{Name: filepath.Base(root), IsDir: true}
	for _, n := range notes {
		insert(rootNode, strings.Split(n.Path, "/"), n.Path)
	}
	return rootNode, nil
}

func insert(parent *TreeNode, parts []string, fullPath string) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		parent.Children = append(parent.Children, &TreeNode{
			Name: strings.TrimSuffix(parts[0], ".md"),
			Path: fullPath,
		})
		return
	}
	for _, child := range parent.Children {
		if child.IsDir && child.Name == parts[0] {
			insert(child, parts[1:], fullPath)
			return
		}
	}
	dir := &TreeNode{Name: parts[0], IsDir: true}
	parent.Children = append(parent.Children, dir)
	insert(dir, parts[1:], fullPath)
}

func ReadNote(root, relPath string) ([]byte, error) {
	clean := filepath.Join(root, filepath.FromSlash(relPath))
	if !strings.HasPrefix(clean, filepath.Clean(root)+string(os.PathSeparator)) &&
		clean != filepath.Clean(root) {
		return nil, os.ErrPermission
	}
	return os.ReadFile(clean)
}

func WriteNote(root, relPath string, content []byte) error {
	clean := filepath.Join(root, filepath.FromSlash(relPath))
	if !strings.HasPrefix(clean, filepath.Clean(root)+string(os.PathSeparator)) &&
		clean != filepath.Clean(root) {
		return os.ErrPermission
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return err
	}
	return os.WriteFile(clean, content, 0o644)
}
