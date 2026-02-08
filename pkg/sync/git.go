package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// InitRepo initializes the data directory as a git repo.
func InitRepo(dir string, remote string) error {
	// Check if already a git repo
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		fmt.Println("Already a git repository.")
	} else {
		cmd := exec.Command("git", "init", dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}
	}

	// Create .gitignore if it doesn't exist
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		content := "# Cairn data\n*.swp\n*.swo\n*~\n.DS_Store\n"
		if err := os.WriteFile(gitignore, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing .gitignore: %w", err)
		}
	}

	// Set remote if provided
	if remote != "" {
		// Remove existing origin first (ignore error if doesn't exist)
		exec.Command("git", "-C", dir, "remote", "remove", "origin").Run()

		cmd := exec.Command("git", "-C", dir, "remote", "add", "origin", remote)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setting remote: %w", err)
		}
		fmt.Printf("Remote set to: %s\n", remote)
	}

	// Initial commit if no commits yet
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	if err := cmd.Run(); err != nil {
		// No commits yet
		exec.Command("git", "-C", dir, "add", "-A").Run()
		exec.Command("git", "-C", dir, "commit", "-m", "Initial cairn data").Run()
	}

	fmt.Printf("Cairn data initialized at: %s\n", dir)
	return nil
}

// SyncRepo performs add, commit, pull --rebase, push.
func SyncRepo(dir string) error {
	// Check if it's a git repo
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository. Run 'cairn init' first")
	}

	steps := []struct {
		desc string
		args []string
		skip bool // if true, don't error on failure
	}{
		{"Staging changes", []string{"add", "-A"}, false},
		{"Committing", []string{"commit", "-m", "sync " + time.Now().Format("2006-01-02 15:04:05")}, true},
		{"Pulling", []string{"pull", "--rebase"}, true},
		{"Pushing", []string{"push"}, true},
	}

	for _, step := range steps {
		fmt.Printf("%s...\n", step.desc)
		args := append([]string{"-C", dir}, step.args...)
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if !step.skip {
				return fmt.Errorf("%s failed: %w", step.desc, err)
			}
		}
	}

	fmt.Println("Sync complete.")
	return nil
}
