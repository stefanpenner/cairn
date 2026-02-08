package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// InitRepo sets the remote for the data directory's git repo.
// Git init is handled by store.initGit(); this only configures the remote.
func InitRepo(dir string, remote string) error {
	// Ensure it's a git repo
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository — open cairn once first to initialize")
	}

	if remote == "" {
		fmt.Println("No remote specified. Use --remote <url> to set one.")
		return nil
	}

	// Remove existing origin first (ignore error if doesn't exist)
	exec.Command("git", "-C", dir, "remote", "remove", "origin").Run()

	cmd := exec.Command("git", "-C", dir, "remote", "add", "origin", remote)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setting remote: %w", err)
	}
	fmt.Printf("Remote set to: %s\n", remote)
	return nil
}

// SyncRepo synchronizes the data directory with the remote.
// Strategy: commit local changes, rebase, fallback to merge, push.
func SyncRepo(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository. Run 'cairn init' first")
	}

	git := func(args ...string) *exec.Cmd {
		return exec.Command("git", append([]string{"-C", dir}, args...)...)
	}

	// 1. Stage and commit any uncommitted local changes
	fmt.Println("Staging changes...")
	git("add", "-A").Run()
	if err := git("diff", "--cached", "--quiet").Run(); err != nil {
		msg := "sync " + time.Now().Format("2006-01-02 15:04:05")
		cmd := git("commit", "-m", msg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	// 2. Try pull --rebase
	fmt.Println("Pulling...")
	rebaseCmd := git("pull", "--rebase")
	rebaseCmd.Stdout = os.Stdout
	rebaseCmd.Stderr = os.Stderr
	if err := rebaseCmd.Run(); err != nil {
		// 3. Rebase failed — abort and try merge
		fmt.Println("Rebase failed, trying merge...")
		git("rebase", "--abort").Run()

		mergeCmd := git("pull", "--no-rebase")
		mergeCmd.Stdout = os.Stdout
		mergeCmd.Stderr = os.Stderr
		if err := mergeCmd.Run(); err != nil {
			// 4. Merge also failed — abort and report
			git("merge", "--abort").Run()
			return fmt.Errorf("sync failed: could not rebase or merge. Resolve conflicts manually")
		}
	}

	// 5. Push
	fmt.Println("Pushing...")
	pushCmd := git("push")
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Println("Sync complete.")
	return nil
}
