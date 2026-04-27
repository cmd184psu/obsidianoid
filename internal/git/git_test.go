package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"obsidianoid/internal/git"
)

// ── IsAvailable ──────────────────────────────────────────────────────────────

func TestIsAvailable(t *testing.T) {
	t.Run("true when .git directory present", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.Mkdir(filepath.Join(dir, ".git"), 0o755)
		if !git.IsAvailable(dir) {
			t.Error("expected true")
		}
	})

	t.Run("true when .git file present (worktree)", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../main/.git/worktrees/x"), 0o644)
		if !git.IsAvailable(dir) {
			t.Error("expected true")
		}
	})

	t.Run("false when .git absent", func(t *testing.T) {
		dir := t.TempDir()
		if git.IsAvailable(dir) {
			t.Error("expected false")
		}
	})
}

// ── Sync ─────────────────────────────────────────────────────────────────────

// mustGit runs a git command in dir, failing the test on error.
func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// defaultBranch returns the current HEAD branch name.
func defaultBranch(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(out))
}

func TestSync(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	// Bare remote so push has somewhere to go.
	remote := t.TempDir()
	mustGit(t, remote, "init", "--bare")

	// Working repo.
	work := t.TempDir()
	mustGit(t, work, "init")
	mustGit(t, work, "config", "user.email", "test@obsidianoid.test")
	mustGit(t, work, "config", "user.name", "Obsidianoid Test")
	mustGit(t, work, "remote", "add", "origin", remote)

	// Seed an initial commit and push so the branch tracking is set up.
	_ = os.WriteFile(filepath.Join(work, "README.md"), []byte("# vault"), 0o644)
	mustGit(t, work, "add", "-A")
	mustGit(t, work, "commit", "-m", "init")
	branch := defaultBranch(t, work)
	mustGit(t, work, "push", "--set-upstream", "origin", branch)

	t.Run("stages, commits and pushes a new file", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(work, "Note.md"), []byte("# Hello"), 0o644)
		out, err := git.Sync(work, "add note")
		if err != nil {
			t.Fatalf("Sync error: %v\nOutput: %s", err, out)
		}
		// Confirm commit message appears in log.
		cmd := exec.Command("git", "log", "--oneline", "-1")
		cmd.Dir = work
		log, _ := cmd.Output()
		if !strings.Contains(string(log), "add note") {
			t.Errorf("expected 'add note' in log, got: %s", log)
		}
	})

	t.Run("nothing to commit is not an error", func(t *testing.T) {
		// Working tree is clean; Sync should succeed (push is a no-op too).
		_, err := git.Sync(work, "empty sync")
		if err != nil {
			t.Fatalf("expected no error on clean tree, got: %v", err)
		}
	})
}
