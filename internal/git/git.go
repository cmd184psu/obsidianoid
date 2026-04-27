package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsAvailable returns true if vaultPath is the root of a git repository.
// Accepts both a .git directory (normal clone) and a .git file (worktree).
func IsAvailable(vaultPath string) bool {
	_, err := os.Stat(filepath.Join(vaultPath, ".git"))
	return err == nil
}

// Sync stages all changes, commits with message, then pushes.
// "Nothing to commit" is treated as success; push still runs so any
// previously committed but unpushed work is sent.
// Returns combined output of all commands and the first non-ignorable error.
func Sync(vaultPath, message string) (string, error) {
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = vaultPath
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	if out, err := run("add", "-A"); err != nil {
		return fmt.Sprintf("git add: %s", out), err
	}

	commitOut, commitErr := run("commit", "-m", message)
	if commitErr != nil && !strings.Contains(commitOut, "nothing to commit") {
		return fmt.Sprintf("git commit: %s", commitOut), commitErr
	}

	pushOut, pushErr := run("push")
	if pushErr != nil {
		return fmt.Sprintf("git push: %s", pushOut), pushErr
	}

	return commitOut, nil
}
