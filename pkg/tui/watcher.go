package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// StartWatcher watches the data directory for changes and sends FileChangedMsg.
func StartWatcher(root string, program *tea.Program) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Walk and add all directories
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden dirs (like .git)
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return nil, err
	}

	done := make(chan struct{})

	go func() {
		var debounceTimer *time.Timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Only care about .md file changes
				if !strings.HasSuffix(event.Name, ".md") {
					continue
				}

				// Debounce: wait 200ms after last change
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
					program.Send(FileChangedMsg{})
				})

				// If a new directory was created, watch it too
				if event.Op&fsnotify.Create != 0 {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
						watcher.Add(event.Name)
					}
				}

			case <-watcher.Errors:
				// Ignore watcher errors silently

			case <-done:
				return
			}
		}
	}()

	cleanup := func() {
		close(done)
		watcher.Close()
	}

	return cleanup, nil
}
